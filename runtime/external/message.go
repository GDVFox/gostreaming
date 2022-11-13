package external

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
)

var (
	// ErrNoEnoughtData возвращается, когда длина полученного сообщения меньше,
	// чем переданная в заголовке.
	ErrNoEnoughtData = errors.New("too small data")
)

var (
	messages = messagePool{
		p: sync.Pool{
			New: func() interface{} {
				return new(message)
			},
		},
	}
)

type messagePool struct {
	p sync.Pool
}

func (p *messagePool) Get() *message {
	return p.p.Get().(*message)
}

func (p *messagePool) Put(o *message) {
	o.Header.Seq = 0
	o.Header.Flags = 0
	o.Header.Length = 0
	o.Data = nil

	p.p.Put(o)
}

// Схема сообщения с заголовком:
//
// 0         8        16        24        32
// +---------+---------+---------+---------+
// |        seq        |       flags       |
// +---------+---------+---------+---------+
// |                length                 |
// +---------+---------+---------+---------+
// |                                       |
// .                 data                  .
// .                                       .
// .                                       .
// +----------------------------------------
//
// Передается в big-endian (network byte order).
//
// seq — порядковый номер сообщения.
// flags — дополнительные флаги
//   0x1 — флаг сжатия, если выставлен используется сжатие с помощью zstd/lz4.
// length — длина сообщения
// data — непосредственно сообщение.

type header struct {
	Seq    uint16
	Flags  uint16
	Length uint32
}

type message struct {
	Header header
	Data   []byte
}

func (m *message) readIn(r io.Reader) error {
	if err := binary.Read(r, binary.BigEndian, &m.Header); err != nil {
		return fmt.Errorf("can not read header: %w", err)
	}

	m.Data = make([]byte, m.Header.Length)
	read, err := r.Read(m.Data)
	if err != nil {
		return fmt.Errorf("can not read data: %w", err)
	}
	if read != int(m.Header.Length) {
		return fmt.Errorf("expected length %d, got %d: %w", m.Header.Length, read, ErrNoEnoughtData)
	}

	return nil
}

func (m *message) writeOut(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, m.Header); err != nil {
		return fmt.Errorf("can not write header (seq: %d): %w", m.Header.Seq, err)
	}
	return binary.Write(w, binary.BigEndian, m.Data)
}
