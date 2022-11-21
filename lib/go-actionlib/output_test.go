package actionlib

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteMessage(t *testing.T) {
	var buff bytes.Buffer
	stdout = &buff

	messages := [][]byte{
		[]byte("Hello"),
		[]byte("World"),
		{0x1, 0x3, 0x3, 0x7},
		[]byte("🍪𓅿"),
	}

	for _, msg := range messages {
		assert.NoError(t, WriteMessage(msg))
	}

	for i, msg := range messages {
		msgLength := uint32(0)
		assert.NoError(t, binary.Read(&buff, binary.BigEndian, &msgLength))
		assert.EqualValuesf(t, len(msg), msgLength, "Failed length check #%d:", i)

		data := make([]byte, msgLength)
		assert.NoError(t, binary.Read(&buff, binary.BigEndian, data))
		assert.EqualValuesf(t, msg, data, "Failed data check #%d:", i)
	}
}

func TestAckMessage(t *testing.T) {
	var buff bytes.Buffer
	stdout = &buff

	// a non-zero value to verify
	// that ackMessage is being overwritten.
	ackMessage := uint32(1337)
	assert.NoError(t, AckMessage())
	assert.NoError(t, binary.Read(&buff, binary.BigEndian, &ackMessage))
	assert.EqualValues(t, 0, ackMessage)
}
