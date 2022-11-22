package main

import (
	"encoding/binary"
	"flag"

	"github.com/GDVFox/gostreaming/lib/go-actionlib"
)

var (
	mod int
)

func init() {
	flag.IntVar(&mod, "mod", 1, "passes messages that are multiples of mod")
}

func main() {
	flag.Parse()

	for {
		data, err := actionlib.ReadMessage()
		if err != nil {
			actionlib.WriteError(err)
			continue
		}

		number := binary.BigEndian.Uint32(data)
		if int(number)%mod != 0 {
			actionlib.AckMessage()
			continue
		}

		if err := actionlib.WriteMessage(data); err != nil {
			actionlib.WriteError(err)
			continue
		}
	}

}
