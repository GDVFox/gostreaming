package actionlib

import (
	"fmt"
	"io"
	"os"
)

var stderr io.Writer = os.Stderr

// WriteError sends an error to runtime.
func WriteError(err error) {
	fmt.Fprintf(stderr, err.Error())
}

// WriteFatal sends an fatal error to runtime.
// After a fatal error, the action stops.
func WriteFatal(err error) {
	WriteError(err)
	os.Exit(1)
}
