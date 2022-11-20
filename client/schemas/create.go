package schemas

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/GDVFox/gostreaming/client/metaclient"
	"github.com/GDVFox/gostreaming/meta_node/planner"
	"github.com/pterm/pterm"
	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

// CreateCommandHelper создание новой схемы.
type CreateCommandHelper struct {
	fs *flag.FlagSet

	help bool
	file string
}

// NewCreateCommandHelper создает новый CreateCommandHelper
func NewCreateCommandHelper() *CreateCommandHelper {
	c := &CreateCommandHelper{
		fs: flag.NewFlagSet("new", flag.ContinueOnError),
	}

	c.fs.StringVarP(&c.file, "file", "f", "", "File with scheme in json or yaml format")
	c.fs.BoolVarP(&c.help, "help", "h", false, "Prints help message")

	return c
}

// PrintHelp печатает сообщение с помощью по команде
func (c *CreateCommandHelper) PrintHelp() {
	pterm.DefaultBasicText.Printfln("Command 'gostreaming %s schemas new' creates new scheme using given description file.", metaclient.MetaNodeAddress)
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

	if c.file == "" {
		return errors.New("file can not be empty")
	}

	fileExt := filepath.Ext(c.file)
	if fileExt != "."+jsonExt && fileExt != "."+yamlExt {
		return fmt.Errorf("possible file formats is: json, yaml: got %s", fileExt)
	}

	return nil
}

// Run запускает команду
func (c *CreateCommandHelper) Run() {
	if c.help {
		c.PrintHelp()
		return
	}

	fileExt := path.Ext(c.file)[1:]
	schemeData, err := os.ReadFile(c.file)
	if err != nil {
		pterm.Error.Printfln("Can not read scheme from file %s: %s", c.file, err)
		return
	}

	scheme := &planner.Scheme{}
	switch fileExt {
	case jsonExt:
		err = json.Unmarshal(schemeData, scheme)
	case yamlExt:
		err = yaml.Unmarshal(schemeData, scheme)
	}

	loadSpinner, _ := pterm.DefaultSpinner.Start("Creating scheme...")
	if err := metaclient.MetaNode.CreateScheme(scheme); err != nil {
		loadSpinner.Fail("Can not create scheme: ", err)
		return
	}
	loadSpinner.Success("Scheme created!")
}
