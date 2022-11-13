package upstreambackup

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/GDVFox/ctxio"
	"github.com/GDVFox/gostreaming/runtime/external"
	"github.com/GDVFox/gostreaming/runtime/logs"
	"golang.org/x/sync/errgroup"
)

// Возможные ошибки работы.
var (
	ErrUpstreamUnknown = errors.New("upstreams is unknown")
)

type workingUpstream struct {
	upstream     *UpstreamReceiver
	stopUpstream context.CancelFunc
}

// DefaiultReceiverConfig набор параметров для DefaiultReceiver.
type DefaiultReceiverConfig struct {
	UpstreamConfig *UpstreamReceiverConfig
}

// DefaiultReceiver получает сообщения от вышестоящих узлов.
type DefaiultReceiver struct {
	upstreamIndex uint16

	acks     chan UpstreamAck
	messages chan *UpstreamMessage
	addr     string

	upstreamInWorkMutex   sync.Mutex
	upstreamInWork        map[string]*workingUpstream
	upstreamInWorkIndexes map[uint16]string

	upstreamNames map[string]struct{}
	upsteamConfig *UpstreamReceiverConfig
}

// NewDefaiultReceiver возвращает новый объект DefaiultReceiver.
func NewDefaiultReceiver(addr string, inNames []string, cfg *DefaiultReceiverConfig) *DefaiultReceiver {
	upstreamNames := make(map[string]struct{})
	for _, in := range inNames {
		upstreamNames[in] = struct{}{}
	}

	return &DefaiultReceiver{
		upstreamIndex:         0,
		addr:                  addr,
		acks:                  make(chan UpstreamAck),
		messages:              make(chan *UpstreamMessage),
		upstreamNames:         upstreamNames,
		upstreamInWork:        make(map[string]*workingUpstream),
		upstreamInWorkIndexes: make(map[uint16]string),
		upsteamConfig:         cfg.UpstreamConfig,
	}
}

// Run запускает обработчик.
func (r *DefaiultReceiver) Run(ctx context.Context) error {
	defer logs.Logger.Debugf("receiver: stopped")
	defer close(r.messages)

	receiverWG := &sync.WaitGroup{}
	defer receiverWG.Wait() // блокируемся, пока все ресурсы не освободятся.

	listener, err := net.Listen("tcp", r.addr)
	if err != nil {
		return fmt.Errorf("can not listen tcp: %w", err)
	}

	receiverWG.Add(1)
	go func() {
		defer receiverWG.Done()

		<-ctx.Done()
		if err := listener.Close(); err != nil {
			logs.Logger.Errorf("can not close listener: %s", err)
		}
		logs.Logger.Info("receiver: listener closed")
	}()

	// receiveCtx будет отменен в случае ошибки и остановит все upstream.
	// для этого здесь и используем errgroup.
	wg, receiveCtx := errgroup.WithContext(ctx)
	wg.Go(func() error {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return fmt.Errorf("can not accept at %s: %w", listener.Addr(), err)
			}
			logs.Logger.Infof("accepted connection from %s", conn.RemoteAddr().String())

			newUpstreamIndex := r.upstreamIndex
			r.upstreamIndex++

			receiverWG.Add(1)
			go func() {
				defer receiverWG.Done()
				r.runUpstream(receiveCtx, newUpstreamIndex, external.NewTCPConnection(conn, r.upsteamConfig.TCPConfig))
			}()
		}
	})
	wg.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case ack, ok := <-r.acks:
				if ack == nil && !ok {
					return nil
				}

				r.ackUpstreams(ack)
			}
		}
	})

	logs.Logger.Infof("receiver: waiting for new connections")
	defer logs.Logger.Infof("receiver: stopped")

	return wg.Wait()
}

func (r *DefaiultReceiver) runUpstream(ctx context.Context, upstreamIndex uint16, tcpConn *external.TCPConnection) {
	defer tcpConn.Close()

	upstreamCtx, upstreamStop := context.WithCancel(ctx)
	defer upstreamStop()

	upstreamName, err := r.listenHello(upstreamCtx, tcpConn)
	if err != nil {
		logs.Logger.Errorf("read hello message failed: %s", err)
		return
	}
	logs.Logger.Debugf("receiver: got hello message from %s: %s", tcpConn.Conn.RemoteAddr(), upstreamName)

	r.upstreamInWorkMutex.Lock()
	// Здесь мы можем попасть на уже завершенный upstream,
	// но это ок, так как ничего не меняется от отправки сигнала.
	if workingUpstream, ok := r.upstreamInWork[upstreamName]; ok {
		workingUpstream.stopUpstream()
		delete(r.upstreamInWork, upstreamName)
		delete(r.upstreamInWorkIndexes, workingUpstream.upstream.upstreamIndex)

		logs.Logger.Debugf("receiver: send stop signal to previous upstream %s", upstreamName)
	}

	upstream := NewUpstreamReceiver(upstreamIndex, upstreamName, tcpConn, r.upsteamConfig)
	r.upstreamInWork[upstreamName] = &workingUpstream{
		upstream:     upstream,
		stopUpstream: upstreamStop,
	}
	r.upstreamInWorkIndexes[upstreamIndex] = upstreamName
	r.upstreamInWorkMutex.Unlock()

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		defer logs.Logger.Debugf("receiver: stopped upstream %s", upstream.name)

		if err := upstream.Run(upstreamCtx); err != nil {
			logs.Logger.Errorf("receiver: upstream %s stopped: %s", upstream.name, err)
		}
	}()
	go func() {
		defer wg.Done()
		defer logs.Logger.Debugf("receiver: stopped read message loop for upstream %s", upstream.name)

		for message := range upstream.output {
			select {
			case <-upstreamCtx.Done():
				return
			case r.messages <- message:
			}
		}
	}()

	wg.Wait()
}

func (r *DefaiultReceiver) listenHello(ctx context.Context, tcpConn *external.TCPConnection) (string, error) {
	connReader := ctxio.NewContextReader(ctx, tcpConn)
	defer connReader.Free()

	hello := &helloMessage{}
	if err := hello.readIn(connReader); err != nil {
		return "", err
	}

	upstreamName := string(hello.Name)
	if _, ok := r.upstreamNames[upstreamName]; !ok {
		return "", fmt.Errorf("for %s: %w", upstreamName, ErrUpstreamUnknown)
	}

	return upstreamName, nil
}

// Messages возвращает канал с сообщениями.
func (r *DefaiultReceiver) Messages() <-chan *UpstreamMessage {
	return r.messages
}

// Acks возвращает канал для передачи Ack сообщений вверх по потоку.
func (r *DefaiultReceiver) Acks() chan<- UpstreamAck {
	return r.acks
}

// AckUpstreams передает в upstreams ack сообщения.
func (r *DefaiultReceiver) ackUpstreams(ack UpstreamAck) {
	r.upstreamInWorkMutex.Lock()
	defer r.upstreamInWorkMutex.Unlock()

	ackWG := &sync.WaitGroup{}
	for upstreamIndex, messageID := range ack {
		upstreamName, ok := r.upstreamInWorkIndexes[upstreamIndex]
		if !ok {
			continue
		}
		upstream, ok := r.upstreamInWork[upstreamName]
		if !ok {
			continue
		}

		ackWG.Add(1)
		go func(upstream *UpstreamReceiver, messageID uint32) {
			defer ackWG.Done()

			if err := upstream.Ack(messageID); err != nil {
				logs.Logger.Errorf("can not send ACK %d to upstream %s: %s", messageID, upstream.name, err)
			}
		}(upstream.upstream, messageID)
	}

	ackWG.Wait()
	logs.Logger.Debugf("receiverd: sending %d acks done", len(ack))
}
