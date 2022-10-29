package external

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"

	"github.com/GDVFox/gostreaming/runtime/logs"
	"golang.org/x/sync/errgroup"
)

// TCPServer клиент для прослушивания сообщений.
type TCPServer struct {
	listener net.Listener
	output   chan []byte

	addr string
	cfg  *TCPConnectionConfig
}

// NewTCPServer создает новый TCP server.
func NewTCPServer(addr string, cfg *TCPConnectionConfig) *TCPServer {
	return &TCPServer{
		addr:   addr,
		cfg:    cfg,
		output: make(chan []byte),
	}
}

// Run запускает TCP server и ожидает завершения
func (s *TCPServer) Run(ctx context.Context) error {
	// Выход из этой функции означает окончание работы сервера.
	// Поэтому новых сообщений не будет и канал можно закрывать.
	defer close(s.output)

	var err error
	s.listener, err = net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("can not listen tcp: %w", err)
	}

	// ошибки при создании новых коннектов нас не волнуют,
	// так как не влияют на остальных клиентов,
	// поэтому от таких ошибок мы не падаем.
	conns := make(chan *connection)
	go s.acceptConnections(conns)

	wg, receiveCtx := errgroup.WithContext(ctx)
	wg.Go(func() error {
	MainLoop:
		for {
			select {
			case <-receiveCtx.Done():
				break MainLoop
			case conn, ok := <-conns:
				if conn == nil && !ok {
					break MainLoop
				}
				wg.Go(func() error {
					return s.receiveLoop(receiveCtx, conn)
				})
			}
		}
		// новые коннекты перестанут создаваться
		return s.listener.Close()
	})

	// на данном этапе все коннекты закрыты и сервер новые коннекты не принимаются
	// можно возвращать значение и закрывать
	return wg.Wait()
}

// Output возвращает канал, в который будут записываться сообщения.
func (s *TCPServer) Output() <-chan []byte {
	return s.output
}

func (s *TCPServer) acceptConnections(conns chan<- *connection) error {
	defer close(conns)
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return fmt.Errorf("can not accept at %s: %w", s.listener.Addr(), err)
		}

		if tcp, ok := conn.(*net.TCPConn); ok {
			if err := tcp.SetNoDelay(s.cfg.NoDelay); err != nil {
				return fmt.Errorf("can not apply no delay parameter: %s: %w", strconv.FormatBool(s.cfg.NoDelay), err)
			}
		}

		logs.Logger.Debugf("received new connection")
		conns <- &connection{
			Conn:         conn,
			readTimeout:  s.cfg.ReadTimeout,
			writeTimeout: s.cfg.WriteTimeout,
		}
	}
}

func (s *TCPServer) receiveLoop(ctx context.Context, conn *connection) error {
	errs := make(chan error, 1)

	go func() {
		for {
			select {
			case <-ctx.Done():
				errs <- nil
				return
			default:
			}

			logs.Logger.Debugf("waiting for data")

			m := messages.Get()
			if err := m.readIn(conn); err != nil {
				messages.Put(m)

				if errors.Unwrap(err) == io.EOF {
					errs <- nil
					return
				}
				errs <- err
				return
			}

			logs.Logger.Debugf("server: received data message: %d (len %d)", m.Header.Seq, m.Header.Length)
			select {
			case <-ctx.Done():
				messages.Put(m)
				errs <- nil
				return
			case s.output <- m.Data:
				messages.Put(m)
			}
		}
	}()

	logs.Logger.Debugf("started receiving loop")
	var err error
	select {
	case <-ctx.Done():
	case err = <-errs:
	}

	conn.Close()

	logs.Logger.Debugf("stopped receiving loop")
	return err
}
