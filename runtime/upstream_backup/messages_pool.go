package upstreambackup

import "sync"

var (
	forwardLogItems = forwardLogItemsPool{
		p: sync.Pool{
			New: func() interface{} {
				return new(forwardLogItem)
			},
		},
	}
)

type forwardLogItemsPool struct {
	p sync.Pool
}

func (p *forwardLogItemsPool) Get() *forwardLogItem {
	return p.p.Get().(*forwardLogItem)
}

func (p *forwardLogItemsPool) Put(o *forwardLogItem) {
	o.Header.InputID = 0
	o.Header.Flags = 0
	o.Header.InputMessageID = 0
	o.Header.OutputMessageID = 0
	o.Header.MessageLength = 0
	o.Data = nil

	p.p.Put(o)
}
