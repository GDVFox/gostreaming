package schemas

import (
	"github.com/GDVFox/gostreaming/client/common"
	"github.com/GDVFox/gostreaming/client/metaclient"
	"github.com/pterm/pterm"
)

var (
	runningPrinter = pterm.PrefixPrinter{
		MessageStyle: &pterm.ThemeDefault.DescriptionMessageStyle,
		Prefix: pterm.Prefix{
			Style: &pterm.ThemeDefault.SuccessPrefixStyle,
			Text:  "RUNNING",
		},
	}
	stoppedPrinter = pterm.PrefixPrinter{
		MessageStyle: &pterm.ThemeDefault.DescriptionMessageStyle,
		Prefix: pterm.Prefix{
			Style: &pterm.ThemeDefault.WarningPrefixStyle,
			Text:  "STOPPED",
		},
	}
)

const (
	jsonExt = "json"
	yamlExt = "yaml"
)

// Список возможных команд.
const (
	ListCommand   common.Command = "list"
	GetCommand    common.Command = "get"
	CreateCommand common.Command = "new"
	DeleteCommand common.Command = "rm"
	RunCommand    common.Command = "run"
	StopCommand   common.Command = "stop"
)

// HandleSchemas обрабатывает вызов schemas.
func HandleSchemas(rawArgs []string) {
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
	case RunCommand:
		commandHelper = NewRunCommandHelper()
	case StopCommand:
		commandHelper = NewStopCommandHelper()
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
