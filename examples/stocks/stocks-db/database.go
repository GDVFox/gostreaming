package main

import (
	"github.com/GDVFox/gostreaming/examples/stocks/util"

	"github.com/jmoiron/sqlx"
	_ "github.com/mailru/go-clickhouse"
)

type stockPriceChange struct {
	Timestamp uint64  `db:"timestamp"`
	FIGI      string  `db:"figi"`
	Ticker    string  `db:"ticker"`
	Currency  string  `db:"currency"`
	PriceRUB  float64 `db:"price_rub"`
	PriceUSD  float64 `db:"price_usd"`
}

func insertPriceChange(db *sqlx.DB, stock *util.StockMessage, ticker string) error {
	change := &stockPriceChange{
		Timestamp: stock.TS,
		FIGI:      stock.FIGI,
		Ticker:    ticker,
		Currency:  stock.Currency,
		PriceRUB:  stock.ClosePriceRUB,
		PriceUSD:  stock.ClosePriceUSD,
	}

	_, err := db.NamedExec("INSERT INTO stocks (timestamp, figi, ticker, currency, price_rub, price_usd) "+
		"VALUES (:timestamp, :figi, :ticker, :currency, :price_rub, :price_usd)", change)
	if err != nil {
		return err
	}
	return nil
}
