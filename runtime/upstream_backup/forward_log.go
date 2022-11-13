package upstreambackup

import (
	"fmt"
)

// ForwardLog лог для записи сообщений с целью обеспечения отказоустойчивости.
type ForwardLog struct {
	buffer *logBuffer
}

// NewForwardLog создает новый ForwardLog.
func NewForwardLog(forwardLogDir string) (*ForwardLog, error) {
	buff, err := newLogBuffer(forwardLogDir)
	if err != nil {
		return nil, err
	}

	return &ForwardLog{buffer: buff}, nil
}

// NewIterator возвращает итератор, который позволяет двигаться по ForwardLog с первой записи в прямом направлении.
func (l *ForwardLog) NewIterator() *LogBufferIterator {
	return l.buffer.NewIterator()
}

func (l *ForwardLog) Write(inputID uint16, inputMsgID, outputMsgID uint32, data []byte) error {
	fLogItem := forwardLogItems.Get()
	defer forwardLogItems.Put(fLogItem)

	fLogItem.Header.InputID = inputID
	fLogItem.Header.InputMessageID = inputMsgID
	fLogItem.Header.OutputMessageID = outputMsgID
	fLogItem.Header.MessageLength = uint32(len(data))
	fLogItem.Data = data

	if err := l.buffer.Append(fLogItem); err != nil {
		return fmt.Errorf("can not write forward log item: %w", err)
	}
	return nil
}

// Trim отрезает от лога все сообщения, у которых output_id <= idBorder.
// Может обрезать сообщения одновременно с записью, так как никогда не будет обрабатывать
// одно и то же сообщение из-за того, что отправка происходит после записи в лог,
// а значит если мы получили подтвержение на это сообщение, то оно уже было отправлено.
func (l *ForwardLog) Trim(idBorder uint32) (map[uint16]uint32, error) {
	inputMaxs := make(map[uint16]uint32)
	for l.buffer.Size() != 0 {
		fLogItem := forwardLogItems.Get()

		if err := l.buffer.LoadFirst(fLogItem); err != nil {
			forwardLogItems.Put(fLogItem)
			return nil, fmt.Errorf("can not read front buffer header: %w", err)
		}

		if fLogItem.Header.OutputMessageID > idBorder {
			forwardLogItems.Put(fLogItem)
			break
		}

		if err := l.buffer.TrimFirst(); err != nil {
			return nil, fmt.Errorf("can not trim buffer: %w", err)
		}

		// Последовательность строго возрастающая, поэтому можно переприсваивать.
		// Но проверку на корректность буффера полезно сделать для дебага.
		// Случай равенства может быть для источника, так как там все InputMessageID есть 0.
		if msgID, ok := inputMaxs[fLogItem.Header.InputID]; ok && msgID > fLogItem.Header.InputMessageID {
			return nil, fmt.Errorf("for input_id %d got message_id %d expected greater than %d",
				fLogItem.Header.InputID, fLogItem.Header.InputMessageID, msgID)
		}
		inputMaxs[fLogItem.Header.InputID] = fLogItem.Header.InputMessageID
	}

	return inputMaxs, nil
}

// Close закрывает ForwardLog и очищает занимаемые ресурсы.
func (l *ForwardLog) Close() error {
	return l.buffer.Close()
}
