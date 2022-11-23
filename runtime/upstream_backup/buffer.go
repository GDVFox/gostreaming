package upstreambackup

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/syndtr/goleveldb/leveldb"
)

var (
	errBufferEmpty = errors.New("buffer is empty")
)

type logBuffer struct {
	lock sync.Mutex

	dataDir string
	db      *leveldb.DB

	front uint64
	tail  uint64
	size  int64
}

func newLogBuffer(dataDir string) (*logBuffer, error) {
	db, err := leveldb.OpenFile(dataDir, nil)
	if err != nil {
		return nil, fmt.Errorf("can not open underlying db: %w", err)
	}

	return &logBuffer{
		dataDir: dataDir,
		db:      db,
		front:   0,
		tail:    0,
		size:    0,
	}, nil
}

func (b *logBuffer) Append(item *forwardLogItem) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	key := uint64Key(atomic.LoadUint64(&b.tail))
	value := &bytes.Buffer{}
	if err := item.writeOut(value); err != nil {
		return fmt.Errorf("can not encode item: %w", err)
	}

	if err := b.db.Put(key, value.Bytes(), nil); err != nil {
		return fmt.Errorf("can not save item: %w", err)
	}

	atomic.AddUint64(&b.tail, 1)
	atomic.AddInt64(&b.size, 1)
	return nil
}

func (b *logBuffer) LoadFirst(item *forwardLogItem) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if atomic.LoadInt64(&b.size) == 0 {
		return errBufferEmpty
	}

	key := uint64Key(atomic.LoadUint64(&b.front))
	value, err := b.db.Get(key, nil)
	if err != nil {
		return fmt.Errorf("can not read item: %w", err)
	}

	if err := item.readIn(bytes.NewReader(value)); err != nil {
		return fmt.Errorf("can not decode item: %w", err)
	}

	return nil
}

func (b *logBuffer) TrimFirst() error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if atomic.LoadInt64(&b.size) == 0 {
		return errBufferEmpty
	}

	key := uint64Key(atomic.LoadUint64(&b.tail))
	if err := b.db.Delete(key, nil); err != nil {
		return fmt.Errorf("can not trim item: %w", err)
	}

	atomic.AddUint64(&b.front, 1)
	atomic.AddInt64(&b.size, -1)
	return nil
}

func (b *logBuffer) Size() int64 {
	b.lock.Lock()
	defer b.lock.Unlock()

	return atomic.LoadInt64(&b.size)
}

func (b *logBuffer) NewIterator() *LogBufferIterator {
	b.lock.Lock()
	defer b.lock.Unlock()

	return NewLogBufferIterator(b)
}

func (b *logBuffer) Close() error {
	defer os.RemoveAll(b.dataDir)
	return b.db.Close()
}

func uint64Key(k uint64) []byte {
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, k)
	return key
}
