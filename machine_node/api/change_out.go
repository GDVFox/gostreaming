package api

import (
	"encoding/json"
	"net/http"

	"github.com/GDVFox/gostreaming/machine_node/watcher"
	"github.com/GDVFox/gostreaming/util"
	"github.com/GDVFox/gostreaming/util/httplib"
	"github.com/GDVFox/gostreaming/util/message"
)

// ChangeActionOut перенапрявляет выходной поток действия в другой узел.
func ChangeActionOut(r *http.Request) (*httplib.Response, error) {
	logger := r.Context().Value(httplib.RequestLogger).(*util.Logger)

	req := &message.ChangeOutRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return httplib.NewBadRequestResponse(httplib.NewErrorBody(BadUnmarshalRequestErrorCode, err.Error())), nil
	}

	err := watcher.RuntimeWatcher.ChangeOutRuntime(req.SchemeName, req.ActionName, req.OldOut, req.NewOut)
	if err != nil {
		logger.Errorf("can not change out %s -> %s for action '%s' from scheme '%s': %s",
			req.OldOut, req.NewOut, req.ActionName, req.SchemeName, err)
		return httplib.NewInternalErrorResponse(httplib.NewErrorBody(InternalError, err.Error())), nil
	}

	logger.Infof("changes out %s -> %s for action '%s' from scheme '%s'",
		req.OldOut, req.NewOut, req.ActionName, req.SchemeName)
	return httplib.NewOKResponse(nil, httplib.ContentTypeRaw), nil
}
