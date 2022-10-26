package external

import (
	"net"
	"time"
)

// TCPConnectionConfig набор параметров для подключения по tcp.
type TCPConnectionConfig struct {
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	NoDelay      bool
}

// NewTCPConnectionConfig возвращает TCPConnectionConfig с настройками по-умолчанию.
func NewTCPConnectionConfig() *TCPConnectionConfig {
	return &TCPConnectionConfig{
		DialTimeout:  0,
		ReadTimeout:  0,
		WriteTimeout: 0,
		NoDelay:      true,
	}
}

// TCPConnection обертка для tcp коннекта
type connection struct {
	net.Conn
	readTimeout  time.Duration
	writeTimeout time.Duration

	lastReadDeadline  time.Time
	lastWriteDeadline time.Time
}

func (c *connection) Read(b []byte) (int, error) {
	now := TimeNow()
	if c.readTimeout != 0 && now.Sub(c.lastReadDeadline) > c.readTimeout {
		c.Conn.SetReadDeadline(now.Add(c.readTimeout))
		c.lastReadDeadline = now
	}
	return c.Conn.Read(b)
}

func (c *connection) Write(b []byte) (int, error) {
	now := TimeNow()
	if c.writeTimeout != 0 && now.Sub(c.lastWriteDeadline) > c.writeTimeout {
		c.Conn.SetWriteDeadline(now.Add(c.writeTimeout))
		c.lastWriteDeadline = now
	}
	return c.Conn.Write(b)
}

func (c *connection) Close() error {
	return c.Conn.Close()
}
