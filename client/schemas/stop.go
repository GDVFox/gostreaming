package schemas

import (
	"errors"

	"github.com/GDVFox/gostreaming/client/metaclient"
	"github.com/pterm/pterm"
	flag "github.com/spf13/pflag"
)

// StopCommandHelper остановка схемы.
type StopCommandHelper struct {
	fs *flag.FlagSet

	help bool
	name string
}

// NewStopCommandHelper создает новый StopCommandHelper
func NewStopCommandHelper() *StopCommandHelper {
	c := &StopCommandHelper{
		fs: flag.NewFlagSet("stop", flag.ContinueOnError),
	}

	c.fs.StringVarP(&c.name, "name", "n", "", "Name of the scheme to stop")
	c.fs.BoolVarP(&c.help, "help", "h", false, "Prints help message")

	return c
}

// PrintHelp печатает сообщение с помощью по команде
func (c *StopCommandHelper) PrintHelp() {
	pterm.DefaultBasicText.Printfln("Command 'gostreaming %s schemas stop' stops specified scheme.", metaclient.MetaNodeAddress)
	pterm.Println()
	pterm.DefaultBasicText.Println("Flags:")
	c.fs.PrintDefaults()
}

// Init инициализирует состояние команды.
func (c *StopCommandHelper) Init(args []string) error {
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
func (c *StopCommandHelper) Run() {
	if c.help {
		c.PrintHelp()
		return
	}

	loadSpinner, _ := pterm.DefaultSpinner.Start("Stopping scheme...")
	if err := metaclient.MetaNode.StopScheme(c.name); err != nil {
		loadSpinner.Fail("Can not stop scheme: ", err)
		return
	}
	loadSpinner.Success("Scheme stopped!")
}
