package planner

import (
	"strconv"

	"github.com/pkg/errors"
)

// Возможные ошибки проверки схемы
var (
	ErrExpectedNodeName         = errors.New("expected not empty node name")
	ErrExpectedAction           = errors.New("expected not empty action name")
	ErrExpectedAddresses        = errors.New("expected not empty addresses")
	ErrExpectedHost             = errors.New("expected not empty host")
	ErrExpectedPort             = errors.New("expected non-zero port")
	ErrExpectedPositiveReplicas = errors.New("expected replicas count > 0")
	ErrExpectedSchemeName       = errors.New("expected not empty scheme name")
	ErrExpectedDataflow         = errors.New("expected not empty dataflow")
	ErrNodeNameUsed             = errors.New("node name already used")
	ErrNodeAddressUsed          = errors.New("address already used")
	ErrEmptyArg                 = errors.New("arg can not be empty")
	ErrEmptyEnvVarName          = errors.New("env variable name can not be empty")
)

// AddrDescription описание адреса сервера, на котором будет запущено действие
type AddrDescription struct {
	Host string `yaml:"host" json:"host"`
	Port int    `yaml:"port" json:"port"`
}

// NodeDescription описание узла.
type NodeDescription struct {
	Name      string             `yaml:"name" json:"name"`
	Action    string             `yaml:"action" json:"action"`
	Addresses []*AddrDescription `yaml:"addresses" json:"addresses"`
	Args      []string           `yaml:"args" json:"args"`
	Env       map[string]string  `yaml:"env" json:"env"`
}

// Check выполняет проверку правильности описания узла.
func (d *NodeDescription) Check() error {
	if d.Name == "" {
		return ErrExpectedNodeName
	}
	if d.Action == "" {
		return ErrExpectedAction
	}
	if len(d.Addresses) == 0 {
		return ErrExpectedAddresses
	}
	for _, addr := range d.Addresses {
		if addr.Host == "" {
			return ErrExpectedHost
		}
		if addr.Port == 0 {
			return ErrExpectedPort
		}
	}
	for _, arg := range d.Args {
		if arg == "" {
			return ErrEmptyArg
		}
	}
	for name := range d.Env {
		if name == "" {
			return ErrEmptyEnvVarName
		}
		// при этом допускается пустое value, как в unix системах.
	}

	return nil
}

// Scheme схема запуска пайплайна.
// Содержит данные о серверах, а также описание программы.
type Scheme struct {
	Name     string             `yaml:"name" json:"name"`
	Nodes    []*NodeDescription `yaml:"nodes" json:"nodes"`
	Dataflow string             `yaml:"dataflow" json:"dataflow"`
}

// Check выполняет проверку правильности задания схемы.
func (s *Scheme) Check() error {
	if s.Name == "" {
		return ErrExpectedSchemeName
	}

	names := make(map[string]struct{}, 0)
	servers := make(map[string]struct{}, 0)
	for _, node := range s.Nodes {
		if err := node.Check(); err != nil {
			return err
		}
		if _, ok := names[node.Name]; ok {
			return errors.Wrapf(ErrNodeNameUsed, "%s", node.Name)
		}
		names[node.Name] = struct{}{}
		for _, addr := range node.Addresses {
			address := addr.Host + ":" + strconv.Itoa(addr.Port)
			if _, ok := servers[address]; ok {
				return errors.Wrapf(ErrNodeAddressUsed, "%s: %s", node.Name, address)
			}
			servers[address] = struct{}{}
		}
	}

	if s.Dataflow == "" {
		return ErrExpectedDataflow
	}

	return nil
}
