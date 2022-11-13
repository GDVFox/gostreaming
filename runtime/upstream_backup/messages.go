package upstreambackup

import (
	"encoding/binary"
	"fmt"
	"io"
)

type helloMessage struct {
	NameLength uint32
	Name       []byte
}

func (m *helloMessage) readIn(r io.Reader) error {
	if err := binary.Read(r, binary.BigEndian, &m.NameLength); err != nil {
		return fmt.Errorf("can not read hello message header: %w", err)
	}
	m.Name = make([]byte, m.NameLength)
	if err := binary.Read(r, binary.BigEndian, m.Name); err != nil {
		return fmt.Errorf("can not read hello message data: %w", err)
	}
	return nil
}

func (m *helloMessage) writeOut(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, m.NameLength); err != nil {
		return fmt.Errorf("can not send hello message header: %w", err)
	}
	if err := binary.Write(w, binary.BigEndian, m.Name); err != nil {
		return fmt.Errorf("can not send hello message data: %w", err)
	}
	return nil
}

type dataMessageHeader struct {
	MessageID     uint32
	Reserved      uint16
	Flags         uint16
	MessageLength uint32
}

type dataMessage struct {
	Header dataMessageHeader
	Data   []byte
}

func (m *dataMessage) readIn(r io.Reader) error {
	if err := binary.Read(r, binary.BigEndian, &m.Header); err != nil {
		return fmt.Errorf("can not read data message header: %w", err)
	}
	m.Data = make([]byte, m.Header.MessageLength)
	if err := binary.Read(r, binary.BigEndian, m.Data); err != nil {
		return fmt.Errorf("can not read data message data: %w", err)
	}
	return nil
}

func (m *dataMessage) writeOut(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, m.Header); err != nil {
		return fmt.Errorf("can not send data message header: %w", err)
	}

	if err := binary.Write(w, binary.BigEndian, m.Data); err != nil {
		return fmt.Errorf("can not send data message data: %w", err)
	}
	return nil
}

type ackMessage uint32

func (m *ackMessage) readIn(r io.Reader) error {
	if err := binary.Read(r, binary.BigEndian, m); err != nil {
		return fmt.Errorf("can not read ack message data: %w", err)
	}
	return nil
}

func (m *ackMessage) writeOut(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, *m); err != nil {
		return fmt.Errorf("can not send ack message data: %w", err)
	}
	return nil
}

type forwardLogHeader struct {
	InputID         uint16
	Flags           uint16
	InputMessageID  uint32
	OutputMessageID uint32
	MessageLength   uint32
}

type forwardLogItem struct {
	Header forwardLogHeader
	Data   []byte
}

func (m *forwardLogItem) readIn(r io.Reader) error {
	if err := binary.Read(r, binary.BigEndian, &m.Header); err != nil {
		return fmt.Errorf("can not read forward log item header: %w", err)
	}
	m.Data = make([]byte, m.Header.MessageLength)
	if err := binary.Read(r, binary.BigEndian, m.Data); err != nil {
		return fmt.Errorf("can not read forward log item data: %w", err)
	}
	return nil
}

func (m *forwardLogItem) writeOut(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, m.Header); err != nil {
		return fmt.Errorf("can not write forward log item header: %w", err)
	}

	if err := binary.Write(w, binary.BigEndian, m.Data); err != nil {
		return fmt.Errorf("can not write forward log item header: %w", err)
	}
	return nil
}
