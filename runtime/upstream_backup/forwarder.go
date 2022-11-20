package upstreambackup

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GDVFox/gostreaming/runtime/logs"
	"github.com/GDVFox/gostreaming/util"
	"golang.org/x/sync/errgroup"
)

// Возможные ошибки.
var (
	ErrUnknownOutAddress = errors.New("unknown out address")
)

// UpstreamAck отображение upstream_id в содержание ACK сообщения.
type UpstreamAck map[uint16]uint32

func (a UpstreamAck) String() string {
	var b strings.Builder
	for upstream, ack := range a {
		b.WriteString("upstream_")
		b.WriteString(strconv.Itoa(int(upstream)))
		b.WriteByte('=')
		b.WriteString(strconv.Itoa(int(ack)))
	}
	return b.String()
}

type workingDownstream struct {
	downstream     *DownstreamForwarder
	stopDownstream context.CancelFunc
	done           chan struct{}
}

// DefaultForwarderConfig набор параметров для DefaultForwarder.
type DefaultForwarderConfig struct {
	ACKPeriod        time.Duration
	ForwardLogDir    string
	DownstreamConfig *DownstreamForwarderConfig
}

// DefaultForwarder предает сообщения дальше по потоку,
// при этом обеспечивая отказоустойчивость по схеме upstream_backup.
type DefaultForwarder struct {
	// После того, как uint32 закончится, снова начнется нумерация с 0.
	// Ничего из-за этого не случится, так как все записывается в порядке очереди
	// и, кроме того, в системе не будут существовать 4294967295 одновременно.
	messageIndex uint32
	name         string

	forwardLog *ForwardLog

	ctx          context.Context
	downstreamWG sync.WaitGroup

	downstreamsAcksLock sync.RWMutex
	downstreamsAcks     map[uint16]uint32

	downstreamsInWorkMutex sync.Mutex
	downstreamsInWork      map[uint16]*workingDownstream

	downstreamsIndexesMutex sync.Mutex
	downstreamsIndexes      map[string]uint16

	upstreamAcks chan UpstreamAck
	ackTicker    *time.Ticker

	downstreamConfig *DownstreamForwarderConfig
}

// NewDefaultForwarder создает новый объект DefaultForwarder.
func NewDefaultForwarder(name string, outs []string, cfg *DefaultForwarderConfig) (*DefaultForwarder, error) {
	forwardLog, err := NewForwardLog(cfg.ForwardLogDir + util.RandString(16))
	if err != nil {
		return nil, err
	}

	downstreamsIndexes := make(map[string]uint16)
	for i, out := range outs {
		downstreamsIndexes[out] = uint16(i)
	}

	return &DefaultForwarder{
		messageIndex:       0,
		name:               name,
		forwardLog:         forwardLog,
		downstreamsAcks:    make(map[uint16]uint32),
		downstreamsInWork:  make(map[uint16]*workingDownstream),
		downstreamsIndexes: downstreamsIndexes,
		upstreamAcks:       make(chan UpstreamAck),
		ackTicker:          time.NewTicker(cfg.ACKPeriod),
		downstreamConfig:   cfg.DownstreamConfig,
	}, nil
}

// Run запускает Forwarder и блокируется.
func (f *DefaultForwarder) Run(ctx context.Context) error {
	defer close(f.upstreamAcks)
	defer f.forwardLog.Close()
	defer f.downstreamWG.Wait()

	// Используем тут errgroup, чтобы получить контекст,
	// который будет отменен в случае ошибки в trimLoop.
	forwarderWG, forwarderCtx := errgroup.WithContext(ctx)
	f.ctx = forwarderCtx
	forwarderWG.Go(func() error {
		return f.trimLoop(forwarderCtx)
	})

	f.downstreamWG.Add(len(f.downstreamsIndexes))
	for addr, index := range f.downstreamsIndexes {
		go func(index uint16, addr string) {
			defer f.downstreamWG.Done()
			f.runDownstream(forwarderCtx, index, addr)
		}(index, addr)
	}

	return forwarderWG.Wait()
}

// ChangeOut изменяет выходной поток из oldOut в newOut
func (f *DefaultForwarder) ChangeOut(oldOut, newOut string) error {
	f.downstreamsIndexesMutex.Lock()
	defer f.downstreamsIndexesMutex.Unlock()

	downdstreamIndex, ok := f.downstreamsIndexes[oldOut]
	if !ok {
		return ErrUnknownOutAddress
	}

	f.downstreamsInWorkMutex.Lock()
	if wd, ok := f.downstreamsInWork[downdstreamIndex]; ok {
		wd.stopDownstream()
		<-wd.done
	}
	f.downstreamsInWorkMutex.Unlock()

	f.downstreamWG.Add(1)
	go func() {
		defer f.downstreamWG.Done()
		f.runDownstream(f.ctx, downdstreamIndex, newOut)
	}()

	delete(f.downstreamsIndexes, oldOut)
	f.downstreamsIndexes[newOut] = downdstreamIndex

	logs.Logger.Debugf("forwarder: changed out %s -> %s", oldOut, newOut)
	return nil
}

// GetOldestOutput возвращает самый старый output_message_id, который хранится в логе.
func (f *DefaultForwarder) GetOldestOutput() (uint32, error) {
	return f.forwardLog.GetOldestOutput()
}

func (f *DefaultForwarder) runDownstream(ctx context.Context, downstreamIndex uint16, addr string) {
	downstreamCtx, downstreamStop := context.WithCancel(ctx)
	defer downstreamStop()

	f.downstreamsInWorkMutex.Lock()
	if workingDownstream, ok := f.downstreamsInWork[downstreamIndex]; ok {
		workingDownstream.stopDownstream()
		delete(f.downstreamsInWork, downstreamIndex)

		logs.Logger.Debugf("forwarder: send stop signal to previous downstream %d", downstreamIndex)
	}

	wd := &workingDownstream{
		downstream:     NewDownstreamForwarder(downstreamIndex, f.name, addr, f.forwardLog.NewIterator(), f.downstreamConfig),
		stopDownstream: downstreamStop,
		done:           make(chan struct{}),
	}
	f.downstreamsInWork[downstreamIndex] = wd
	f.downstreamsInWorkMutex.Unlock()

	defer close(wd.done)
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		defer logs.Logger.Debugf("forwarder: stopped downstream %d", wd.downstream.downstreamIndex)

		if err := wd.downstream.Run(downstreamCtx); err != nil {
			logs.Logger.Errorf("forwarder: downstream %d stopped with error: %s", wd.downstream.downstreamIndex, err)
		}
	}()
	go func() {
		defer wg.Done()
		defer logs.Logger.Debugf("forwarder: stopped ACK handle for downstream %d", wd.downstream.downstreamIndex)

		if err := f.handleAcks(downstreamCtx, wd.downstream.acks); err != nil {
			logs.Logger.Errorf("forwarder: downstream %d ACK handle stopped with error: %s", wd.downstream.downstreamIndex, err)
		}
	}()

	wg.Wait()
	return
}

// Forward отправляет сообщение дальше с гарантиями доставки.
func (f *DefaultForwarder) Forward(inputID uint16, inputMsgID uint32, data []byte) error {
	// Всегда увеличиваем счетчик, пропуски в случае ошибок не должны ни на что влиять
	defer func() { f.messageIndex++ }()

	if err := f.forwardLog.Write(inputID, inputMsgID, f.messageIndex, data); err != nil {
		logs.Logger.Errorf("forwarder: write forward log failed for message %d (len %d): %s", f.messageIndex, len(data), err)
		return fmt.Errorf("can not write forward log: %w", err)
	}
	logs.Logger.Debugf("forwarder: forward message %d (len %d) done", f.messageIndex, len(data))
	return nil
}

// AckMessages возвращает канал с UpstreamAck для передачи далее по пайплайну.
func (f *DefaultForwarder) AckMessages() <-chan UpstreamAck {
	return f.upstreamAcks
}

func (f *DefaultForwarder) handleAcks(ctx context.Context, acks chan *downstreamAck) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case ack, ok := <-acks:
			if ack == nil && !ok {
				return nil
			}

			logs.Logger.Debugf("forwarder: got ACK from downstream %d: %d", ack.DownstreamIndex, ack.ackMessage)

			f.downstreamsAcksLock.Lock()
			ackOutputMessageID, ok := f.downstreamsAcks[ack.DownstreamIndex]
			if ok && ackOutputMessageID >= uint32(ack.ackMessage) {
				logs.Logger.Errorf("forwarder: new ack message from %d '%d' is less than saved '%d'",
					ack.DownstreamIndex, ack.ackMessage, ackOutputMessageID)

				f.downstreamsAcksLock.Unlock()
				continue
			}
			f.downstreamsAcks[ack.DownstreamIndex] = uint32(ack.ackMessage)
			f.downstreamsAcksLock.Unlock()

		}
	}
}

func (f *DefaultForwarder) trimLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-f.ackTicker.C:
			logs.Logger.Debugf("forwarder: starting trim forward log")

			inputMaxs, minAck, err := f.trimForwardLog()
			if err != nil {
				return fmt.Errorf("can not trim forward log: %w", err)
			}

			if len(inputMaxs) == 0 {
				logs.Logger.Debugf("forwarder: nothing to trim")
				continue
			}

			logs.Logger.Debugf("forwarder: trim forward log to %d done", minAck)
			logs.Logger.Debugf("forwarder: got upstreams for ACK: %s", inputMaxs)

			select {
			case <-ctx.Done():
				return nil
			case f.upstreamAcks <- inputMaxs:
			}
		}
	}
}

func (f *DefaultForwarder) trimForwardLog() (UpstreamAck, uint32, error) {
	f.downstreamsAcksLock.RLock()
	defer f.downstreamsAcksLock.RUnlock()

	// Возможно, что в начале работы с некоторых нод не успели прийти ack.
	// тогда обрезать что-либо ещё рано.
	if len(f.downstreamsAcks) != len(f.downstreamsIndexes) {
		return nil, 0, nil
	}

	wasMin := false
	minAck := uint32(0)
	for _, ack := range f.downstreamsAcks {
		if !wasMin || minAck < ack {
			wasMin = true
			minAck = ack
		}
	}

	inputMax, err := f.forwardLog.Trim(minAck)
	if err != nil {
		return nil, 0, fmt.Errorf("can not trim log: %w", err)
	}

	return inputMax, minAck, err
}
