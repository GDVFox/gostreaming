package util

import (
	"context"

	"github.com/GDVFox/gostreaming/lib/go-actionlib"
	sdk "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
)

// StockDescription описание акции.
type StockDescription struct {
	FIGI     string
	Ticker   string
	Name     string
	Currency string
}

// LoadStocks загрузка описаний акций.
func LoadStocks(token string) (map[string]*StockDescription, error) {
	restClient := sdk.NewSandboxRestClient(token)
	tcsStocks, err := restClient.Stocks(context.Background())
	if err != nil {
		actionlib.WriteFatal(err)
	}

	stocks := make(map[string]*StockDescription, len(tcsStocks))
	for _, stock := range tcsStocks {
		description := &StockDescription{
			FIGI:     stock.FIGI,
			Ticker:   stock.Ticker,
			Name:     stock.Name,
			Currency: string(stock.Currency),
		}
		stocks[stock.FIGI] = description
	}
	return stocks, nil
}

// StockMessage сообщение об обновлении цены акции
type StockMessage struct {
	FIGI          string  `json:"figi"`
	TS            uint64  `json:"ts"`
	Currency      string  `json:"currency"`
	ClosePriceUSD float64 `json:"close_price_usd"`
	ClosePriceRUB float64 `json:"close_price_rub"`
}
