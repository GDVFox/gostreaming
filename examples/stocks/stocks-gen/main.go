package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GDVFox/gostreaming/examples/stocks/util"
	"github.com/GDVFox/gostreaming/lib/go-actionlib"
	sdk "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	"github.com/kelseyhightower/envconfig"
)

// Config contains config info
type Config struct {
	TCSToken string `envconfig:"token"`
}

func processCandle(eventTime time.Time, candle *sdk.Candle, currency string) error {
	// пропускаем не рублевые и не долларовые акции.
	if currency != "RUB" && currency != "USD" {
		return actionlib.AckMessage()
	}

	msg := &util.StockMessage{
		FIGI:     candle.FIGI,
		TS:       uint64(eventTime.UnixNano()),
		Currency: currency,
	}

	switch currency {
	case "RUB":
		msg.ClosePriceRUB = candle.ClosePrice
	case "USD":
		msg.ClosePriceUSD = candle.ClosePrice
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return actionlib.WriteMessage(data)
}

func main() {
	var conf Config
	err := envconfig.Process("stocksgen", &conf)
	if err != nil {
		actionlib.WriteFatal(err)
	}

	fakeLogger := &util.FakeLogger{}
	stocksFIGI, err := util.LoadStocks(conf.TCSToken)
	if err != nil {
		actionlib.WriteFatal(err)
	}

	stocks := make(map[string]*util.StockDescription, len(stocksFIGI))
	for _, stock := range stocksFIGI {
		stocks[stock.Ticker] = stock
	}

	client, err := sdk.NewStreamingClient(fakeLogger, conf.TCSToken)
	if err != nil {
		actionlib.WriteFatal(err)
	}
	defer client.Close()

	go func() {
		err = client.RunReadLoop(func(event interface{}) error {
			candleEvent, ok := event.(sdk.CandleEvent)
			if !ok {
				actionlib.WriteError(fmt.Errorf("expected CandleEvent got %#v", event))
				return nil
			}

			currency := stocksFIGI[candleEvent.Candle.FIGI].Currency
			if err := processCandle(candleEvent.Time, &candleEvent.Candle, currency); err != nil {
				actionlib.WriteError(err)
				return nil
			}
			return nil
		})
		if err != nil {
			actionlib.WriteFatal(err)
		}
	}()

	for _, stock := range stocks {
		err = client.SubscribeCandle(stock.FIGI, sdk.CandleInterval1Min, util.RequestID())
		if err != nil {
			actionlib.WriteFatal(err)
		}
	}

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalChannel)
	stopChannel := make(chan struct{})
	go func() {
		defer close(stopChannel)
		<-signalChannel
	}()

	<-stopChannel
}
