package upstreambackup

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/GDVFox/ctxio"
	"github.com/GDVFox/gostreaming/util"
	"github.com/GDVFox/gostreaming/util/connutil"
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

// DefaultReceiver получает сообщения от вышестоящих узлов.
type DefaultReceiver struct {
	upstreamIndex uint16

	acks     chan UpstreamAck
	messages chan *UpstreamMessage
	addr     string

	upstreamInWorkMutex   sync.Mutex
	upstreamInWork        map[string]*workingUpstream
	upstreamInWorkIndexes map[uint16]string

	upstreamNames map[string]struct{}

	logger *util.Logger
}

// NewDefaultReceiver возвращает новый объект DefaultReceiver.
func NewDefaultReceiver(addr string, inNames []string, l *util.Logger) *DefaultReceiver {
	upstreamNames := make(map[string]struct{})
	for _, in := range inNames {
		upstreamNames[in] = struct{}{}
	}

	return &DefaultReceiver{
		upstreamIndex:         0,
		addr:                  addr,
		acks:                  make(chan UpstreamAck),
		messages:              make(chan *UpstreamMessage),
		upstreamNames:         upstreamNames,
		upstreamInWork:        make(map[string]*workingUpstream),
		upstreamInWorkIndexes: make(map[uint16]string),
		logger:                l.WithName("default_receiver"),
	}
}

// Run запускает обработчик.
func (r *DefaultReceiver) Run(ctx context.Context) error {
	defer r.logger.Info("receiver stopped")
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
			r.logger.Errorf("can not close listener: %s", err)
			return
		}
		r.logger.Info("listener closed")
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
			r.logger.Infof("accepted connection %s", conn.RemoteAddr())

			newUpstreamIndex := r.upstreamIndex
			r.upstreamIndex++

			receiverWG.Add(1)
			go func() {
				defer receiverWG.Done()
				r.runUpstream(receiveCtx, newUpstreamIndex, connutil.NewDefaultConnection(conn))
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

	r.logger.Infof("waiting for new connections")
	return wg.Wait()
}

func (r *DefaultReceiver) runUpstream(ctx context.Context, upstreamIndex uint16, tcpConn *connutil.Connection) {
	defer tcpConn.Close()

	upstreamCtx, upstreamStop := context.WithCancel(ctx)
	defer upstreamStop()

	upstreamName, err := r.listenHello(upstreamCtx, tcpConn)
	if err != nil {
		r.logger.Errorf("read hello message failed: %s", err)
		return
	}
	r.logger.Infof("got hello message from %s: %s", tcpConn.Conn.RemoteAddr(), upstreamName)

	r.upstreamInWorkMutex.Lock()
	// Здесь мы можем попасть на уже завершенный upstream,
	// но это ок, так как ничего не меняется от отправки сигнала.
	if workingUpstream, ok := r.upstreamInWork[upstreamName]; ok {
		workingUpstream.stopUpstream()
		delete(r.upstreamInWork, upstreamName)
		delete(r.upstreamInWorkIndexes, workingUpstream.upstream.upstreamIndex)

		r.logger.Debugf("send stop signal to previous upstream %s", upstreamName)
	}

	upstream := NewUpstreamReceiver(upstreamIndex, upstreamName, tcpConn, r.logger)
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
		defer r.logger.Infof("stopped upstream %s", upstream.name)

		if err := upstream.Run(upstreamCtx); err != nil {
			r.logger.Errorf("upstream %s stopped: %s", upstream.name, err)
		}
	}()
	go func() {
		defer wg.Done()
		defer r.logger.Infof("receiver: stopped read message loop for upstream %s", upstream.name)

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

func (r *DefaultReceiver) listenHello(ctx context.Context, tcpConn *connutil.Connection) (string, error) {
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
func (r *DefaultReceiver) Messages() <-chan *UpstreamMessage {
	return r.messages
}

// Acks возвращает канал для передачи Ack сообщений вверх по потоку.
func (r *DefaultReceiver) Acks() chan<- UpstreamAck {
	return r.acks
}

// AckUpstreams передает в upstreams ack сообщения.
func (r *DefaultReceiver) ackUpstreams(ack UpstreamAck) {
	defer r.logger.Debugf("sending %d acks done", len(ack))

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
				r.logger.Errorf("can not send ACK %d to upstream %s: %s", messageID, upstream.name, err)
			}
		}(upstream.upstream, messageID)
	}

	ackWG.Wait()
}
