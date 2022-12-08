package main

import (
	"encoding/json"

	"github.com/GDVFox/gostreaming/examples/stocks/util"
	"github.com/GDVFox/gostreaming/lib/go-actionlib"
	"github.com/jmoiron/sqlx"
	"github.com/kelseyhightower/envconfig"
)

// Config contains config info
type Config struct {
	ClickhouseDSN string `envconfig:"clickhouse"`
	TCSToken      string `envconfig:"token"`
}

// OpenDatabase opens connection with clickhouse
func OpenDatabase(dsn string) (*sqlx.DB, error) {
	return sqlx.Connect("clickhouse", dsn)
}

func main() {
	var conf Config
	err := envconfig.Process("stocksupdate", &conf)
	if err != nil {
		actionlib.WriteFatal(err)
	}

	stocksFIGI, err := util.LoadStocks(conf.TCSToken)
	if err != nil {
		actionlib.WriteFatal(err)
	}

	db, err := OpenDatabase(conf.ClickhouseDSN)
	if err != nil {
		actionlib.WriteFatal(err)
	}

	for {
		stockBin, err := actionlib.ReadMessage()
		if err != nil {
			actionlib.WriteFatal(err)
		}

		stock := &util.StockMessage{}
		if err := json.Unmarshal(stockBin, stock); err != nil {
			actionlib.WriteError(err)

			actionlib.AckMessage()
			continue
		}

		ticker := stocksFIGI[stock.FIGI].Ticker
		if err := insertPriceChange(db, stock, ticker); err != nil {
			actionlib.WriteError(err)

			actionlib.AckMessage()
			continue
		}

		if err := actionlib.AckMessage(); err != nil {
			actionlib.WriteFatal(err)
		}
	}
}
