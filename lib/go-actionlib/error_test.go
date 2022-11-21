package actionlib

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteError(t *testing.T) {
	var buff bytes.Buffer
	stderr = &buff

	someErr := errors.New("aAAaAaaAAAAAaaA")
	WriteError(someErr)
	assert.EqualValues(t, someErr.Error(), string(buff.Bytes()))
}
