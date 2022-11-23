package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"sync/atomic"

	"golang.org/x/sync/errgroup"

	"github.com/GDVFox/ctxio"
	"github.com/GDVFox/gostreaming/runtime/config"
	upstreambackup "github.com/GDVFox/gostreaming/runtime/upstream_backup"
	"github.com/GDVFox/gostreaming/util"
)

// Runtime обертка над действием.
type Runtime struct {
	path      string
	isRunning uint32
	isSource  bool

	receiver  *upstreambackup.DefaultReceiver
	forwarder *upstreambackup.DefaultForwarder
	opt       *config.ActionOptions

	messagesQueue chan *upstreambackup.UpstreamMessage
	logger        *util.Logger
}

// NewRuntime создает новый объект Runtime.
func NewRuntime(path string, isSource bool, in *upstreambackup.DefaultReceiver, out *upstreambackup.DefaultForwarder, opt *config.ActionOptions, l *util.Logger) *Runtime {
	return &Runtime{
		path:          path,
		isSource:      isSource,
		isRunning:     0,
		receiver:      in,
		forwarder:     out,
		opt:           opt,
		messagesQueue: make(chan *upstreambackup.UpstreamMessage, 1),
		logger:        l.WithName("runtime"),
	}
}

// Run запускает действие.
func (r *Runtime) Run(ctx context.Context) error {
	defer r.logger.Info("runtime stopped")

	cancelableCtx, runtimeCancel := context.WithCancel(ctx)
	defer runtimeCancel()

	wg, runCtx := errgroup.WithContext(cancelableCtx)
	runActionCommand := exec.CommandContext(runCtx, r.path, r.opt.Args...)
	runActionCommand.Env = os.Environ()
	runActionCommand.Env = append(runActionCommand.Env, r.opt.EnvAsSlice()...)

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
	atomic.StoreUint32(&r.isRunning, 1)
	defer atomic.StoreUint32(&r.isRunning, 0)

	// Запускаем команду асинхронно, так как используем StdoutPipe, StderrPipe.
	// Когда команда завершиться, то StdoutPipe, StderrPipe будут закрыты автоматически.
	// https://golang.org/pkg/os/exec/#Cmd.StdoutPipe
	if err := runActionCommand.Start(); err != nil {
		return fmt.Errorf("can not start action: %w", err)
	}
	r.logger.Infof("action started with command: %s", runActionCommand.String())

	wg.Go(func() error {
		defer runtimeCancel()
		return r.handleOut(runCtx, outCmd)
	})
	wg.Go(func() error {
		defer runtimeCancel()
		return r.handleErr(runCtx, errCmd)
	})
	wg.Go(func() error {
		defer runtimeCancel()
		return r.handleIn(runCtx, inCmd)
	})
	wg.Go(func() error {
		defer runtimeCancel()
		return r.handleAcks(runCtx)
	})
	wg.Go(func() error {
		defer runtimeCancel()
		return r.forwarder.Run(runCtx)
	})
	wg.Go(func() error {
		defer runtimeCancel()
		return r.receiver.Run(runCtx)
	})
	if err := wg.Wait(); err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, io.EOF) {
		return fmt.Errorf("action io got error: %w", err)
	}

	r.logger.Info("io done, waitring command end")
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
func (r *Runtime) IsRunning() bool {
	return atomic.LoadUint32(&r.isRunning) == 1
}

// ChangeOut заменяет отправку в oldOut на отправку в newOut.
func (r *Runtime) ChangeOut(oldOut, newOut string) error {
	return r.forwarder.ChangeOut(oldOut, newOut)
}

// GetOldestOutput возвращает самый старый output_message_id, который хранится в логе.
func (r *Runtime) GetOldestOutput() (uint32, error) {
	return r.forwarder.GetOldestOutput()
}

// inCmd закроет handleIn, так как он писатель и может это делать по
// https://golang.org/pkg/os/exec/#Cmd.StdinPipe
func (r *Runtime) handleIn(ctx context.Context, cmdIn io.WriteCloser) error {
	defer r.logger.Info("handle STDIN stopped")

	cmdWriter := ctxio.NewContextWriter(ctx, cmdIn)
	defer cmdWriter.Close()
	defer close(r.messagesQueue)

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-r.receiver.Messages():
			if msg == nil && !ok {
				return nil
			}

			r.logger.Debugf("got input data from input %d with number %d", msg.InputID, msg.Header.MessageID)

			select {
			case <-ctx.Done():
				return nil
			case r.messagesQueue <- msg:
			}

			if err := binary.Write(cmdWriter, binary.BigEndian, msg.Header.MessageLength); err != nil {
				return fmt.Errorf("can not write message length: %w", err)
			}

			if err := binary.Write(cmdWriter, binary.BigEndian, msg.Data); err != nil {
				return fmt.Errorf("can not write message data: %w", err)
			}
		}
	}
}

func (r *Runtime) handleErr(ctx context.Context, cmdErr io.Reader) error {
	defer r.logger.Info("handle STDERR stopped")

	errData := make([]byte, 4096)
	for {
		n, err := cmdErr.Read(errData)
		if err != nil {
			return fmt.Errorf("can not read stderr data: %w", err)
		}
		r.logger.Errorf("STDERR: %s", string(errData[:n]))
	}
}

func (r *Runtime) handleOut(ctx context.Context, cmdOut io.Reader) error {
	defer r.logger.Info("handle STDOUT stopped")

	for {
		// Сообщение, из которого будет получено ожидаемый выход.
		inputMsg := upstreambackup.DummyUpstreamMessage
		if !r.isSource {
			var ok bool
			select {
			case <-ctx.Done():
				return nil
			case inputMsg, ok = <-r.messagesQueue:
				if inputMsg == nil && !ok {
					return nil
				}
			}
		}
		// В этом месте ждем, что при отключении писатель, т.е. действие,
		// закроет io.Reader и разблокирует нас.
		messsageLength := uint32(0)
		if err := binary.Read(cmdOut, binary.BigEndian, &messsageLength); err != nil {
			return fmt.Errorf("can not read message length: %w", err)
		}
		r.logger.Debugf("got output data from action with length %d", messsageLength)

		data := make([]byte, messsageLength)
		if err := binary.Read(cmdOut, binary.BigEndian, data); err != nil {
			return fmt.Errorf("can not read message data: %w", err)
		}

		if err := r.forwarder.Forward(inputMsg.InputID, inputMsg.Header.MessageID, data); err != nil {
			return fmt.Errorf("can not forward message: %w", err)
		}
	}
}

func (r *Runtime) handleAcks(ctx context.Context) error {
	defer r.logger.Info("handle ACK stopped")

	acks := r.receiver.Acks()
	defer close(acks)

	for {
		select {
		case <-ctx.Done():
			return nil
		case ack, ok := <-r.forwarder.AckMessages():
			if ack == nil && !ok {
				return nil
			}

			r.logger.Debugf("got ACK: %s", ack)
			select {
			case <-ctx.Done():
				return nil
			case acks <- ack:
			}
		}
	}
}
