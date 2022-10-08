package schemas

import (
	"encoding/json"
	"net/http"

	"github.com/GDVFox/gostreaming/meta_node/api/common"
	"github.com/GDVFox/gostreaming/meta_node/external"
	"github.com/GDVFox/gostreaming/util/httplib"
)

// SchemeList список имен схем.
type SchemeList struct {
	Schemas []string `json:"schemas"`
}

// ListSchemas получает список названий схем.
func ListSchemas(r *http.Request) (*httplib.Response, error) {
	schemeNames, err := external.ETCD.LoadPlanNames(r.Context())
	if err != nil {
		return httplib.NewInternalErrorResponse(httplib.NewErrorBody(common.ETCDErrorCode, err.Error())), nil
	}

	list := &SchemeList{Schemas: schemeNames}
	schemasData, err := json.Marshal(list)
	if err != nil {
		return httplib.NewInternalErrorResponse(httplib.NewErrorBody(common.BadSchemeErrorCode, err.Error())), nil
	}

	return httplib.NewOKResponse(schemasData, true), nil
}
