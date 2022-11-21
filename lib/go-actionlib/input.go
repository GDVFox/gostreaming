package actionlib

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

var stdin io.Reader = os.Stdin

// ReadMessage reads message from input dataflow.
func ReadMessage() ([]byte, error) {
	messageLength := uint32(0)
	if err := binary.Read(stdin, binary.BigEndian, &messageLength); err != nil {
		return nil, fmt.Errorf("read message header error: %w", err)
	}

	data := make([]byte, messageLength)
	if err := binary.Read(stdin, binary.BigEndian, data); err != nil {
		return nil, fmt.Errorf("read message data error: %w", err)
	}

	return data, nil
}
