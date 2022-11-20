package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync/atomic"

	"golang.org/x/sync/errgroup"

	"github.com/GDVFox/ctxio"
	"github.com/GDVFox/gostreaming/runtime/config"
	"github.com/GDVFox/gostreaming/runtime/logs"
	upstreambackup "github.com/GDVFox/gostreaming/runtime/upstream_backup"
)

// Runtime обертка над действием.
type Runtime struct {
	path      string
	isRunning uint32
	isSource  bool
	isSink    bool

	receiver  *upstreambackup.DefaiultReceiver
	forwarder upstreambackup.Forwarder
	opt       *config.ActionOptions

	messagesQueue chan *upstreambackup.UpstreamMessage
}

// NewRuntime создает новый объект Runtime.
func NewRuntime(path string, isSource, isSink bool, in *upstreambackup.DefaiultReceiver, out upstreambackup.Forwarder, opt *config.ActionOptions) *Runtime {
	return &Runtime{
		path:          path,
		isSource:      isSource,
		isSink:        isSink,
		isRunning:     0,
		receiver:      in,
		forwarder:     out,
		opt:           opt,
		messagesQueue: make(chan *upstreambackup.UpstreamMessage, 1),
	}
}

// Run запускает действие.
func (a *Runtime) Run(ctx context.Context) error {
	wg, runCtx := errgroup.WithContext(ctx)
	runActionCommand := exec.CommandContext(runCtx, a.path, a.opt.Args...)
	runActionCommand.Env = os.Environ()
	runActionCommand.Env = append(runActionCommand.Env, a.opt.EnvAsSlice()...)

	inCmd, err := runActionCommand.StdinPipe()
	if err != nil {
		return fmt.Errorf("can not get stdin pipe: %w", err)
	}

	outCmd, err := runActionCommand.StdoutPipe()
	if err != nil {
		return fmt.Errorf("can not get stdout pipe: %w", err)
	}

	errCmd, err := runActionCommand.StderrPipe()
	if err != nil {
		return fmt.Errorf("can not get stderr pipe: %w", err)
	}

	// Выставляем флаг запуска, так как следующие операции будут асинхронно все запускать.
	atomic.StoreUint32(&a.isRunning, 1)
	defer atomic.StoreUint32(&a.isRunning, 0)

	// Запускаем команду асинхронно, так как используем StdoutPipe, StderrPipe.
	// Когда команда завершиться, то StdoutPipe, StderrPipe будут закрыты автоматически.
	// https://golang.org/pkg/os/exec/#Cmd.StdoutPipe
	if err := runActionCommand.Start(); err != nil {
		return fmt.Errorf("can not start action: %w", err)
	}
	logs.Logger.Infof("action started with command: %s", runActionCommand.String())

	wg.Go(func() error {
		return a.handleOut(runCtx, outCmd)
	})
	wg.Go(func() error {
		return a.handleErr(runCtx, errCmd)
	})
	wg.Go(func() error {
		return a.handleIn(runCtx, inCmd)
	})
	wg.Go(func() error {
		return a.handleAcks(runCtx)
	})
	wg.Go(func() error {
		return a.forwarder.Run(runCtx)
	})
	wg.Go(func() error {
		return a.receiver.Run(runCtx)
	})
	if err := wg.Wait(); err != nil {
		return fmt.Errorf("action io got error: %w", err)
	}

	// Если input/output завершился, то ждем окончания работы действия.
	if err := runActionCommand.Wait(); err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			return err
		}
		if exitErr.Success() || exitErr.ExitCode() == -1 {
			return nil
		}
		return exitErr
	}

	return nil
}

// IsRunning возвращает true, если действие сейчас работает и false иначе.
func (a *Runtime) IsRunning() bool {
	return atomic.LoadUint32(&a.isRunning) == 1
}

// ChangeOut заменяет отправку в oldOut на отправку в newOut.
func (a *Runtime) ChangeOut(oldOut, newOut string) error {
	return a.forwarder.ChangeOut(oldOut, newOut)
}

// GetOldestOutput возвращает самый старый output_message_id, который хранится в логе.
func (a *Runtime) GetOldestOutput() (uint32, error) {
	return a.forwarder.GetOldestOutput()
}

// inCmd закроет handleIn, так как он писатель и может это делать по
// https://golang.org/pkg/os/exec/#Cmd.StdinPipe
func (a *Runtime) handleIn(ctx context.Context, cmdIn io.WriteCloser) error {
	defer logs.Logger.Debug("runtime: handle STDIN stopped")

	cmdWriter := ctxio.NewContextWriter(ctx, cmdIn)
	defer cmdWriter.Close()
	defer close(a.messagesQueue)

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-a.receiver.Messages():
			if msg == nil && !ok {
				return nil
			}

			logs.Logger.Debugf("runtime: got input data from input %d with number %d", msg.InputID, msg.Header.MessageID)

			a.messagesQueue <- msg
			if err := binary.Write(cmdWriter, binary.BigEndian, msg.Header.MessageLength); err != nil {
				return fmt.Errorf("can not write message length: %w", err)
			}

			if err := binary.Write(cmdWriter, binary.BigEndian, msg.Data); err != nil {
				return fmt.Errorf("can not write message data: %w", err)
			}
		}
	}
}

func (a *Runtime) handleErr(ctx context.Context, cmdErr io.Reader) error {
	defer logs.Logger.Debug("runtime: handle STDERR stopped")

	errData := make([]byte, 4096)
	for {
		n, err := cmdErr.Read(errData)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("can not read stderr data: %w", err)
		}
		logs.Logger.Errorf("got error from action: %s", string(errData[:n]))
	}
}

func (a *Runtime) handleOut(ctx context.Context, cmdOut io.Reader) error {
	defer logs.Logger.Debug("runtime: handle STDOUT stopped")

	for {
		logs.Logger.Debug("runtime: waiting for new command out")
		// Сообщение, из которого будет получено ожидаемый выход.
		inputMsg := upstreambackup.DummyUpstreamMessage
		if !a.isSource {
			inputMsg = <-a.messagesQueue
		}
		// В этом месте ждем, что при отключении писатель, т.е. действие,
		// закроет io.Reader и разблокирует нас.
		messsageLength := uint32(0)
		if err := binary.Read(cmdOut, binary.BigEndian, &messsageLength); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("can not read message length: %w", err)
		}

		if a.isSink {
			if _, err := io.CopyN(io.Discard, cmdOut, int64(messsageLength)); err != nil {
				return fmt.Errorf("can not discard cmd out of sink: %w", err)
			}

			if err := a.forwarder.Forward(inputMsg.InputID, inputMsg.Header.MessageID, nil); err != nil {
				return fmt.Errorf("can not forward fake message: %w", err)
			}
		} else {
			// Таким образом действие дает понять, что сообщение пропускается.
			if messsageLength == 0 {
				continue
			}

			data := make([]byte, messsageLength)
			if err := binary.Read(cmdOut, binary.BigEndian, data); err != nil {
				return fmt.Errorf("can not read message data: %w", err)
			}

			if err := a.forwarder.Forward(inputMsg.InputID, inputMsg.Header.MessageID, data); err != nil {
				return fmt.Errorf("can not forward message: %w", err)
			}
		}

		logs.Logger.Debugf("runtime: sended output data")
	}
}

func (a *Runtime) handleAcks(ctx context.Context) error {
	acks := a.receiver.Acks()
	defer close(acks)

	for {
		select {
		case <-ctx.Done():
			return nil
		case ack, ok := <-a.forwarder.AckMessages():
			if ack == nil && !ok {
				return nil
			}

			select {
			case <-ctx.Done():
				return nil
			case acks <- ack:
			}
		}
	}
}
