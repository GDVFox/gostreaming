package main

import (
	"strconv"

	"github.com/pterm/pterm"
)

// HandleList вывод списка акций в портфеле
func HandleList(client *tcsClient) {
	stocks, err := client.getPortfolio()
	if err != nil {
		pterm.Error.Printfln("Get portfolio error: %s", err)
		return
	}

	table := pterm.TableData{{"FIGI", "Ticker", "Lots"}}
	for _, stock := range stocks {
		table = append(table, []string{stock.FIGI, stock.Ticker, strconv.Itoa(stock.Lots)})
	}

	pterm.DefaultBasicText.Println("Portfolio:")
	pterm.DefaultTable.WithHasHeader().WithData(table).Render()
	pterm.Println()
}
