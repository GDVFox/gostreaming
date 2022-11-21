package actionlib

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

var stdout io.Writer = os.Stdout

// WriteMessage writes message to output dataflow.
func WriteMessage(message []byte) error {
	messageLength := uint32(len(message))
	if err := binary.Write(stdout, binary.BigEndian, messageLength); err != nil {
		return fmt.Errorf("write message header error: %w", err)
	}
	if err := binary.Write(stdout, binary.BigEndian, message); err != nil {
		return fmt.Errorf("write message data error: %w", err)
	}
	return nil
}

// AckMessage method required to acknowledge the message in the sink component.
// In the case where the action skips the message AckMessage() call is not necessary.
func AckMessage() error {
	if err := binary.Write(stdout, binary.BigEndian, uint32(0)); err != nil {
		return fmt.Errorf("write ACK error: %w", err)
	}
	return nil
}
