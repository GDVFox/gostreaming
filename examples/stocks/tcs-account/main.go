package main

import (
	"context"
	"fmt"
	"os"

	"github.com/GDVFox/gostreaming/examples/stocks/util"

	sdk "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	"github.com/pterm/pterm"
)

type tcsClient struct {
	client *sdk.SandboxRestClient
	acc    sdk.Account

	stocks map[string]*util.StockDescription
}

func newTCSClient(token string) (*tcsClient, error) {
	client := sdk.NewSandboxRestClient(token)

	accounts, err := client.Accounts(context.Background())
	if err != nil {
		return nil, err
	}

	stocksFIGI, err := util.LoadStocks(token)
	if err != nil {
		return nil, err
	}

	stocks := make(map[string]*util.StockDescription, len(stocksFIGI))
	for _, stock := range stocksFIGI {
		stocks[stock.Ticker] = stock
	}

	return &tcsClient{
		client: client,
		acc:    accounts[0],
		stocks: stocks,
	}, nil
}

func (c *tcsClient) addStock(ticker string, lots int) (int, error) {
	stock, ok := c.stocks[ticker]
	if !ok {
		return 0, fmt.Errorf("unknown ticker %s", ticker)
	}

	postions, err := c.getPortfolio()
	if err != nil {
		return 0, err
	}

	currentLots := 0
	if postion, ok := postions[stock.FIGI]; ok {
		currentLots = postion.Lots
	}

	return currentLots, c.client.SetPositionsBalance(context.Background(), c.acc.ID, stock.FIGI, float64(currentLots+lots))
}

func (c *tcsClient) rmStock(ticker string, lots int) (int, error) {
	stock, ok := c.stocks[ticker]
	if !ok {
		return 0, fmt.Errorf("unknown ticker %s", ticker)
	}

	postions, err := c.getPortfolio()
	if err != nil {
		return 0, err
	}

	currentLots := 0
	if postion, ok := postions[stock.FIGI]; ok {
		currentLots = postion.Lots
	}

	if currentLots-lots < 0 {
		return 0, fmt.Errorf("available only %d lots, can not delete %d", currentLots, lots)
	}

	return currentLots, c.client.SetPositionsBalance(context.Background(), c.acc.ID, stock.FIGI, float64(currentLots-lots))
}

func (c *tcsClient) getPortfolio() (map[string]sdk.PositionBalance, error) {
	portfolio, err := c.client.Portfolio(context.Background(), sdk.DefaultAccount)
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}

	postions := make(map[string]sdk.PositionBalance, 0)
	for _, pos := range portfolio.Positions {
		postions[pos.FIGI] = pos
	}

	return postions, nil
}

func main() {
	pterm.DisableDebugMessages()
	pterm.Error.ShowLineNumber = false

	if len(os.Args) < 2 {
		HandleHelp()
		return
	}

	token := os.Getenv("TCS_TOK")
	client, err := newTCSClient(token)
	if err != nil {
		fmt.Println(err)
		return
	}

	switch os.Args[1] {
	case "list":
		HandleList(client)
	case "add":
		HandleAdd(client, os.Args[1:])
	case "rm":
		HandleRm(client, os.Args[1:])
	case "help":
		HandleHelp()
	default:
		HandleHelp()
	}
}
