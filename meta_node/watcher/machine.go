package watcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/GDVFox/gostreaming/meta_node/planner"
	"github.com/GDVFox/gostreaming/util"
	"github.com/GDVFox/gostreaming/util/httplib"
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
	// Host хост, на котором машина ожидает подключения.
	Host string `yaml:"host"`
	// Post сервисный порт машины, который используется
	// для передачи команд от meta_node и возврата статистики выше.
	Port int `yaml:"port"`
	// Timeout время, по истечению которого в случае отсутствия ответа
	// машина признается не работающей и начинается процесс восстановления.
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
	addr := cfg.Host + ":" + strconv.Itoa(cfg.Port)
	return &Machine{
		client: &http.Client{
			Timeout: time.Duration(cfg.Timeout),
		},
		addr:   addr,
		cfg:    cfg,
		logger: l.WithName("machine " + addr),
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
	defer m.logger.Infof("sended run action '%s' for plan '%s'", node.Name, schemeName)

	machineURL := &url.URL{
		Scheme: runHTTPScheme,
		Host:   m.addr,
		Path:   runPath,
	}
	reqBody := &message.RunActionRequest{
		SchemeName:    schemeName,
		ActionName:    node.Name,
		Action:        node.Action,
		Port:          node.Port,
		In:            node.In,
		Out:           node.Out,
		Args:          node.Args,
		Env:           node.Env,
		ConnWhitelist: node.ConnWhitelist,
	}
	return m.sendCommand(machineURL.String(), reqBody)
}

// SendStopAction отправляет запрос для остановки действия на машине.
func (m *Machine) SendStopAction(ctx context.Context, schemeName string, node *planner.NodePlan) error {
	defer m.logger.Infof("sended run stop '%s' for plan '%s'", node.Name, schemeName)

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
	defer m.logger.Infof("sended change out for action '%s' for plan '%s'", actionName, schemeName)

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

	machineError := &httplib.ErrorBody{}
	if err := json.NewDecoder(resp.Body).Decode(machineError); err != nil {
		return fmt.Errorf("can not decode error response %s: %w", err.Error(), ErrMachineError)
	}
	return fmt.Errorf("machine error %s: %w", machineError.Message, ErrMachineError)
}
