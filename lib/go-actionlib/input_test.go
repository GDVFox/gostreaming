package actionlib

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadMessage(t *testing.T) {
	var buff bytes.Buffer
	stdin = &buff

	messages := [][]byte{
		[]byte("Hello"),
		[]byte("World"),
		{0x1, 0x3, 0x3, 0x7},
		[]byte("🍪𓅿"),
	}

	for _, msg := range messages {
		assert.NoError(t, binary.Write(&buff, binary.BigEndian, uint32(len(msg))))
		assert.NoError(t, binary.Write(&buff, binary.BigEndian, msg))
	}

	for i, msg := range messages {
		data, err := ReadMessage()
		assert.NoError(t, err)
		assert.EqualValuesf(t, msg, data, "Failed #%d:", i)
	}
}
