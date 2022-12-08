package main

import (
	"fmt"

	"github.com/pterm/pterm"
	flag "github.com/spf13/pflag"
)

var (
	rmFS     = flag.NewFlagSet("rm", flag.ContinueOnError)
	rmTicker string
	rmLots   int
)

func init() {
	rmFS.StringVarP(&rmTicker, "ticker", "t", "", "Ticker of stock")
	rmFS.IntVarP(&rmLots, "lots", "l", 0, "Number of lots for remove")
	rmFS.BoolVarP(&help, "help", "h", false, "Shows help message")
}

// PrintRmHelp печатает сообщение с помощью по команде rm
func PrintRmHelp() {
	pterm.DefaultBasicText.Println("Command 'tcs-accout rm' removes lots by ticker.")
	pterm.DefaultBasicText.Println("Flags:")
	rmFS.PrintDefaults()
}

// HandleRm удаление ищ портфеля
func HandleRm(client *tcsClient, args []string) {
	if err := rmFS.Parse(args); err != nil {
		pterm.Error.Printfln("Can not parse args: %s", err)
		PrintRmHelp()
		return
	}
	if help {
		PrintRmHelp()
		return
	}
	if rmTicker == "" {
		pterm.Error.Printfln("Ticker can not be empty")
		PrintRmHelp()
		return
	}

	loadSpinner, _ := pterm.DefaultSpinner.Start("Removing lots...")
	wasLots, err := client.rmStock(rmTicker, rmLots)
	if err != nil {
		loadSpinner.Fail("Can not remove lots: ", err)
		return
	}
	loadSpinner.Success(fmt.Sprintf("Lots removed: old %d, new %d!", wasLots, wasLots-rmLots))
}
