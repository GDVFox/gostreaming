package schemas

import (
	flag "github.com/spf13/pflag"

	"github.com/GDVFox/gostreaming/gostreaming/metaclient"
	"github.com/pterm/pterm"
)

// ListCommandHelper получение списка схем.
type ListCommandHelper struct {
	fs *flag.FlagSet

	help bool
}

// NewListCommandHelper возвращает новый ListCommandHelper.
func NewListCommandHelper() *ListCommandHelper {
	c := &ListCommandHelper{
		fs: flag.NewFlagSet("list", flag.ContinueOnError),
	}

	c.fs.BoolVarP(&c.help, "help", "h", false, "Prints help message")
	return c
}

// Init инициализирует состояние команды.
func (c *ListCommandHelper) Init(args []string) error {
	return c.fs.Parse(args)
}

// PrintHelp печатает сообщение с помощью по команде
func (c *ListCommandHelper) PrintHelp() {
	pterm.DefaultBasicText.Printfln("Command 'gostreaming %s schemas list' returns list of available schemas.", metaclient.MetaNodeAddress)
	pterm.Println()
	pterm.DefaultBasicText.Println("Flags:")
	c.fs.PrintDefaults()
}

// Run запускает комнаду.
func (c *ListCommandHelper) Run() {
	if c.help {
		c.PrintHelp()
		return
	}

	loadSpinner, _ := pterm.DefaultSpinner.Start("Loading schemas list...")
	schemasList, err := metaclient.MetaNode.GetSchemasList()
	if err != nil {
		loadSpinner.Fail("Can not load schemas list: ", err)
		return
	}

	loadSpinner.Success("Schemas loaded:")
	pterm.Println()

	for _, scheme := range schemasList.Schemas {
		if scheme.Status == 0 {
			stoppedPrinter.Printfln(scheme.Name)
		} else if scheme.Status == 1 {
			runningPrinter.Printfln(scheme.Name)
		}
	}
}
