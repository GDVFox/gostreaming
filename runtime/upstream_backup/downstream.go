package upstreambackup

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/GDVFox/ctxio"
	"github.com/GDVFox/gostreaming/util"
	"github.com/GDVFox/gostreaming/util/connutil"
	"golang.org/x/sync/errgroup"
)

// Список возможных ошибок
var (
	ErrConnectionClosed = errors.New("can not write to closed connection")
)

type downstreamAck struct {
	ackMessage
	DownstreamIndex uint16
}

// DownstreamForwarder клиент для передачи сообщений далее по пайплайну.
type DownstreamForwarder struct {
	writeCtx context.Context

	iter *LogBufferIterator
	acks chan *downstreamAck

	downstreamIndex uint16
	name            string
	addr            string
	logger          *util.Logger
}

// NewDownstreamForwarder создает новый объект DownstreamForwarder.
func NewDownstreamForwarder(downstreamIndex uint16, name string, addr string, iter *LogBufferIterator, l *util.Logger) *DownstreamForwarder {
	return &DownstreamForwarder{
		downstreamIndex: downstreamIndex,
		name:            name,
		addr:            addr,

		iter:   iter,
		acks:   make(chan *downstreamAck),
		logger: l.WithName("downstream_forwarder " + addr),
	}
}

// Run запускает клиент действия и блокируется до окончания работы.
func (f *DownstreamForwarder) Run(ctx context.Context) error {
	defer f.logger.Info("downstream forwarder stopped")
	defer close(f.acks)

	conn, err := net.Dial("tcp", f.addr)
	if err != nil {
		return fmt.Errorf("can not dial tcp: %w", err)
	}
	defer conn.Close()
	tcpConn := connutil.NewDefaultConnection(conn)

	if err := f.sayHello(ctx, tcpConn); err != nil {
		return fmt.Errorf("can not say hello: %w", err)
	}

	wg, downstreamCtx := errgroup.WithContext(ctx)
	f.writeCtx = downstreamCtx
	wg.Go(func() error {
		return f.receivingLoop(downstreamCtx, tcpConn)
	})
	wg.Go(func() error {
		return f.transmitingLoop(downstreamCtx, tcpConn)
	})

	return wg.Wait()
}

func (f *DownstreamForwarder) sayHello(ctx context.Context, conn *connutil.Connection) error {
	connWriter := ctxio.NewContextWriter(ctx, conn)
	defer connWriter.Free()

	hello := &helloMessage{}
	hello.Name = []byte(f.name)
	hello.NameLength = uint32(len(hello.Name))
	if err := hello.writeOut(connWriter); err != nil {
		return fmt.Errorf("can not send hello message: %w", err)
	}

	return nil
}

func (f *DownstreamForwarder) receivingLoop(ctx context.Context, conn *connutil.Connection) error {
	defer f.logger.Info("receiving loop done")

	connReader := ctxio.NewContextReader(ctx, conn)
	defer connReader.Close()

	for {
		ack := &downstreamAck{DownstreamIndex: f.downstreamIndex}
		if err := ack.ackMessage.readIn(connReader); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return nil
		case f.acks <- ack:
		}
	}
}

func (f *DownstreamForwarder) transmitingLoop(ctx context.Context, conn *connutil.Connection) error {
	defer f.logger.Info("transmiting loop done")

	connWriter := ctxio.NewContextWriter(ctx, conn)
	defer connWriter.Close()

	for {
		fLogItem := forwardLogItems.Get()
		if err := f.iter.Next(ctx, fLogItem); err != nil {
			forwardLogItems.Put(fLogItem)
			return fmt.Errorf("can not get next item: %w", err)
		}

		msg := &dataMessage{
			Header: dataMessageHeader{
				MessageID:     fLogItem.Header.OutputMessageID,
				MessageLength: fLogItem.Header.MessageLength,
			},
			Data: fLogItem.Data,
		}
		forwardLogItems.Put(fLogItem)

		if err := msg.writeOut(connWriter); err != nil {
			return fmt.Errorf("can not send message %d: %w", msg.Header.MessageID, err)
		}
	}
}
