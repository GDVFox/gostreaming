package api

import (
	"encoding/json"
	"net/http"

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

	opt := &watcher.ActionOptions{
		Port:             req.Port,
		Replicas:         req.Replicas,
		In:               req.In,
		Out:              req.Out,
		RuntimePath:      config.Conf.RuntimePath,
		RuntimeLogsDir:   config.Conf.RuntimeLogsDir,
		ActionStartRetry: config.Conf.ActionStartRetry,
	}
	action := watcher.NewAction(req.SchemeName, req.ActionName, actionBytes, logger, opt)

	if err := action.Start(r.Context()); err != nil {
		return httplib.NewInternalErrorResponse(httplib.NewErrorBody(InternalError, err.Error())), nil
	}
	logger.Debugf("action '%s' started", action.Name())

	if err := watcher.ActionWatcher.RegisterAction(action); err != nil {
		return httplib.NewInternalErrorResponse(httplib.NewErrorBody(InternalError, err.Error())), nil
	}
	logger.Debugf("action '%s' registered", action.Name())

	logger.Infof("started action '%s' from scheme '%s'", req.ActionName, req.SchemeName)
	return httplib.NewOKResponse(nil, false), nil
}
