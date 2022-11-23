package connutil

import (
	"net"
	"time"
)

// ConnectionConfig набор параметров для подключения по tcp.
type ConnectionConfig struct {
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// NewConnectionConfig возвращает ConnectionConfig с настройками по-умолчанию.
func NewConnectionConfig() *ConnectionConfig {
	return &ConnectionConfig{
		DialTimeout:  0,
		ReadTimeout:  0,
		WriteTimeout: 0,
	}
}

// Connection обертка для tcp коннекта с поддержкой таймаутов.
type Connection struct {
	net.Conn
	readTimeout  time.Duration
	writeTimeout time.Duration

	lastReadDeadline  time.Time
	lastWriteDeadline time.Time
}

// NewConnection создает новую обертку для TCP подключения.
func NewConnection(conn net.Conn, cfg *ConnectionConfig) *Connection {
	return &Connection{
		Conn:         conn,
		readTimeout:  cfg.ReadTimeout,
		writeTimeout: cfg.WriteTimeout,
	}
}

// NewDefaultConnection возвращает обертку для TCP подключения, без активации таймаутов.
func NewDefaultConnection(conn net.Conn) *Connection {
	return &Connection{
		Conn:         conn,
		readTimeout:  0,
		writeTimeout: 0,
	}
}

func (c *Connection) Read(b []byte) (int, error) {
	now := TimeNow()
	if c.readTimeout != 0 && now.Sub(c.lastReadDeadline) > (c.readTimeout>>2) {
		c.Conn.SetReadDeadline(time.Now().Add(c.readTimeout))
		c.lastReadDeadline = now
	}
	return c.Conn.Read(b)
}

func (c *Connection) Write(b []byte) (int, error) {
	now := TimeNow()
	if c.writeTimeout != 0 && now.Sub(c.lastWriteDeadline) > (c.writeTimeout>>2) {
		c.Conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
		c.lastWriteDeadline = now
	}
	return c.Conn.Write(b)
}

// Close закрывает вложенное TCP соединение.
func (c *Connection) Close() error {
	return c.Conn.Close()
}
