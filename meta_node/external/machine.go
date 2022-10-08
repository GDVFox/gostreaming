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

// SendRunAction отправляет запрос
func (m *MachineClinets) SendRunAction(ctx context.Context, node *planner.NodePlan) error {
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
		Name:     node.Name,
		Action:   node.Action,
		Port:     node.Port,
		Replicas: node.Replicas,
		In:       node.In,
		Out:      node.Out,
	}
	reqBodyEncoded, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	resp, err := http.Post(machineURL.String(), "application/json", bytes.NewReader(reqBodyEncoded))
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
