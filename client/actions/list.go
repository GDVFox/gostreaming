package actions

import (
	flag "github.com/spf13/pflag"

	"github.com/GDVFox/gostreaming/client/metaclient"
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
	pterm.DefaultBasicText.Printfln("Command 'gostreaming %s actions list' returns list of available actions.", metaclient.MetaNodeAddress)
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

	loadSpinner, _ := pterm.DefaultSpinner.Start("Loading actions list...")
	actionsList, err := metaclient.MetaNode.GetActionsList()
	if err != nil {
		loadSpinner.Fail("Can not load actions list: ", err)
		return
	}
	loadSpinner.Success("Actions loaded:")
	pterm.Println()

	items := make([]pterm.BulletListItem, 0)
	for _, action := range actionsList.Actions {
		item := pterm.BulletListItem{
			Level:       0,
			Text:        action,
			Bullet:      ">",
			BulletStyle: pterm.NewStyle(pterm.FgYellow),
		}
		items = append(items, item)
	}

	pterm.DefaultBulletList.WithItems(items).Render()
}
