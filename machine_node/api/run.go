package api

import (
	"encoding/json"
	"net/http"

	"github.com/GDVFox/gostreaming/machine_node/external"
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

	_, err := external.ETCD.LoadAction(r.Context(), req.Action)
	if err != nil {
		if errors.Cause(err) == storage.ErrNotFound {
			return httplib.NewNotFoundResponse(httplib.NewErrorBody(NoActionErrorCode, err.Error())), nil
		}
		return httplib.NewInternalErrorResponse(httplib.NewErrorBody(ETCDErrorCode, err.Error())), nil
	}

	logger.Debugf("Starting %s", req.Name)
	logger.Debugf("Action %s loaded", req.Action)
	logger.Debugf("Port %d", req.Port)
	logger.Debugf("Replicas: %d", req.Replicas)
	logger.Debugf("In: %v", req.In)
	logger.Debugf("Out: %v", req.Out)

	return httplib.NewOKResponse(nil, false), nil
}
