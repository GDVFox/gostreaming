package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/GDVFox/gostreaming/runtime/logs"
	"golang.org/x/sync/errgroup"
)

const (
	// PingCommand команда для проверки состояния runtime.
	PingCommand uint8 = 0x1
)

const (
	// OKResponse ответ, предполагающий успешное выполнение действия.
	OKResponse uint8 = 0x0
	// FailResponse ответ, предполагающий ошибочное выполнение действия.
	FailResponse uint8 = 0x1
	// UnknownCommandResponse полученная неизвестная команда.
	UnknownCommandResponse uint8 = 0x2
)

// ServiceServer UDP сервис для получения команд от machine_node.
type ServiceServer struct {
	listener net.Listener
	action   *Action

	sockAddr string
}

// NewServiceServer создает новый UDP сервисный сервер.
func NewServiceServer(addr string, action *Action) *ServiceServer {
	return &ServiceServer{
		action:   action,
		sockAddr: addr,
	}
}

// Run запускает сервисный сервер и ожидает завершения.
func (s *ServiceServer) Run(ctx context.Context) error {
	var err error
	if err := os.RemoveAll(s.sockAddr); err != nil {
		return fmt.Errorf("can not remove previous socket: %w", err)
	}

	s.listener, err = net.Listen("unix", s.sockAddr)
	if err != nil {
		return fmt.Errorf("can not listen unix: %w", err)
	}
	defer s.listener.Close()

	conns := make(chan net.Conn)
	go s.acceptConnections(conns)

	wg, _ := errgroup.WithContext(ctx)
	wg.Go(func() error {
		var (
			conn net.Conn
			errs = make(chan error)
		)
		for {
			select {
			case <-ctx.Done():
				return nil
			case serviceErr := <-errs:
				if conn != nil {
					conn.Close()
					conn = nil
				}
				// управляющий сервер прервал соединение, мы от этого не падаем,
				// так как на runtime это никак не отражается, он не зависим и
				// работает пока machine_node не подаст команду на завершение.
				// А machine_node может сделать это через новое соединение
				if serviceErr == nil || serviceErr == io.EOF {
					logs.Logger.Warn("service connection closed by client")
					continue
				}
				return serviceErr
			case newConn, ok := <-conns:
				if newConn == nil && !ok {
					return nil
				}
				logs.Logger.Info("new service connection received")
				// Если до этого была горутина обработчик, то отключаем её
				// ошибки в данном случае не важны, так как этот коннект уже не будет использоваться
				// и отвечать некому.
				if conn != nil {
					conn.Close()
					<-errs

					logs.Logger.Info("previous connection closed")
				}

				conn = newConn
				go s.handleService(conn, errs)
			}
		}
	})

	logs.Logger.Info("service server started")
	defer logs.Logger.Info("service server stopped")
	return wg.Wait()
}

func (s *ServiceServer) acceptConnections(conns chan<- net.Conn) error {
	defer close(conns)
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return fmt.Errorf("can not accept at %s: %w", s.listener.Addr(), err)
		}
		conns <- conn
	}
}

func (s *ServiceServer) handleService(conn net.Conn, errs chan error) {
	for {
		var command uint8
		if err := binary.Read(conn, binary.BigEndian, &command); err != nil {
			errs <- err
			break
		}

		var err error
		switch command {
		case PingCommand:
			logs.Logger.Info("got ping command")
			err = s.ping(conn)
		default:
			logs.Logger.Warn("got unknown command")
			err = s.unknown(conn)
		}
		if err != nil {
			errs <- err
			break
		}
	}
}

func (s *ServiceServer) ping(conn net.Conn) error {
	if s.action.IsRunning() {
		return binary.Write(conn, binary.BigEndian, OKResponse)
	}
	return binary.Write(conn, binary.BigEndian, FailResponse)
}

func (s *ServiceServer) unknown(conn net.Conn) error {
	return binary.Write(conn, binary.BigEndian, UnknownCommandResponse)
}
