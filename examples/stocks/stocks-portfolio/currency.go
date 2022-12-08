package main

import (
	"sync/atomic"

	"github.com/GDVFox/gostreaming/examples/stocks/util"

	sdk "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
)

const (
	mult       = 10000.
	usdrubFIGI = "BBG0013HGFT4"
)

type currencyWatcher struct {
	client *sdk.StreamingClient

	rubsForDollar uint64
	ready         uint32
}

func newCurrencyWatcher(token string) (*currencyWatcher, error) {
	fakeLogger := &util.FakeLogger{}

	client, err := sdk.NewStreamingClient(fakeLogger, token)
	if err != nil {
		return nil, err
	}

	return &currencyWatcher{
		client: client,
	}, nil
}

func (w *currencyWatcher) updateLoop() error {
	err := w.client.SubscribeCandle(usdrubFIGI, sdk.CandleInterval1Min, util.RequestID())
	if err != nil {
		return err
	}
	defer w.client.UnsubscribeCandle(usdrubFIGI, sdk.CandleInterval1Min, util.RequestID())

	return w.client.RunReadLoop(func(event interface{}) error {
		candleEvent, ok := event.(sdk.CandleEvent)
		if !ok {
			return nil
		}

		atomic.StoreUint64(&w.rubsForDollar, uint64(candleEvent.Candle.ClosePrice*mult))
		atomic.StoreUint32(&w.ready, 1)
		return nil
	})
}

func (w *currencyWatcher) getRubsForDollar() float64 {
	for atomic.LoadUint32(&w.ready) != 1 {
	}

	return float64(atomic.LoadUint64(&w.rubsForDollar)) / mult
}
