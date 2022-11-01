package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/GDVFox/gostreaming/meta_node/config"
	"github.com/GDVFox/gostreaming/meta_node/planner"
	"github.com/GDVFox/gostreaming/util/message"
	"github.com/pkg/errors"
)

// Возможные ошибки
var (
	ErrNoAction     = errors.New("no action")
	ErrNoHost       = errors.New("unknown machine host")
	ErrMachineError = errors.New("machine internal error")
)

var (
	runHTTPScheme = "http"
	runPath       = "/v1/run"
	stopPath      = "/v1/stop"
)

// MachineClinets абстракция для обращения к машинам.
type MachineClinets struct {
	hosts map[string]int
}

// NewMachineClinets создает объект Machines
func NewMachineClinets(cfg []*config.Machine) (*MachineClinets, error) {
	hosts := make(map[string]int, len(cfg))
	for _, machine := range cfg {
		if _, ok := hosts[machine.Host]; ok {
			return nil, fmt.Errorf("duplicate host: %s", machine.Host)
		}
		hosts[machine.Host] = machine.Port
	}

	return &MachineClinets{hosts: hosts}, nil
}

// SendRunAction отправляет запрос для запуска действия на машине.
func (m *MachineClinets) SendRunAction(ctx context.Context, schemeName string, node *planner.NodePlan) error {
	port, ok := m.hosts[node.Host]
	if !ok {
		return ErrNoHost
	}

	machineURL := &url.URL{
		Scheme: runHTTPScheme,
		Host:   node.Host + ":" + strconv.Itoa(port),
		Path:   runPath,
	}
	reqBody := &message.RunActionRequest{
		SchemeName: schemeName,
		ActionName: node.Name,
		Action:     node.Action,
		Port:       node.Port,
		In:         node.In,
		Out:        node.Out,
		Args:       node.Args,
		Env:        node.Env,
	}
	return m.sendCommand(machineURL.String(), reqBody)
}

// SendStopAction отправляет запрос для остановки действия на машине.
func (m *MachineClinets) SendStopAction(ctx context.Context, schemeName string, node *planner.NodePlan) error {
	port, ok := m.hosts[node.Host]
	if !ok {
		return ErrNoHost
	}

	machineURL := &url.URL{
		Scheme: runHTTPScheme,
		Host:   node.Host + ":" + strconv.Itoa(port),
		Path:   stopPath,
	}
	reqBody := &message.StopActionRequest{
		SchemeName: schemeName,
		ActionName: node.Name,
	}
	return m.sendCommand(machineURL.String(), reqBody)
}

func (m *MachineClinets) sendCommand(url string, cmd interface{}) error {
	reqBodyEncoded, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(reqBodyEncoded))
	if err != nil {
		return errors.Wrap(ErrMachineError, err.Error())
	}

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	if resp.StatusCode == http.StatusNotFound {
		return ErrNoAction
	}
	return ErrMachineError
}
