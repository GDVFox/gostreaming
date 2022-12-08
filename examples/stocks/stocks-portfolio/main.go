package main

import (
	"encoding/json"

	"github.com/GDVFox/gostreaming/examples/stocks/util"

	"github.com/GDVFox/gostreaming/lib/go-actionlib"
	sdk "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	"github.com/kelseyhightower/envconfig"
)

// Config contains config info
type Config struct {
	TCSToken string `envconfig:"token"`
}

func main() {
	var conf Config
	err := envconfig.Process("stocksportfolio", &conf)
	if err != nil {
		actionlib.WriteFatal(err)
	}

	client := sdk.NewSandboxRestClient(conf.TCSToken)
	positions := newPositions(client)
	currency, err := newCurrencyWatcher(conf.TCSToken)
	if err != nil {
		actionlib.WriteFatal(err)
	}

	go positions.updateLoop()
	go func() {
		if err := currency.updateLoop(); err != nil {
			actionlib.WriteFatal(err)
		}
	}()

	for {
		stockBin, err := actionlib.ReadMessage()
		if err != nil {
			actionlib.WriteFatal(err)
			continue
		}

		stock := &util.StockMessage{}
		if err := json.Unmarshal(stockBin, stock); err != nil {
			actionlib.WriteError(err)

			actionlib.AckMessage()
			continue
		}

		// пропускаем акции не из портфеля.
		balance, ok := positions.getPosition(stock.FIGI)
		if !ok {
			actionlib.AckMessage()
			continue
		}

		switch stock.Currency {
		case "USD":
			stock.ClosePriceUSD *= float64(balance.Lots)
			stock.ClosePriceRUB = stock.ClosePriceUSD * currency.getRubsForDollar()
		case "RUB":
			stock.ClosePriceRUB *= float64(balance.Lots)
			stock.ClosePriceUSD = stock.ClosePriceRUB / currency.getRubsForDollar()
		}

		stockBin, err = json.Marshal(stock)
		if err != nil {
			actionlib.WriteError(err)

			actionlib.AckMessage()
			continue
		}

		if err := actionlib.WriteMessage(stockBin); err != nil {
			actionlib.WriteFatal(err)
			continue
		}
	}
}
