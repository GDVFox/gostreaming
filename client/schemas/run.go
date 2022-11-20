package schemas

import (
	"errors"

	"github.com/GDVFox/gostreaming/client/metaclient"
	"github.com/pterm/pterm"
	flag "github.com/spf13/pflag"
)

// RunCommandHelper запуск схемы в работу.
type RunCommandHelper struct {
	fs *flag.FlagSet

	help bool
	name string
}

// NewRunCommandHelper создает новый RunCommandHelper
func NewRunCommandHelper() *RunCommandHelper {
	c := &RunCommandHelper{
		fs: flag.NewFlagSet("run", flag.ContinueOnError),
	}

	c.fs.StringVarP(&c.name, "name", "n", "", "Name of the scheme to run")
	c.fs.BoolVarP(&c.help, "help", "h", false, "Prints help message")

	return c
}

// PrintHelp печатает сообщение с помощью по команде
func (c *RunCommandHelper) PrintHelp() {
	pterm.DefaultBasicText.Printfln("Command 'gostreaming %s schemas run' runs specified scheme using saved description.", metaclient.MetaNodeAddress)
	pterm.Println()
	pterm.DefaultBasicText.Println("Flags:")
	c.fs.PrintDefaults()
}

// Init инициализирует состояние команды.
func (c *RunCommandHelper) Init(args []string) error {
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
func (c *RunCommandHelper) Run() {
	if c.help {
		c.PrintHelp()
		return
	}

	loadSpinner, _ := pterm.DefaultSpinner.Start("Starting scheme...")
	if err := metaclient.MetaNode.RunScheme(c.name); err != nil {
		loadSpinner.Fail("Can not run scheme: ", err)
		return
	}
	loadSpinner.Success("Scheme started!")
}
