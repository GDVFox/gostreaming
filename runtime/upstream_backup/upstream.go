package upstreambackup

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/GDVFox/ctxio"
	"github.com/GDVFox/gostreaming/util"
	"github.com/GDVFox/gostreaming/util/connutil"
	"golang.org/x/sync/errgroup"
)

var (
	// ErrNoEnoughtData возвращается, когда длина полученного сообщения меньше,
	// чем переданная в заголовке.
	ErrNoEnoughtData = errors.New("too small data")
)

// UpstreamMessage сообщение из вышестоящего узла.
type UpstreamMessage struct {
	*dataMessage
	InputID uint16
}

// DummyUpstreamMessage пустое сообщение из upstream,
// используется для поддержки источников.
var DummyUpstreamMessage = &UpstreamMessage{
	dataMessage: &dataMessage{},
}

// UpstreamReceiver структура, для получения сообщений от узлов выше по потоку.
type UpstreamReceiver struct {
	upstreamIndex uint16
	name          string

	conn       *connutil.Connection
	connWriter *ctxio.ContextWriter

	output chan *UpstreamMessage
	logger *util.Logger
}

// NewUpstreamReceiver создает новый UpstreamReceiver.
func NewUpstreamReceiver(upstreamIndex uint16, name string, tcpConn *connutil.Connection, l *util.Logger) *UpstreamReceiver {
	return &UpstreamReceiver{
		upstreamIndex: upstreamIndex,
		name:          name,
		conn:          tcpConn,
		output:        make(chan *UpstreamMessage),
		logger:        l.WithName("upstream_receiver " + name),
	}
}

// Run запускает UpstreamReceiver и блокируется.
func (r *UpstreamReceiver) Run(ctx context.Context) error {
	defer r.logger.Info("upstream stopped")
	defer close(r.output)

	wg, upstreamCtx := errgroup.WithContext(ctx)
	r.connWriter = ctxio.NewContextWriter(upstreamCtx, r.conn)
	defer r.connWriter.Close()

	wg.Go(func() error {
		err := r.receivingLoop(upstreamCtx)
		return err
	})

	return wg.Wait()
}

func (r *UpstreamReceiver) receivingLoop(ctx context.Context) error {
	defer r.logger.Info("receiving loop done")

	connReader := ctxio.NewContextReader(ctx, r.conn)
	defer connReader.Close()

	for {
		msg := &UpstreamMessage{
			dataMessage: &dataMessage{},
			InputID:     r.upstreamIndex,
		}

		if err := msg.dataMessage.readIn(connReader); err != nil {
			return fmt.Errorf("can not read message: %w", err)
		}

		select {
		case <-ctx.Done():
			return nil
		case r.output <- msg:
		}
	}
}

// Ack передает ACK сообщение вверх по потку.
func (r *UpstreamReceiver) Ack(ack uint32) error {
	if err := binary.Write(r.connWriter, binary.BigEndian, ack); err != nil {
		return fmt.Errorf("can not send ack %d: %w", ack, err)
	}
	return nil
}
