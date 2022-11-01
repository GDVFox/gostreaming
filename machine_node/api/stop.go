package api

import (
	"encoding/json"
	"net/http"

	"github.com/GDVFox/gostreaming/machine_node/watcher"
	"github.com/GDVFox/gostreaming/util/httplib"
	"github.com/GDVFox/gostreaming/util/message"
)

// StopAction останавливает action.
func StopAction(r *http.Request) (*httplib.Response, error) {
	req := &message.StopActionRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return httplib.NewBadRequestResponse(httplib.NewErrorBody(BadUnmarshalRequestErrorCode, err.Error())), nil
	}

	if err := watcher.RuntimeWatcher.StopRuntime(req.SchemeName, req.ActionName); err != nil {
		if err == watcher.ErrUnknownRuntime {
			return httplib.NewNotFoundResponse(httplib.NewErrorBody(NoActionErrorCode, err.Error())), nil
		}
		return httplib.NewInternalErrorResponse(httplib.NewErrorBody(InternalError, err.Error())), nil
	}

	return httplib.NewOKResponse(nil, false), nil
}
