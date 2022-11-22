package watcher

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/GDVFox/gostreaming/util"
)

const (
	// PingCommand команда для проверки состояния runtime.
	PingCommand uint8 = 0x1
	// ChangeOutCommand команда для изменения выходного потока.
	ChangeOutCommand uint8 = 0x2
)

const (
	// OKResponse ответ, предполагающий успешное выполнение действия.
	OKResponse uint8 = 0x0
	// FailResponse ответ, предполагающий ошибочное выполнение действия.
	FailResponse uint8 = 0x1
)

// Возможные ошибки
var (
	ErrCommandFailed = errors.New("command returned not OK response")
	ErrBadOut        = errors.New("address must be in format <host>:<port>")
)

type changeOutRequest struct {
	OldIP   uint32
	NewIP   uint32
	OldPort uint16
	NewPort uint16
}

// RuntimeTelemetry информация о состоянии runtime.
type RuntimeTelemetry struct {
	OldestOutput uint32
}

// ActionOptions опции для запуска действия
type ActionOptions struct {
	Args []string          `json:"args"`
	Env  map[string]string `json:"env"`
}

// RuntimeOptions набор параметров при запуске действия.
type RuntimeOptions struct {
	Port          int
	Replicas      int
	In            []string
	Out           []string
	ActionOptions *ActionOptions

	RuntimePath      string
	RuntimeLogsDir   string
	RuntimeLogsLevel string
	ActionStartRetry *util.RetryConfig
}

// Runtime структура, представляющая собой запущенное действие
type Runtime struct {
	communicationMutex sync.Mutex

	name       string
	schemeName string
	actionName string

	bin []byte
	opt *RuntimeOptions

	binPath         string
	cmd             *exec.Cmd
	stderr          io.ReadCloser
	serviceSockPath string
	serviceConn     net.Conn

	logger *util.Logger
}

// NewRuntime создает новое действие.
func NewRuntime(schemeName, actionName string, bin []byte, l *util.Logger, opt *RuntimeOptions) *Runtime {
	runtimeName := buildRuntimeName(schemeName, actionName)
	return &Runtime{
		name:       runtimeName,
		schemeName: schemeName,
		actionName: actionName,
		bin:        bin,
		opt:        opt,
		logger:     l.WithName("runtime " + runtimeName),
	}
}

// Name возвращает имя запущенного действия
func (r *Runtime) Name() string {
	return r.name
}

// SchemeName возвращает имя схемы, частью которой является рантайм.
func (r *Runtime) SchemeName() string {
	return r.schemeName
}

// ActionName возвращает имя действия, частью которого является рантайм.
func (r *Runtime) ActionName() string {
	return r.actionName
}

// Start запускает действие и выходит, в случае успешного запуска.
func (r *Runtime) Start(ctx context.Context) error {
	var err error
	r.binPath, err = r.createTmpBinary(r.name, r.bin)
	if err != nil {
		return fmt.Errorf("can not create binary: %w", err)
	}
	r.logger.Infof("created tmp binary with path %s", r.binPath)

	actionOptions, err := json.Marshal(r.opt.ActionOptions)
	if err != nil {
		return fmt.Errorf("invalid action options: %w", err)
	}

	r.serviceSockPath = filepath.Join("/", "tmp", r.name+strconv.Itoa(r.opt.Port)+".sock")
	logFileAddr := filepath.Join("/", r.opt.RuntimeLogsDir, r.name+strconv.Itoa(r.opt.Port)+".log")
	r.cmd = exec.Command(
		r.opt.RuntimePath,
		"--name="+r.Name(),
		"--action="+r.binPath,
		"--replicas="+strconv.Itoa(r.opt.Replicas),
		"--port="+strconv.Itoa(r.opt.Port),
		"--service-sock="+r.serviceSockPath,
		"--log-file="+logFileAddr,
		"--log-level="+r.opt.RuntimeLogsLevel,
		"--in="+strings.Join(r.opt.In, ","),
		"--out="+strings.Join(r.opt.Out, ","),
		"--action-opt="+string(actionOptions),
	)

	r.stderr, err = r.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("can not get stderr connect: %w", err)
	}

	if err := r.cmd.Start(); err != nil {
		return fmt.Errorf("can not start action: %w", err)
	}
	r.logger.Infof("runtime started with command: %s", r.cmd.String())

	if err := r.connect(ctx); err != nil {
		return fmt.Errorf("can not connect to action: %w", err)
	}
	r.logger.Info("service connection created")

	return nil
}

func (r *Runtime) createTmpBinary(name string, bin []byte) (string, error) {
	actionFile, err := ioutil.TempFile("", name)
	if err != nil {
		return "", fmt.Errorf("can not create tmp file for bin: %w", err)
	}
	defer actionFile.Close()

	if _, err := actionFile.Write(bin); err != nil {
		return "", fmt.Errorf("can not writer bin: %w", err)
	}

	if err := os.Chmod(actionFile.Name(), 0700); err != nil {
		return "", fmt.Errorf("can not change file mod: %w", err)
	}

	return actionFile.Name(), nil
}

func (r *Runtime) connect(ctx context.Context) error {
	dialErr := util.Retry(ctx, r.opt.ActionStartRetry, func() error {
		var err error
		r.serviceConn, err = net.Dial("unix", r.serviceSockPath)
		if err != nil {
			return err
		}

		if _, err := r.Ping(); err != nil {
			return err
		}

		return nil
	})
	if dialErr != nil {
		if r.cmd.Process == nil {
			return dialErr
		}

		if err := r.Stop(); err != nil {
			return err
		}
		return dialErr
	}
	return nil
}

// Ping проверяет работоспособность действия с помощью отправки ping.
func (r *Runtime) Ping() (*RuntimeTelemetry, error) {
	r.communicationMutex.Lock()
	defer r.communicationMutex.Unlock()

	if err := binary.Write(r.serviceConn, binary.BigEndian, PingCommand); err != nil {
		return nil, err
	}

	var resp uint8
	if err := binary.Read(r.serviceConn, binary.BigEndian, &resp); err != nil {
		return nil, err
	}

	if resp != OKResponse {
		return nil, ErrCommandFailed
	}

	telemetry := &RuntimeTelemetry{}
	if err := binary.Read(r.serviceConn, binary.BigEndian, telemetry); err != nil {
		return nil, err
	}

	return telemetry, nil
}

// ChangeOut заменяет oldOut на newOut.
func (r *Runtime) ChangeOut(oldOut, newOut string) error {
	r.communicationMutex.Lock()
	defer r.communicationMutex.Unlock()

	oldIP, oldPort, err := r.parseAddr(oldOut)
	if err != nil {
		return fmt.Errorf("can not parse old out %s: %w", oldOut, err)
	}

	newIP, newPort, err := r.parseAddr(newOut)
	if err != nil {
		return fmt.Errorf("can not parse new out %s: %w", newOut, err)
	}

	if err := binary.Write(r.serviceConn, binary.BigEndian, ChangeOutCommand); err != nil {
		return fmt.Errorf("can not send change out command: %w", err)
	}

	req := changeOutRequest{
		OldIP:   oldIP,
		OldPort: oldPort,
		NewIP:   newIP,
		NewPort: newPort,
	}
	if err := binary.Write(r.serviceConn, binary.BigEndian, req); err != nil {
		return fmt.Errorf("can not send change out request: %w", err)
	}

	var resp uint8
	if err := binary.Read(r.serviceConn, binary.BigEndian, &resp); err != nil {
		return err
	}

	if resp != OKResponse {
		return ErrCommandFailed
	}
	return nil
}

func (r *Runtime) parseAddr(addr string) (uint32, uint16, error) {
	addrParts := strings.Split(addr, ":")
	if len(addrParts) != 2 {
		return 0, 0, fmt.Errorf("can not split %s: %w", addr, ErrBadOut)
	}

	ipPart := net.ParseIP(addrParts[0])
	if ipPart == nil {
		return 0, 0, fmt.Errorf("can not parse IP %s: %w", addrParts[0], ErrBadOut)
	}
	ipPart = ipPart.To4()

	portPart, err := strconv.ParseUint(addrParts[1], 10, 16)
	if err != nil {
		return 0, 0, fmt.Errorf("can not parse PORT %s: %w", addrParts[0], ErrBadOut)
	}

	return binary.BigEndian.Uint32(ipPart), uint16(portPart), nil
}

// Stop завершает работу действия, возвращает ошибку из stderr.
func (r *Runtime) Stop() error {
	defer os.Remove(r.binPath)
	defer os.Remove(r.serviceSockPath)
	defer func() {
		if r.stderr != nil {
			r.stderr.Close()
		}
	}()
	defer func() {
		if r.serviceConn != nil {
			r.serviceConn.Close()
		}
	}()

	if err := r.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("can not send SIGTERM to runtime: %w", err)
	}
	r.logger.Info("SIGTERM sended")

	state, err := r.cmd.Process.Wait()
	if err != nil {
		return fmt.Errorf("can not wait and of runtime process: %w", err)
	}
	r.logger.Info("Process stopped")

	// успешное завершение процесса
	if state.Success() {
		return nil
	}

	b := &bytes.Buffer{}
	if _, err := io.Copy(b, r.stderr); err != nil {
		return fmt.Errorf("can not copy stderr: %w", err)
	}

	return errors.New(b.String())
}

func buildRuntimeName(schemeName, actionName string) string {
	return schemeName + "_" + actionName
}
