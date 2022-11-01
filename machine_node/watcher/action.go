package watcher

import (
	"bytes"
	"context"
	"encoding/binary"
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
	"syscall"

	"github.com/GDVFox/gostreaming/util"
)

const (
	// PingCommand команда для проверки состояния runtime.
	PingCommand uint8 = 0x1
)

const (
	// OKResponse ответ, предполагающий успешное выполнение действия.
	OKResponse uint8 = 0x0
	// FailResponse ответ, предполагающий ошибочное выполнение действия.
	FailResponse uint8 = 0x1
)

// Возможные ошибки
var (
	ErrPingFailed = errors.New("ping returned not OK response")
)

// ActionOptions набор параметров при запуске действия.
type ActionOptions struct {
	Port     int
	Replicas int
	In       []string
	Out      []string

	RuntimePath      string
	RuntimeLogsDir   string
	ActionStartRetry *util.RetryConfig
}

// Action структура, представляющая собой запущенное действие
type Action struct {
	name string
	bin  []byte
	opt  *ActionOptions

	binPath         string
	cmd             *exec.Cmd
	stderr          io.ReadCloser
	serviceSockPath string
	serviceConn     net.Conn

	logger *util.Logger
}

// NewAction создает новое действие.
func NewAction(schemeName, actionName string, bin []byte, l *util.Logger, opt *ActionOptions) *Action {
	return &Action{
		name: buildActionName(schemeName, actionName),
		bin:  bin,
		opt:  opt,

		logger: l,
	}
}

// Name возвращает имя запущенного действия
func (a *Action) Name() string {
	return a.name
}

// Start запускает действие и выходит, в случае успешного запуска.
func (a *Action) Start(ctx context.Context) error {
	var err error
	a.binPath, err = a.createTmpBinary(a.name, a.bin)
	if err != nil {
		return fmt.Errorf("can not create binary: %w", err)
	}
	a.logger.Debugf("created tmp binary with path %s", a.binPath)

	a.serviceSockPath = filepath.Join("/", "tmp", a.name+strconv.Itoa(a.opt.Port)+".sock")
	logFileAddr := filepath.Join("/", a.opt.RuntimeLogsDir, a.name+strconv.Itoa(a.opt.Port)+".log")
	a.cmd = exec.Command(
		a.opt.RuntimePath,
		"--action="+a.binPath,
		"--replicas="+strconv.Itoa(a.opt.Replicas),
		"--port="+strconv.Itoa(a.opt.Port),
		"--service-sock="+a.serviceSockPath,
		"--log-file="+logFileAddr,
		"--log-level=debug",
		"--in="+strings.Join(a.opt.In, ","),
		"--out="+strings.Join(a.opt.Out, ","),
	)

	a.stderr, err = a.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("can not get stderr connect: %w", err)
	}

	if err := a.cmd.Start(); err != nil {
		return fmt.Errorf("can not start action: %w", err)
	}
	a.logger.Debugf("runtime started with command: %s", a.cmd.String())

	if err := a.connect(ctx); err != nil {
		return fmt.Errorf("can not connect to action: %w", err)
	}

	return nil
}

func (a *Action) createTmpBinary(name string, bin []byte) (string, error) {
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

func (a *Action) connect(ctx context.Context) error {
	dialErr := util.Retry(ctx, a.opt.ActionStartRetry, func() error {
		var err error
		a.serviceConn, err = net.Dial("unix", a.serviceSockPath)
		if err != nil {
			return err
		}

		if err := a.Ping(); err != nil {
			return err
		}

		a.logger.Debug("start confirm received")
		return nil
	})
	if dialErr != nil {
		if a.cmd.Process == nil {
			return dialErr
		}

		if err := a.Stop(); err != nil {
			return err
		}
		return dialErr
	}
	return nil
}

// Ping проверяет работоспособность действия с помощью отправки ping.
func (a *Action) Ping() error {
	if err := binary.Write(a.serviceConn, binary.BigEndian, PingCommand); err != nil {
		return err
	}

	var resp uint8
	if err := binary.Read(a.serviceConn, binary.BigEndian, &resp); err != nil {
		return err
	}

	if resp != OKResponse {
		return ErrPingFailed
	}
	return nil
}

// Stop завершает работу действия, возвращает ошибку из stderr.
func (a *Action) Stop() error {
	defer os.Remove(a.binPath)
	defer os.Remove(a.serviceSockPath)
	defer a.stderr.Close()
	defer a.serviceConn.Close()

	if err := a.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("can not send SIGTERM to runtime: %w", err)
	}

	state, err := a.cmd.Process.Wait()
	if err != nil {
		return fmt.Errorf("can not wait and of runtime process: %w", err)
	}

	// успешное завершение процесса
	if state.Success() {
		return nil
	}

	b := &bytes.Buffer{}
	if _, err := io.Copy(b, a.stderr); err != nil {
		return fmt.Errorf("can not copy stderr: %w", err)
	}

	return errors.New(b.String())
}

func buildActionName(schemeName, actionName string) string {
	return schemeName + "_" + actionName
}
