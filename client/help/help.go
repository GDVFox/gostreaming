package help

import "github.com/pterm/pterm"

// HandleHelp выводит сообщение с помощью.
func HandleHelp() {
	pterm.DisableColor()
	pterm.DefaultBasicText.Printfln("Usage: gostreaming ADDRESS CATERGORY COMMAND [OPTIONS]")
	pterm.Println()
	pterm.DefaultBasicText.Printfln("Distributed dataflow managing tool")
	pterm.Println()
	pterm.Println("ADDRESS means address of meta_node in format <host>[:port]")

	pterm.DefaultTable.WithHasHeader().WithData(pterm.TableData{
		{"CATEGORY", "COMMAND", "Description"},
		{"schemas", "", "Managing a list of schemas"},
		{"", "list", "Returns list of available schemas"},
		{"", "get", "Returns description of specified scheme"},
		{"", "new", "Creates new scheme using given description file"},
		{"", "rm", "Removes specified scheme"},
		{"", "run", "Runs specified scheme using saved description"},
		{"", "stop", "Stops specified scheme"},
		{"actions", "", "Managing a list of actions"},
		{"", "list", "Returns list of available actions"},
		{"", "get", "Returns binary file of specified action"},
		{"", "new", "Loads action binary file for use when declaring schemas"},
		{"", "rm", "Removes action binary file"},
		{"help", "", "Prints help message"},
	}).Render()
	pterm.Println()
	pterm.DefaultBasicText.Printfln("Use 'gostreaming ADDRESS CATERGORY COMMAND --help' to see command [OPTIONS]")

	pterm.EnableColor()
}
