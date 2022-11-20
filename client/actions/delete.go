package actions

import (
	"errors"

	"github.com/GDVFox/gostreaming/client/metaclient"
	"github.com/pterm/pterm"
	flag "github.com/spf13/pflag"
)

// DeleteCommandHelper удаление конкретной схемы.
type DeleteCommandHelper struct {
	fs *flag.FlagSet

	help bool
	name string
}

// NewDeleteCommandHelper создает новый DeleteCommandHelper
func NewDeleteCommandHelper() *DeleteCommandHelper {
	c := &DeleteCommandHelper{
		fs: flag.NewFlagSet("rm", flag.ContinueOnError),
	}

	c.fs.StringVarP(&c.name, "name", "n", "", "Name of the action to remove")
	c.fs.BoolVarP(&c.help, "help", "h", false, "Prints help message")

	return c
}

// PrintHelp печатает сообщение с помощью по команде
func (c *DeleteCommandHelper) PrintHelp() {
	pterm.DefaultBasicText.Printfln("Command 'gostreaming %s actions rm' removes action binary file.", metaclient.MetaNodeAddress)
	pterm.Println()
	pterm.DefaultBasicText.Println("Flags:")
	c.fs.PrintDefaults()
}

// Init инициализирует состояние команды.
func (c *DeleteCommandHelper) Init(args []string) error {
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
func (c *DeleteCommandHelper) Run() {
	if c.help {
		c.PrintHelp()
		return
	}

	loadSpinner, _ := pterm.DefaultSpinner.Start("Removing action...")
	if err := metaclient.MetaNode.DeleteAction(c.name); err != nil {
		loadSpinner.Fail("Can not remove action: ", err)
		return
	}
	loadSpinner.Success("Action removed!")
}
