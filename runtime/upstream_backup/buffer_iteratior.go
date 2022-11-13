package upstreambackup

import (
	"bytes"
	"context"
	"fmt"
	"sync/atomic"
)

// LogBufferIterator итератор для прямого передвижения по logBuffer.
type LogBufferIterator struct {
	wasStarted bool
	lastKey    uint64

	logBuffer *logBuffer
}

// NewLogBufferIterator содает новый LogBufferIterator, чтение начинается с начала лога.
func NewLogBufferIterator(logBuffer *logBuffer) *LogBufferIterator {
	return &LogBufferIterator{
		wasStarted: false,
		lastKey:    0,
		logBuffer:  logBuffer,
	}
}

// Next загружает в item следющий элемент.
// В случае, если итератор находится в конце лога, блокируется в ожидании новых записей.
func (i *LogBufferIterator) Next(ctx context.Context, item *forwardLogItem) error {
	if !i.wasStarted {
		i.wasStarted = true
		i.lastKey = atomic.LoadUint64(&i.logBuffer.front)
	}

	for atomic.LoadUint64(&i.logBuffer.tail) == i.lastKey {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	key := uint64Key(i.lastKey)
	value, err := i.logBuffer.db.Get(key, nil)
	if err != nil {
		return fmt.Errorf("can not read item: %w", err)
	}

	if err := item.readIn(bytes.NewReader(value)); err != nil {
		return fmt.Errorf("can not decode item: %w", err)
	}

	i.lastKey++
	return nil
}
