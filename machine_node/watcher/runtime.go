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
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/GDVFox/gostreaming/util"
	"github.com/GDVFox/gostreaming/util/connutil"
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

	Timeout       time.Duration
	AckPeriod     time.Duration
	ForwardLogDir string
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
	serviceConn     *connutil.Connection

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
		"--ack-period="+r.opt.AckPeriod.String(),
		"--buffer-dir="+path.Join(r.opt.ForwardLogDir, r.Name(), util.RandString(16)),
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
		r.logger.Errorf("runtime started, but not responsing, stopping runtime: %s", err)
		if err := r.Stop(); err != nil {
			return fmt.Errorf("can not stop failed runtime: %w", err)
		}

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
		connConfig := &connutil.ConnectionConfig{
			DialTimeout:  r.opt.Timeout,
			ReadTimeout:  r.opt.Timeout,
			WriteTimeout: r.opt.Timeout,
		}
		conn, err := net.DialTimeout("unix", r.serviceSockPath, connConfig.DialTimeout)
		if err != nil {
			return err
		}
		r.serviceConn = connutil.NewDefaultConnection(conn)

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

	if err := binary.Write(r.serviceConn, binary.BigEndian, ChangeOutCommand); err != nil {
		return fmt.Errorf("can not send change out command: %w", err)
	}
	if err := r.writeChangeOutAddr(oldOut); err != nil {
		return fmt.Errorf("can not send change out old address: %w", err)
	}
	if err := r.writeChangeOutAddr(newOut); err != nil {
		return fmt.Errorf("can not send change out new address: %w", err)
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

func (r *Runtime) writeChangeOutAddr(addr string) error {
	if err := binary.Write(r.serviceConn, binary.BigEndian, uint64(len(addr))); err != nil {
		return fmt.Errorf("can not send change out length: %w", err)
	}
	if err := binary.Write(r.serviceConn, binary.BigEndian, []byte(addr)); err != nil {
		return fmt.Errorf("can not send change out addr: %w", err)
	}
	return nil
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

	procErrs := make(chan error, 1)
	go func() {
		procErrs <- r.waitProcess()
	}()

	select {
	case <-time.After(r.opt.Timeout):
		r.logger.Warn("wait process timeout sending SIGKILL")
		if err := r.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
		if err := <-procErrs; err != nil {
			return err
		}
		return nil
	case err := <-procErrs:
		r.logger.Info("successfully stopped process")
		return err
	}
}

func (r *Runtime) waitProcess() error {
	state, err := r.cmd.Process.Wait()
	if err != nil {
		return fmt.Errorf("can not wait and of runtime process: %w", err)
	}

	// state.ExitCode() == -1 process killed
	if state.Success() || state.ExitCode() == -1 {
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
