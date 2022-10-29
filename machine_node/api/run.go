package api

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/GDVFox/gostreaming/machine_node/config"
	"github.com/GDVFox/gostreaming/machine_node/external"
	"github.com/GDVFox/gostreaming/util"
	"github.com/GDVFox/gostreaming/util/httplib"
	"github.com/GDVFox/gostreaming/util/message"
	"github.com/GDVFox/gostreaming/util/storage"
	"github.com/pkg/errors"
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

// RunAction запускает action.
func RunAction(r *http.Request) (*httplib.Response, error) {
	logger := r.Context().Value(httplib.RequestLogger).(*util.Logger)

	req := &message.RunActionRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return httplib.NewBadRequestResponse(httplib.NewErrorBody(BadUnmarshalRequestErrorCode, err.Error())), nil
	}

	actionBytes, err := external.ETCD.LoadAction(r.Context(), req.Action)
	if err != nil {
		if errors.Cause(err) == storage.ErrNotFound {
			return httplib.NewNotFoundResponse(httplib.NewErrorBody(NoActionErrorCode, err.Error())), nil
		}
		return httplib.NewInternalErrorResponse(httplib.NewErrorBody(ETCDErrorCode, err.Error())), nil
	}
	logger.Debugf("binary action '%s' received", req.Action)

	actionPath, err := createTmpBinary(req.Name, actionBytes)
	if err != nil {
		return httplib.NewInternalErrorResponse(httplib.NewErrorBody(InternalError, err.Error())), nil
	}
	logger.Debugf("created tmp binary with path %s", actionPath)

	if err := startAction(r.Context(), actionPath, req, logger); err != nil {
		return httplib.NewInternalErrorResponse(httplib.NewErrorBody(InternalError, err.Error())), nil
	}
	logger.Debug("action started")

	return httplib.NewOKResponse(nil, false), nil
}

func createTmpBinary(name string, bin []byte) (string, error) {
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

func startAction(ctx context.Context, actionPath string, req *message.RunActionRequest, logger *util.Logger) error {
	socketAddr := filepath.Join("/", "tmp", req.Name+strconv.Itoa(req.Port)+".sock")
	logFileAddr := filepath.Join("/", config.Conf.RuntimeLogsDir, req.Name+strconv.Itoa(req.Port)+".log")
	runtimeStartCommand := exec.Command(
		config.Conf.RuntimePath,
		"--action="+actionPath,
		"--replicas="+strconv.Itoa(req.Replicas),
		"--port="+strconv.Itoa(req.Port),
		"--service-sock="+socketAddr,
		"--log-file="+logFileAddr,
		"--log-level=debug",
		"--in="+strings.Join(req.In, ","),
		"--out="+strings.Join(req.Out, ","),
	)

	errorsOutput, err := runtimeStartCommand.StderrPipe()
	if err != nil {
		return fmt.Errorf("can not get stderr connect: %w", err)
	}

	if err := runtimeStartCommand.Start(); err != nil {
		return fmt.Errorf("can not start command: %w", err)
	}
	logger.Debugf("runtime started with command: %s", runtimeStartCommand.String())

	dialErr := util.Retry(ctx, config.Conf.ActionStartRetry, func() error {
		conn, err := net.Dial("unix", socketAddr)
		if err != nil {
			return err
		}
		defer conn.Close()

		if err := binary.Write(conn, binary.BigEndian, PingCommand); err != nil {
			return err
		}
		logger.Debug("ping sended")

		var resp uint8
		if err := binary.Read(conn, binary.BigEndian, &resp); err != nil {
			return err
		}

		if resp != OKResponse {
			return errors.New("expected OK responce")
		}
		logger.Debug("start confirm received")
		return nil
	})
	if dialErr != nil {
		if runtimeStartCommand.Process == nil {
			return dialErr
		}

		if err := runtimeStartCommand.Process.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("can not send SIGTERM to runtime: %w", err)
		}

		state, err := runtimeStartCommand.Process.Wait()
		if err != nil {
			return fmt.Errorf("can not wait and of runtime process: %w", err)
		}

		// с процессом все хорошо, дело было в dial.
		if state.Success() {
			return dialErr
		}

		b := &bytes.Buffer{}
		if _, err := io.Copy(b, errorsOutput); err != nil {
			return fmt.Errorf("can not copy stderr: %w", err)
		}

		return errors.New(b.String())
	}
	return nil
}
