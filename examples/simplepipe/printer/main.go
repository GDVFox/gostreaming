package main

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"

	"github.com/GDVFox/gostreaming/lib/go-actionlib"
	"golang.org/x/sync/errgroup"
)

var (
	port int
)

func init() {
	flag.IntVar(&port, "port", 0, "Port of tcp server")
}

func runReadLoop(dataCh chan uint32) {
	defer close(dataCh)

	for {
		data, err := actionlib.ReadMessage()
		if err != nil {
			actionlib.WriteFatal(err)
		}
		num := binary.BigEndian.Uint32(data)
		dataCh <- num
		if err := actionlib.AckMessage(); err != nil {
			actionlib.WriteFatal(err)
		}
	}
}

type broadcaster struct {
	connsMutex sync.Mutex
	conns      []net.Conn

	listener net.Listener
}

func newBroadcater(port int) (*broadcaster, error) {
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return nil, fmt.Errorf("can not create listener: %w", err)
	}

	return &broadcaster{
		conns:    make([]net.Conn, 0),
		listener: listener,
	}, nil
}

func (b *broadcaster) run(dataCh chan uint32) error {
	wg, ctx := errgroup.WithContext(context.Background())
	wg.Go(func() error {
		return b.accpetLoop()
	})
	wg.Go(func() error {
		return b.transmitLoop(ctx, dataCh)
	})

	return wg.Wait()
}

func (b *broadcaster) accpetLoop() error {
	defer b.listener.Close()

	for {
		conn, err := b.listener.Accept()
		if err != nil {
			return err
		}

		b.connsMutex.Lock()
		b.conns = append(b.conns, conn)
		b.connsMutex.Unlock()
	}
}

func (b *broadcaster) transmitLoop(ctx context.Context, dataCh chan uint32) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case data, ok := <-dataCh:
			if data == 0 && !ok {
				return nil
			}

			row := []byte(strconv.Itoa(int(data)))
			b.connsMutex.Lock()
			i := 0
			for _, conn := range b.conns {
				_, err := conn.Write(append(row, '\n'))
				if err != nil {
					conn.Close()
					continue
				}
				// replace bad connections
				b.conns[i] = conn
				i++
			}

			b.conns = b.conns[:i]
			b.connsMutex.Unlock()
		}
	}
}

func main() {
	flag.Parse()

	key := os.Getenv("SECRET_KEY")
	if key != "please" {
		panic("secret key is unknown")
	}

	dataCh := make(chan uint32)
	go runReadLoop(dataCh)

	b, err := newBroadcater(port)
	if err != nil {
		actionlib.WriteFatal(err)
	}

	if err := b.run(dataCh); err != nil {
		actionlib.WriteFatal(err)
	}
}
