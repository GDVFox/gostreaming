package upstreambackup

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/GDVFox/gostreaming/runtime/logs"
)

// Forwarder объект для регистрации сообщений от действия, передачи их дальше по потоку,
// а также по контролю за доставкой сообщений.
type Forwarder interface {
	Run(ctx context.Context) error
	GetOldestOutput() (uint32, error)
	Forward(inputID uint16, inputMsgID uint32, data []byte) error
	ChangeOut(oldOut, newOut string) error
	AckMessages() <-chan UpstreamAck
}

// FakeForwarder специальный Forwarder для действий, которые являются стоками.
type FakeForwarder struct {
	inputMaxsMutex sync.Mutex
	inputMaxs      UpstreamAck

	upstreamAcks chan UpstreamAck
	ackTicker    *time.Ticker
}

// NewFakeForwarder создает новый объект FakeForwarder.
func NewFakeForwarder() *FakeForwarder {
	return &FakeForwarder{
		inputMaxs:    make(UpstreamAck),
		upstreamAcks: make(chan UpstreamAck),
		ackTicker:    time.NewTicker(10 * time.Second), // TODO: config
	}
}

// Run запускает Forwarder и блокируется.
func (f *FakeForwarder) Run(ctx context.Context) error {
	defer close(f.upstreamAcks)
	return f.trimLoop(ctx)
}

// ChangeOut заглушка для изменения out.
func (f *FakeForwarder) ChangeOut(oldOut, newOut string) error {
	return nil
}

// GetOldestOutput всегда возвращает 0, так как forward log не нужен стоку.
func (f *FakeForwarder) GetOldestOutput() (uint32, error) {
	return 0, nil
}

// Forward отправляет сообщение дальше с гарантиями доставки.
func (f *FakeForwarder) Forward(inputID uint16, inputMsgID uint32, data []byte) error {
	f.inputMaxsMutex.Lock()
	defer f.inputMaxsMutex.Unlock()

	inputMax, ok := f.inputMaxs[inputID]
	if ok && inputMax >= inputMsgID {
		return fmt.Errorf("fake_forwarder: got from %d message %d; current max is %d", inputID, inputMsgID, inputMax)
	}

	f.inputMaxs[inputID] = inputMsgID
	logs.Logger.Debugf("fake_forwarder: register message %d from input %d done", inputMsgID, inputID)
	return nil
}

// AckMessages возвращает канал с UpstreamAck для передачи далее по пайплайну.
func (f *FakeForwarder) AckMessages() <-chan UpstreamAck {
	return f.upstreamAcks
}

func (f *FakeForwarder) trimLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-f.ackTicker.C:
			logs.Logger.Debugf("forwarder: starting trim forward log")

			f.inputMaxsMutex.Lock()
			inputMaxs := f.inputMaxs
			f.inputMaxs = make(map[uint16]uint32)
			f.inputMaxsMutex.Unlock()

			if len(inputMaxs) == 0 {
				logs.Logger.Debugf("forwarder: nothing to trim")
				continue
			}

			logs.Logger.Debugf("forwarder: got upstreams for ACK: %s", inputMaxs)

			select {
			case <-ctx.Done():
				return nil
			case f.upstreamAcks <- inputMaxs:
			}
		}
	}
}
