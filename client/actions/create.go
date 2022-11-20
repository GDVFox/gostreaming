package actions

import (
	"errors"
	"os"

	"github.com/GDVFox/gostreaming/client/metaclient"
	"github.com/pterm/pterm"
	flag "github.com/spf13/pflag"
)

// CreateCommandHelper создание новой схемы.
type CreateCommandHelper struct {
	fs *flag.FlagSet

	help bool
	name string
	file string
}

// NewCreateCommandHelper создает новый CreateCommandHelper
func NewCreateCommandHelper() *CreateCommandHelper {
	c := &CreateCommandHelper{
		fs: flag.NewFlagSet("new", flag.ContinueOnError),
	}

	c.fs.StringVarP(&c.name, "name", "n", "", "Name of a new action")
	c.fs.StringVarP(&c.file, "file", "f", "", "Binary file with new action")
	c.fs.BoolVarP(&c.help, "help", "h", false, "Prints help message")

	return c
}

// PrintHelp печатает сообщение с помощью по команде
func (c *CreateCommandHelper) PrintHelp() {
	pterm.DefaultBasicText.Printfln("Command 'gostreaming %s action new' loads action binary file for use when declaring schemas.", metaclient.MetaNodeAddress)
	pterm.Println()
	pterm.DefaultBasicText.Println("Flags:")
	c.fs.PrintDefaults()
}

// Init инициализирует состояние команды.
func (c *CreateCommandHelper) Init(args []string) error {
	if err := c.fs.Parse(args); err != nil {
		return err
	}
	if c.help {
		return nil
	}

	if c.name == "" {
		return errors.New("name can not be empty")
	}
	if c.file == "" {
		return errors.New("file can not be empty")
	}
	return nil
}

// Run запускает команду
func (c *CreateCommandHelper) Run() {
	if c.help {
		c.PrintHelp()
		return
	}

	actionBinary, err := os.ReadFile(c.file)
	if err != nil {
		pterm.Error.Printfln("Can not read action binary %s: %s", c.file, err)
		return
	}

	loadSpinner, _ := pterm.DefaultSpinner.Start("Creating action...")
	if err := metaclient.MetaNode.CreateAction(c.name, actionBinary); err != nil {
		loadSpinner.Fail("Can not create action: ", err)
		return
	}
	loadSpinner.Success("Action created!")
}
