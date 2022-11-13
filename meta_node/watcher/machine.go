package watcher

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/GDVFox/gostreaming/meta_node/planner"
	"github.com/GDVFox/gostreaming/util"
	"github.com/GDVFox/gostreaming/util/message"
	"github.com/pkg/errors"
)

// Возможные ошибки
var (
	ErrNoAction     = errors.New("no action")
	ErrMachineError = errors.New("machine internal error")
)

var (
	runHTTPScheme = "http"
	pingPath      = "/v1/ping"
	runPath       = "/v1/run"
	stopPath      = "/v1/stop"
	changeOutPath = "/v1/change_out"
)

// MachineConfig настройки машины, на котором запущен machine_node
type MachineConfig struct {
	Host    string        `yaml:"host"`
	Port    int           `yaml:"port"`
	Timeout util.Duration `yaml:"timeout"`
}

// Machine абстракция машины
type Machine struct {
	client *http.Client

	addr   string
	cfg    *MachineConfig
	logger *util.Logger
}

// NewMachine содает новый объект Machine.
func NewMachine(l *util.Logger, cfg *MachineConfig) *Machine {
	return &Machine{
		client: &http.Client{
			Timeout: time.Duration(cfg.Timeout),
		},
		addr:   cfg.Host + ":" + strconv.Itoa(cfg.Port),
		cfg:    cfg,
		logger: l,
	}
}

// Ping возвращает список работающих действий, запущенных на машине.
func (m *Machine) Ping() ([]*message.RuntimeTelemetry, error) {
	machineURL := &url.URL{
		Scheme: runHTTPScheme,
		Host:   m.addr,
		Path:   pingPath,
	}

	resp, err := m.client.Get(machineURL.String())
	if err != nil {
		return nil, errors.Wrap(ErrMachineError, err.Error())
	}

	telemetry := make([]*message.RuntimeTelemetry, 0)
	if err := json.NewDecoder(resp.Body).Decode(&telemetry); err != nil {
		return nil, errors.Wrap(ErrMachineError, err.Error())
	}

	return telemetry, nil
}

// SendRunAction отправляет запрос для запуска действия на машине.
func (m *Machine) SendRunAction(ctx context.Context, schemeName string, node *planner.NodePlan) error {
	machineURL := &url.URL{
		Scheme: runHTTPScheme,
		Host:   m.addr,
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
func (m *Machine) SendStopAction(ctx context.Context, schemeName string, node *planner.NodePlan) error {
	machineURL := &url.URL{
		Scheme: runHTTPScheme,
		Host:   m.addr,
		Path:   stopPath,
	}
	reqBody := &message.StopActionRequest{
		SchemeName: schemeName,
		ActionName: node.Name,
	}
	return m.sendCommand(machineURL.String(), reqBody)
}

// SendChangeOut отправляет запрос на изменение Out у действия.
func (m *Machine) SendChangeOut(ctx context.Context, schemeName, actionName, oldOut, newOut string) error {
	machineURL := &url.URL{
		Scheme: runHTTPScheme,
		Host:   m.addr,
		Path:   changeOutPath,
	}
	reqBody := &message.ChangeOutRequest{
		SchemeName: schemeName,
		ActionName: actionName,
		OldOut:     oldOut,
		NewOut:     newOut,
	}
	return m.sendCommand(machineURL.String(), reqBody)
}

func (m *Machine) sendCommand(url string, cmd interface{}) error {
	reqBodyEncoded, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	resp, err := m.client.Post(url, "application/json", bytes.NewReader(reqBodyEncoded))
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
