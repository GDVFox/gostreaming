package actions

import (
	"encoding/json"
	"net/http"

	"github.com/GDVFox/gostreaming/meta_node/api/common"
	"github.com/GDVFox/gostreaming/meta_node/external"
	"github.com/GDVFox/gostreaming/util/httplib"
)

// ActionList список имен действий.
type ActionList struct {
	Actions []string `json:"actions"`
}

// ListActions получает список названий действий.
func ListActions(r *http.Request) (*httplib.Response, error) {
	actionsNames, err := external.ETCD.LoadActionNames(r.Context())
	if err != nil {
		return httplib.NewInternalErrorResponse(httplib.NewErrorBody(common.ETCDErrorCode, err.Error())), nil
	}

	list := &ActionList{Actions: actionsNames}
	actionsData, err := json.Marshal(list)
	if err != nil {
		return httplib.NewInternalErrorResponse(httplib.NewErrorBody(common.BadActionErrorCode, err.Error())), nil
	}

	return httplib.NewOKResponse(actionsData, httplib.ContentTypeJSON), nil
}
