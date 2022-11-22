package actions

import (
	"errors"
	"os"

	flag "github.com/spf13/pflag"

	"github.com/GDVFox/gostreaming/gostreaming/metaclient"
	"github.com/pterm/pterm"
)

// GetCommandHelper получение конкретной схемы.
type GetCommandHelper struct {
	fs *flag.FlagSet

	help bool
	name string
	out  string
}

// NewGetCommandHelper создает новый GetCommandHelper
func NewGetCommandHelper() *GetCommandHelper {
	c := &GetCommandHelper{
		fs: flag.NewFlagSet("get", flag.ContinueOnError),
	}

	c.fs.StringVarP(&c.name, "name", "n", "", "Name of the actions to load")
	c.fs.StringVarP(&c.out, "out", "o", "", "Output binary file")
	c.fs.BoolVarP(&c.help, "help", "h", false, "Prints help message")

	return c
}

// PrintHelp печатает сообщение с помощью по команде
func (c *GetCommandHelper) PrintHelp() {
	pterm.DefaultBasicText.Printfln("Command 'gostreaming %s actions get' returns binary file of specified action.", metaclient.MetaNodeAddress)
	pterm.Println()
	pterm.DefaultBasicText.Println("Flags:")
	c.fs.PrintDefaults()
}

// Init инициализирует состояние команды.
func (c *GetCommandHelper) Init(args []string) error {
	if err := c.fs.Parse(args); err != nil {
		return err
	}
	if c.help {
		return nil
	}

	if c.name == "" {
		return errors.New("name can not be empty")
	}

	return nil
}

// Run запускает команду
func (c *GetCommandHelper) Run() {
	if c.help {
		c.PrintHelp()
		return
	}

	loadSpinner, _ := pterm.DefaultSpinner.Start("Loading action...")
	actionBinary, err := metaclient.MetaNode.GetAction(c.name)
	if err != nil {
		loadSpinner.Fail("Can not get action: ", err)
		return
	}
	loadSpinner.Success("Action loaded!")

	if c.out == "" {
		pterm.Println()
		pterm.DefaultBasicText.Println(string(actionBinary))
	} else {
		f, err := os.Create(c.out)
		if err != nil {
			pterm.Error.Printfln("Can not open output file %s: %s", c.out, err)
			return
		}
		if _, err := f.Write(actionBinary); err != nil {
			pterm.Error.Println("Can not write binary action to file %s: %s", c.out, err)
			return
		}
	}
}
