package schemas

import (
	"encoding/json"
	"net/http"

	"github.com/GDVFox/gostreaming/meta_node/api/common"
	"github.com/GDVFox/gostreaming/meta_node/external"
)

// SchemeList список имен схем.
type SchemeList struct {
	Schemas []string `json:"schemas"`
}

// ListSchemas получает список названий схем.
func ListSchemas(r *http.Request) (*common.Response, error) {
	schemeNames, err := external.ETCD.LoadPlanNames(r.Context())
	if err != nil {
		return common.NewInternalErrorResponse(common.NewErrorBody(common.ETCDErrorCode, err.Error())), nil
	}

	list := &SchemeList{Schemas: schemeNames}
	schemasData, err := json.Marshal(list)
	if err != nil {
		return common.NewInternalErrorResponse(common.NewErrorBody(common.BadSchemeErrorCode, err.Error())), nil
	}

	return common.NewOKResponse(schemasData, true), nil
}
