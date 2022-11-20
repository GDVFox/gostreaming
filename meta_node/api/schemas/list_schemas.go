package schemas

import (
	"encoding/json"
	"net/http"

	"github.com/GDVFox/gostreaming/meta_node/api/common"
	"github.com/GDVFox/gostreaming/meta_node/external"
	"github.com/GDVFox/gostreaming/meta_node/watcher"
	"github.com/GDVFox/gostreaming/util/httplib"
)

// SchemeDescription описание конкретной схемы.
type SchemeDescription struct {
	Name   string `json:"name"`
	Status int    `json:"status"`
}

// SchemeList список всех хранящихся схем с их статусами.
type SchemeList struct {
	Schemas []*SchemeDescription `json:"schemas"`
}

// ListSchemas получает список названий схем.
func ListSchemas(r *http.Request) (*httplib.Response, error) {
	schemeNames, err := external.ETCD.LoadPlanNames(r.Context())
	if err != nil {
		return httplib.NewInternalErrorResponse(httplib.NewErrorBody(common.ETCDErrorCode, err.Error())), nil
	}

	workingSchemas := watcher.Watcher.WorkingPlans()
	schemasList := &SchemeList{Schemas: make([]*SchemeDescription, 0, len(schemeNames))}
	for _, name := range schemeNames {
		description := &SchemeDescription{
			Name:   name,
			Status: 0,
		}
		if _, ok := workingSchemas[name]; ok {
			description.Status = 1
		}
		schemasList.Schemas = append(schemasList.Schemas, description)
	}

	schemasData, err := json.Marshal(schemasList)
	if err != nil {
		return httplib.NewInternalErrorResponse(httplib.NewErrorBody(common.BadSchemeErrorCode, err.Error())), nil
	}

	return httplib.NewOKResponse(schemasData, httplib.ContentTypeJSON), nil
}
