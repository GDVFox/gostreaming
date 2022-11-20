package api

import (
	"encoding/json"
	"net/http"

	"github.com/GDVFox/gostreaming/machine_node/watcher"
	"github.com/GDVFox/gostreaming/util/httplib"
)

// Ping возвращает информацию о состоянии запущенных действий.
func Ping(r *http.Request) (*httplib.Response, error) {
	telemetry := watcher.RuntimeWatcher.GetRuntimesTelemetry()
	schemeData, err := json.Marshal(telemetry)
	if err != nil {
		return httplib.NewInternalErrorResponse(httplib.NewErrorBody(BadTelemetry, err.Error())), nil
	}
	return httplib.NewOKResponse(schemeData, httplib.ContentTypeJSON), nil
}
