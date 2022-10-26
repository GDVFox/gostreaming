package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/GDVFox/gostreaming/runtime/external"
	"github.com/GDVFox/gostreaming/runtime/logs"
)

// Action обертка над действием.
type Action struct {
	path string

	in  *external.TCPServer
	out []*external.TCPClient
}

// NewAction создает новый объект Action.
func NewAction(path string, in *external.TCPServer, out []*external.TCPClient) *Action {
	return &Action{
		path: path,
		in:   in,
		out:  out,
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
	}()

	wg, runCtx := errgroup.WithContext(ctx)
	runActionCommand := exec.CommandContext(runCtx, a.path)
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

	wg.Go(func() error {
		return a.handleIn(inCmd)
	})
	wg.Go(func() error {
		return a.handleOut(outCmd)
	})
	wg.Go(func() error {
		return runActionCommand.Run()
	})
	// Здесь дождемся, когда каждая горутина завершится,
	// т.е. будут отправлены последние сообщения,
	// бинарник будет отключен.
	return wg.Wait()
}

func (a *Action) handleIn(cmdIn io.WriteCloser) error {
	for data := range a.in.Output() {
		logs.Logger.Debug("action: got input data")
		if _, err := cmdIn.Write(data); err != nil {
			return err
		}
	}
	return nil
}

func (a *Action) handleOut(cmdOut io.ReadCloser) error {
	reader := bufio.NewReader(cmdOut)
	sendWG := sync.WaitGroup{}
	for {
		data, err := reader.ReadBytes('\n')
		if err != nil {
			// Когда действие остановится здесь через контекст, то
			// получим тут EOF.
			if err == io.EOF {
				return nil
			}
			return err
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
