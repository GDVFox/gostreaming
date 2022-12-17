package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"

	"github.com/GDVFox/ctxio"
	"github.com/GDVFox/gostreaming/util"
	"golang.org/x/sync/errgroup"
)

const (
	// PingCommand команда для проверки состояния runtime.
	PingCommand uint8 = 0x1
	// ChangeOutCommand команда для изменения выходного потока.
	ChangeOutCommand uint8 = 0x2
)

const (
	// OKResponse ответ, предполагающий успешное выполнение действия.
	OKResponse uint8 = 0x0
	// FailResponse ответ, предполагающий ошибочное выполнение действия.
	FailResponse uint8 = 0x1
	// UnknownCommandResponse полученная неизвестная команда.
	UnknownCommandResponse uint8 = 0x2
)

type runtimeTelemetry struct {
	OldestOutput uint32
}

// ServiceServer UDP сервис для получения команд от machine_node.
type ServiceServer struct {
	listener net.Listener
	runtime  *Runtime

	sockAddr string
	logger   *util.Logger
}

// NewServiceServer создает новый UDP сервисный сервер.
func NewServiceServer(addr string, runtime *Runtime, l *util.Logger) *ServiceServer {
	return &ServiceServer{
		runtime:  runtime,
		sockAddr: addr,
		logger:   l.WithName("service_server"),
	}
}

// Run запускает сервисный сервер и ожидает завершения.
func (s *ServiceServer) Run(ctx context.Context) error {
	defer s.logger.Info("service server stopped")

	var err error
	if err := os.RemoveAll(s.sockAddr); err != nil {
		return fmt.Errorf("can not remove previous socket: %w", err)
	}

	serviceWG := &sync.WaitGroup{}
	defer serviceWG.Wait()

	s.listener, err = net.Listen("unix", s.sockAddr)
	if err != nil {
		return fmt.Errorf("can not listen unix: %w", err)
	}

	serviceWG.Add(1)
	go func() {
		defer serviceWG.Done()

		<-ctx.Done()
		if err := s.listener.Close(); err != nil {
			s.logger.Errorf("can not close listener: %s", err)
			return
		}
		s.logger.Info("listener closed")
	}()

	wg, serviceCtx := errgroup.WithContext(ctx)
	wg.Go(func() error {
		var conn net.Conn
		for {
			newConn, err := s.listener.Accept()
			if err != nil {
				return fmt.Errorf("can not accept at %s: %w", s.listener.Addr(), err)
			}
			s.logger.Infof("new service connection received %s", newConn.RemoteAddr())

			if conn != nil {
				conn.Close()
				conn = nil

				s.logger.Info("previous connection closed for %s", newConn.RemoteAddr())
			}
			conn = newConn

			serviceWG.Add(1)
			go func() {
				defer serviceWG.Done()
				s.handleService(serviceCtx, conn)
			}()
		}
	})

	s.logger.Info("service server started")
	if err := wg.Wait(); err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, io.EOF) {
		return fmt.Errorf("service server got error: %w", err)
	}
	return nil
}

func (s *ServiceServer) handleService(ctx context.Context, conn net.Conn) {
	defer s.logger.Infof("handle service %s stopped", conn.RemoteAddr())

	connReader := ctxio.NewContextReader(ctx, conn)
	defer connReader.Free()

	for {
		var command uint8
		if err := binary.Read(connReader, binary.BigEndian, &command); err != nil {
			s.logger.Errorf("handle service can not read command: %s", err)
			return
		}

		var err error
		switch command {
		case PingCommand:
			s.logger.Debug("got ping command")
			err = s.ping(ctx, conn)
		case ChangeOutCommand:
			s.logger.Info("got change out command")
			err = s.changeOut(ctx, conn)
		default:
			s.logger.Warn("got unknown command")
			err = s.unknown(ctx, conn)
		}
		if err != nil {
			s.logger.Errorf("handle service command err: %s", err)
			return
		}
	}
}

func (s *ServiceServer) ping(ctx context.Context, conn net.Conn) error {
	connWriter := ctxio.NewContextWriter(ctx, conn)
	defer connWriter.Free()

	if s.runtime.IsRunning() {
		oldestOutput, err := s.runtime.GetOldestOutput()
		if err != nil {
			s.logger.Errorf("service: can not get oldest output: %s", err)
			return binary.Write(connWriter, binary.BigEndian, FailResponse)
		}

		if err := binary.Write(connWriter, binary.BigEndian, OKResponse); err != nil {
			return err
		}

		telemetry := runtimeTelemetry{
			OldestOutput: oldestOutput,
		}
		return binary.Write(connWriter, binary.BigEndian, telemetry)
	}

	return binary.Write(connWriter, binary.BigEndian, FailResponse)
}

func (s *ServiceServer) changeOut(ctx context.Context, conn net.Conn) error {
	connWriter := ctxio.NewContextWriter(ctx, conn)
	defer connWriter.Free()

	connReader := ctxio.NewContextReader(ctx, conn)
	defer connReader.Free()

	oldAddr, err := s.readChangeOutAddr(connReader)
	if err != nil {
		return err
	}
	newAddr, err := s.readChangeOutAddr(connReader)
	if err != nil {
		return err
	}

	s.logger.Errorf("OLOLO: %s %s", oldAddr, newAddr)
	if err := s.runtime.ChangeOut(oldAddr, newAddr); err != nil {
		s.logger.Errorf("can not change out: %s", err)
		return binary.Write(connWriter, binary.BigEndian, FailResponse)
	}

	return binary.Write(connWriter, binary.BigEndian, OKResponse)
}

func (s *ServiceServer) readChangeOutAddr(r io.Reader) (string, error) {
	var addrLen uint64
	if err := binary.Read(r, binary.BigEndian, &addrLen); err != nil {
		return "", fmt.Errorf("can not read change out addr len: %s", err)
	}

	rawAddr := make([]byte, addrLen)
	if err := binary.Read(r, binary.BigEndian, rawAddr); err != nil {
		return "", fmt.Errorf("can not read change out addr: %s", err)
	}

	return string(rawAddr), nil
}

func (s *ServiceServer) unknown(ctx context.Context, conn net.Conn) error {
	connWriter := ctxio.NewContextWriter(ctx, conn)
	defer connWriter.Free()

	return binary.Write(connWriter, binary.BigEndian, UnknownCommandResponse)
}
