package external

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/GDVFox/gostreaming/runtime/logs"
)

// TCPClient клиент для передачи сообщений далее по пайплайну.
type TCPClient struct {
	msgCounter uint16
	input      chan []byte

	addr string
	cfg  *TCPConnectionConfig
}

// NewTCPClient создает новый TCP connect.
func NewTCPClient(addr string, cfg *TCPConnectionConfig) *TCPClient {
	return &TCPClient{
		msgCounter: 0,
		addr:       addr,
		cfg:        cfg,
		input:      make(chan []byte),
	}
}

// Run запускает клиент.
func (c *TCPClient) Run(ctx context.Context) error {
	defer logs.Logger.Debugf("client: stopped")

	conn, err := net.DialTimeout("tcp", c.addr, c.cfg.DialTimeout)
	if err != nil {
		return fmt.Errorf("can not dial tcp: %w", err)
	}
	defer conn.Close() // соединение всегда закрывается.

	if tcp, ok := conn.(*net.TCPConn); ok {
		if err := tcp.SetNoDelay(c.cfg.NoDelay); err != nil {
			return fmt.Errorf("can not apply no delay parameter: %s: %w", strconv.FormatBool(c.cfg.NoDelay), err)
		}
	}

	tcpConn := &connection{
		Conn:         conn,
		readTimeout:  c.cfg.ReadTimeout,
		writeTimeout: c.cfg.WriteTimeout,
	}

	return c.transmitingLoop(ctx, tcpConn)
}

func (c *TCPClient) transmitingLoop(ctx context.Context, conn *connection) error {
	errs := make(chan error, 1)
	go func() {
		for {
			// Если отключение будет тут,
			// то писатель должен закрыть канал.
			// Если здесь опираться на контекст, то возможно ситуация,
			// когда писатель зависнет в записи в канал.
			data, ok := <-c.input
			if data == nil && !ok {
				errs <- nil
				return
			}

			m := messages.Get()
			m.Header.Seq = c.msgCounter
			m.Header.Length = uint32(len(data))
			m.Data = data
			// если отключение застянет нас тут,
			// то далее выбьет select по контексту,
			// который закроет коннект и данная горутина завершится.
			if err := m.writeOut(conn); err != nil {
				messages.Put(m)
				errs <- err
				return
			}

			logs.Logger.Debugf("client: transmitted data message: %d (len %d)", m.Header.Seq, m.Header.Length)

			messages.Put(m)
			c.msgCounter++
		}
	}()

	logs.Logger.Debugf("started transmiting loop")
	var err error
	select {
	case <-ctx.Done():
	case err = <-errs:
	}

	conn.Close()
	logs.Logger.Debugf("stopped transmiting loop")
	return err
}

// Input возвращает входной канал.
func (c *TCPClient) Input() chan<- []byte {
	return c.input
}

// CloseInput закрывает входной канал.
func (c *TCPClient) CloseInput() {
	close(c.input)
}
