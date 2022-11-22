package schemas

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v3"

	"github.com/GDVFox/gostreaming/gostreaming/metaclient"
	"github.com/pterm/pterm"
)

// GetCommandHelper получение конкретной схемы.
type GetCommandHelper struct {
	fs *flag.FlagSet

	help   bool
	name   string
	format string
	out    string
}

// NewGetCommandHelper создает новый GetCommandHelper
func NewGetCommandHelper() *GetCommandHelper {
	c := &GetCommandHelper{
		fs: flag.NewFlagSet("get", flag.ContinueOnError),
	}

	c.fs.StringVarP(&c.name, "name", "n", "", "Name of the scheme to load")
	c.fs.StringVarP(&c.format, "format", "f", "yaml", "Format of output, possible values: json, yaml")
	c.fs.StringVarP(&c.out, "out", "o", "", "Output file")
	c.fs.BoolVarP(&c.help, "help", "h", false, "Prints help message")

	return c
}

// PrintHelp печатает сообщение с помощью по команде
func (c *GetCommandHelper) PrintHelp() {
	pterm.DefaultBasicText.Printfln("Command 'gostreaming %s schemas get' returns description of specified scheme.", metaclient.MetaNodeAddress)
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

	if c.format != jsonExt && c.format != yamlExt {
		return fmt.Errorf("format possible values is: json, yaml: got %s", c.format)
	}

	return nil
}

// Run запускает команду
func (c *GetCommandHelper) Run() {
	if c.help {
		c.PrintHelp()
		return
	}

	loadSpinner, _ := pterm.DefaultSpinner.Start("Loading scheme...")
	scheme, err := metaclient.MetaNode.GetScheme(c.name)
	if err != nil {
		loadSpinner.Fail("Can not get scheme: ", err)
		return
	}
	loadSpinner.Success("Scheme loaded!")

	var schemeData []byte
	switch c.format {
	case jsonExt:
		schemeData, err = json.MarshalIndent(scheme, "", "\t")
	case yamlExt:
		schemeData, err = yaml.Marshal(scheme)
	}
	if err != nil {
		pterm.Error.Printfln("Can not marshal scheme data to format %s: %s", c.format, err)
		return
	}

	if c.out == "" {
		pterm.Println()
		pterm.DefaultBasicText.Println(string(schemeData))
	} else {
		f, err := os.Create(c.out)
		if err != nil {
			pterm.Error.Printfln("Can not open output file %s: %s", c.out, err)
			return
		}
		if _, err := f.Write(schemeData); err != nil {
			pterm.Error.Println("Can not write scheme to file %s: %s", c.out, err)
			return
		}
	}
}
