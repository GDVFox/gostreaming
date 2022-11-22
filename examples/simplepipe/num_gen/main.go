package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"time"

	"github.com/GDVFox/gostreaming/lib/go-actionlib"
)

var (
	freq int64
)

func init() {
	flag.Int64Var(&freq, "freq", 100, "delay between messages in milliseconds")
}

func main() {
	flag.Parse()

	for i := 1; ; i++ {
		data := make([]byte, 4)
		binary.BigEndian.PutUint32(data, uint32(i))

		if err := actionlib.WriteMessage(data); err != nil {
			actionlib.WriteError(fmt.Errorf("write size error: %w", err))
		}

		time.Sleep(time.Duration(int64(time.Millisecond) * freq))
	}
}
