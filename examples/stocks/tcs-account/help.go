package main

import "github.com/pterm/pterm"

var (
	help bool
)

// HandleHelp выводит сообщение с помощью.
func HandleHelp() {
	pterm.DisableColor()
	pterm.DefaultBasicText.Printfln("Usage: tcs-accout COMMAND [OPTIONS]")
	pterm.Println()
	pterm.DefaultBasicText.Printfln("Sandbox portfolio management")
	pterm.Println()

	pterm.DefaultTable.WithHasHeader().WithData(pterm.TableData{
		{"COMMAND", "Description"},
		{"list", "Shows positions"},
		{"add", "Adds N stocks"},
		{"rm", "Remove N stocks"},
		{"help", "", "Prints help message"},
	}).Render()
	pterm.Println()
	pterm.DefaultBasicText.Printfln("Use 'tcs-accout COMMAND --help' to see command [OPTIONS]")

	pterm.EnableColor()
}
