package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/GDVFox/gostreaming/machine_node/config"
	"github.com/GDVFox/gostreaming/machine_node/external"
	"github.com/GDVFox/gostreaming/machine_node/watcher"
	"github.com/GDVFox/gostreaming/util"
	"github.com/GDVFox/gostreaming/util/httplib"
	"github.com/GDVFox/gostreaming/util/message"
	"github.com/GDVFox/gostreaming/util/storage"
	"github.com/pkg/errors"
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

	opt := &watcher.RuntimeOptions{
		Port:             req.Port,
		In:               req.In,
		Out:              req.Out,
		RuntimePath:      config.Conf.Runtime.BinaryPath,
		RuntimeLogsDir:   config.Conf.Runtime.LogsDir,
		RuntimeLogsLevel: config.Conf.Runtime.LogsLevel,
		ActionStartRetry: config.Conf.Runtime.ActionStartRetry,
		Timeout:          time.Duration(config.Conf.Runtime.Timeout),
		AckPeriod:        time.Duration(config.Conf.Runtime.AckPeriod),
		ForwardLogDir:    config.Conf.Runtime.ForwardLogDir,
		ActionOptions: &watcher.ActionOptions{
			Args: req.Args,
			Env:  req.Env,
		},
	}
	runtime := watcher.NewRuntime(req.SchemeName, req.ActionName, actionBytes, logger, opt)

	if err := watcher.RuntimeWatcher.StartRuntime(r.Context(), runtime); err != nil {
		return httplib.NewInternalErrorResponse(httplib.NewErrorBody(InternalError, err.Error())), nil
	}
	logger.Debugf("runtime '%s' started", runtime.Name())

	logger.Infof("started action '%s' from scheme '%s'", req.ActionName, req.SchemeName)
	return httplib.NewOKResponse(nil, httplib.ContentTypeRaw), nil
}
