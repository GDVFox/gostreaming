package main

import (
	"fmt"

	"github.com/pterm/pterm"
	flag "github.com/spf13/pflag"
)

var (
	addFS     = flag.NewFlagSet("add", flag.ContinueOnError)
	addTicker string
	addLots   int
)

func init() {
	addFS.StringVarP(&addTicker, "ticker", "t", "", "Ticker of stock")
	addFS.IntVarP(&addLots, "lots", "l", 0, "Number of new lots")
	addFS.BoolVarP(&help, "help", "h", false, "Shows help message")
}

// PrintAddHelp печатает сообщение с помощью по команде add
func PrintAddHelp() {
	pterm.DefaultBasicText.Println("Command 'tcs-accout add' add new lots for ticker.")
	pterm.DefaultBasicText.Println("Flags:")
	addFS.PrintDefaults()
}

// HandleAdd добавление в портфель
func HandleAdd(client *tcsClient, args []string) {
	if err := addFS.Parse(args); err != nil {
		pterm.Error.Printfln("Can not parse args: %s", err)
		PrintAddHelp()
		return
	}
	if help {
		PrintAddHelp()
		return
	}
	if addTicker == "" {
		pterm.Error.Printfln("Ticker can not be empty")
		PrintAddHelp()
		return
	}

	loadSpinner, _ := pterm.DefaultSpinner.Start("Adding lots...")
	wasLots, err := client.addStock(addTicker, addLots)
	if err != nil {
		loadSpinner.Fail("Can not add lots: ", err)
		return
	}
	loadSpinner.Success(fmt.Sprintf("Lots added: old %d, new %d!", wasLots, wasLots+addLots))
}
