package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/errgroup"

	"github.com/GDVFox/gostreaming/runtime/config"
	"github.com/GDVFox/gostreaming/runtime/external"
	"github.com/GDVFox/gostreaming/runtime/logs"
)

// Action обертка над действием.
type Action struct {
	path      string
	isRunning uint32

	in  *external.TCPServer
	out []*external.TCPClient
	opt *config.ActionOptions
}

// NewAction создает новый объект Action.
func NewAction(path string, in *external.TCPServer, out []*external.TCPClient, opt *config.ActionOptions) *Action {
	return &Action{
		path:      path,
		isRunning: 0,
		in:        in,
		out:       out,
		opt:       opt,
	}
}

// Run запускает действие.
func (a *Action) Run(ctx context.Context) error {
	// Выход из этой функции означает окончание работы сервера.
	// Поэтому новых сообщений не будет и канал можно закрывать.
	defer func() {
		for _, out := range a.out {
			out.CloseInput()
		}
		logs.Logger.Infof("action: %s stopped", a.path)
	}()

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

	// errCmd, err := runActionCommand.StderrPipe()
	// if err != nil {
	// 	return fmt.Errorf("can not get stderr pipe: %w", err)
	// }

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

	// Запускаем обработчики input/output для действия.
	wg.Go(func() error {
		// inCmd закроет handleIn, так как он писатель и может это делать по
		// https://golang.org/pkg/os/exec/#Cmd.StdinPipe
		return a.handleIn(runCtx, inCmd)
	})
	wg.Go(func() error {
		return a.handleOut(runCtx, outCmd)
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
func (a *Action) IsRunning() bool {
	return atomic.LoadUint32(&a.isRunning) == 1
}

func (a *Action) handleIn(ctx context.Context, cmdIn io.WriteCloser) error {
	done := make(chan struct{})
	errs := make(chan error, 1)
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			select {
			case <-ctx.Done():
				errs <- nil
				return
			case data, ok := <-a.in.Output():
				if data == nil && !ok {
					errs <- nil
					return
				}

				logs.Logger.Debug("action: got input data")
				if _, err := cmdIn.Write(data); err != nil {
					errs <- err
					return
				}
			}
		}
	}()

	var err error
	select {
	case <-ctx.Done():
	case err = <-errs:
	}

	// В случае, если выходим по контексту, а горутина заблокирована на Write,
	// то этот Close вызовет io.EOF у Writer и обработка завершится.
	cmdIn.Close()
	// проверяем, что горутина завершилась, так как не хотим утечек.
	<-done

	return err
}

func (a *Action) handleOut(ctx context.Context, cmdOut io.Reader) error {
	reader := bufio.NewReader(cmdOut)
	dataCh := make(chan []byte, 0)

	wg, readCtx := errgroup.WithContext(ctx)
	wg.Go(func() error {
		// писатель закрывает канал.
		defer close(dataCh)

		for {
			// В этом месте ждем, что при отключении писатель, т.е. действие,
			// закроет io.Reader и разблокирует нас.
			data, err := reader.ReadBytes('\n')
			if err != nil {
				// Когда действие остановится здесь через контекст, то
				// получим тут EOF.
				if err == io.EOF {
					return nil
				}
				return err
			}

			select {
			case <-readCtx.Done():
				return nil
			case dataCh <- data:
			}
		}
	})
	wg.Go(func() error {
		sendWG := sync.WaitGroup{}
		for {
			select {
			case <-readCtx.Done():
				return nil
			case data, ok := <-dataCh:
				if data == nil && !ok {
					return nil
				}

				logs.Logger.Debug("action: got output data")
				// В случае, если контекст отменился, то дожидаемся
				// в этом месте, пока сообщение отправится всем получателям.
				// Иначе возможны ситуации, когда только часть получателей получат сообщение.
				sendWG.Add(len(a.out))
				for _, output := range a.out {
					go func(out *external.TCPClient) {
						defer sendWG.Done()
						out.Input() <- data
					}(output)
				}
				sendWG.Wait()
			}
		}
	})

	return wg.Wait()
}
