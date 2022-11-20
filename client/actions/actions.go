package actions

import (
	"github.com/GDVFox/gostreaming/client/common"
	"github.com/GDVFox/gostreaming/client/metaclient"
	"github.com/pterm/pterm"
)

// Список возможных команд.
const (
	ListCommand   common.Command = "list"
	GetCommand    common.Command = "get"
	CreateCommand common.Command = "new"
	DeleteCommand common.Command = "rm"
)

// HandleActions обрабатывает вызов actions.
func HandleActions(rawArgs []string) {
	if len(rawArgs) < 2 {
		pterm.Error.Printfln("Expected COMMAND, run 'gostreaming %s help' for more information", metaclient.MetaNodeAddress)
		return
	}
	args := rawArgs[1:]

	var commandHelper common.CommandHelper
	switch common.Command(args[0]) {
	case ListCommand:
		commandHelper = NewListCommandHelper()
	case GetCommand:
		commandHelper = NewGetCommandHelper()
	case CreateCommand:
		commandHelper = NewCreateCommandHelper()
	case DeleteCommand:
		commandHelper = NewDeleteCommandHelper()
	default:
		pterm.Error.Printfln("Unknown command '%s', run 'gostreaming %s help' for more information", args[0], metaclient.MetaNodeAddress)
		return
	}

	if err := commandHelper.Init(args); err != nil {
		pterm.Error.Printfln("Can not parse command flags: %s", err)
		pterm.Println()
		commandHelper.PrintHelp()
		return
	}
	commandHelper.Run()
}
