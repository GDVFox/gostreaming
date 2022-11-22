package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"

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

type changeOutRequest struct {
	OldIP   uint32
	NewIP   uint32
	OldPort uint16
	NewPort uint16
}

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

	s.listener, err = net.Listen("unix", s.sockAddr)
	if err != nil {
		return fmt.Errorf("can not listen unix: %w", err)
	}
	defer s.listener.Close()

	conns := make(chan net.Conn)
	go s.acceptConnections(conns)
	s.logger.Infof("waiting service connection on %s", s.sockAddr)

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
				clientAddr := conn.RemoteAddr()
				if conn != nil {
					conn.Close()
					conn = nil
				}
				// управляющий сервер прервал соединение, мы от этого не падаем,
				// так как на runtime это никак не отражается, он не зависим и
				// работает пока machine_node не подаст команду на завершение.
				// А machine_node может сделать это через новое соединение.
				if serviceErr == nil || serviceErr == io.EOF {
					s.logger.Warnf("service connection closed by client %s", clientAddr)
					continue
				}
				return serviceErr
			case newConn, ok := <-conns:
				if newConn == nil && !ok {
					return nil
				}
				s.logger.Infof("new service connection received %s", newConn.RemoteAddr())
				// Если до этого была горутина обработчик, то отключаем её
				// ошибки в данном случае не важны, так как этот коннект уже не будет использоваться
				// и отвечать некому.
				if conn != nil {
					conn.Close()
					<-errs

					s.logger.Info("previous connection closed for %s", newConn.RemoteAddr())
				}

				conn = newConn
				go s.handleService(conn, errs)
			}
		}
	})

	s.logger.Info("service server started")
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
			s.logger.Debug("got ping command")
			err = s.ping(conn)
		case ChangeOutCommand:
			s.logger.Info("got change out command")
			err = s.changeOut(conn)
		default:
			s.logger.Warn("got unknown command")
			err = s.unknown(conn)
		}
		if err != nil {
			errs <- err
			break
		}
	}
}

func (s *ServiceServer) ping(conn net.Conn) error {
	if s.runtime.IsRunning() {
		oldestOutput, err := s.runtime.GetOldestOutput()
		if err != nil {
			s.logger.Errorf("service: can not get oldest output: %s", err)
			return binary.Write(conn, binary.BigEndian, FailResponse)
		}

		if err := binary.Write(conn, binary.BigEndian, OKResponse); err != nil {
			return err
		}

		telemetry := runtimeTelemetry{
			OldestOutput: oldestOutput,
		}
		return binary.Write(conn, binary.BigEndian, telemetry)
	}

	return binary.Write(conn, binary.BigEndian, FailResponse)
}

func (s *ServiceServer) changeOut(conn net.Conn) error {
	req := &changeOutRequest{}
	if err := binary.Read(conn, binary.BigEndian, req); err != nil {
		return err
	}

	oldAddr := buildAddress(req.OldIP, req.OldPort)
	newAddr := buildAddress(req.NewIP, req.NewPort)
	if err := s.runtime.ChangeOut(oldAddr, newAddr); err != nil {
		s.logger.Errorf("can not change out: %s", err)
		return binary.Write(conn, binary.BigEndian, FailResponse)
	}

	return binary.Write(conn, binary.BigEndian, OKResponse)
}

func buildAddress(ipNum uint32, portNum uint16) string {
	ipByte := make([]byte, 4)
	binary.BigEndian.PutUint32(ipByte, ipNum)
	ip := net.IP(ipByte)

	port := strconv.FormatUint(uint64(portNum), 10)
	return ip.String() + ":" + port
}

func (s *ServiceServer) unknown(conn net.Conn) error {
	return binary.Write(conn, binary.BigEndian, UnknownCommandResponse)
}
