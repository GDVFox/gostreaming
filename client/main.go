package main

import (
	"os"

	"github.com/GDVFox/gostreaming/client/about"
	"github.com/GDVFox/gostreaming/client/actions"
	"github.com/GDVFox/gostreaming/client/help"
	"github.com/GDVFox/gostreaming/client/metaclient"
	"github.com/GDVFox/gostreaming/client/schemas"
	"github.com/pterm/pterm"
)

// Category категория команд
type Category string

// Список возможных категорий.
const (
	SchemasCatergory Category = "schemas"
	ActionsCategory  Category = "actions"
	HelpCategory     Category = "help"
	AboutCategory    Category = "about"
)

func main() {
	pterm.DisableDebugMessages()
	pterm.Error.ShowLineNumber = false

	if len(os.Args) < 2 {
		help.HandleHelp()
		return
	}
	// Для категорий help и about вводить адрес meta_node необязательно.
	switch Category(os.Args[1]) {
	case HelpCategory:
		help.HandleHelp()
		return
	case AboutCategory:
		about.HandleAbout()
		return
	default:
	}

	// С этого момента помимо адреса должна быть указана категория
	if len(os.Args) < 3 {
		help.HandleHelp()
		return
	}

	cfg := &metaclient.MetaNodeClientConfig{Address: os.Args[1]}
	metaclient.OpenMetaNodeClient(cfg)

	args := os.Args[2:]
	switch Category(args[0]) {
	case SchemasCatergory:
		schemas.HandleSchemas(args)
	case ActionsCategory:
		actions.HandleActions(args)
	case HelpCategory:
		help.HandleHelp()
	case AboutCategory:
		about.HandleAbout()
	default:
		pterm.Error.Printfln("Unknown category '%s', run 'gostreaming %s help' for more information", args[0], metaclient.MetaNodeAddress)
	}
}
