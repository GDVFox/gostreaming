package schemas

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"

	"github.com/GDVFox/gostreaming/meta_node/api/common"
	"github.com/GDVFox/gostreaming/meta_node/external"
)

// GetScheme получает описание схемы.
func GetScheme(r *http.Request) (*common.Response, error) {
	vars := mux.Vars(r)
	schemeName := vars["scheme_name"]
	if schemeName == "" {
		return common.NewBadRequestResponse(common.NewErrorBody(common.BadNameErrorCode, "scheme_name must be not empty")), nil
	}

	plan, err := external.ETCD.LoadPlan(r.Context(), schemeName)
	if err != nil {
		if errors.Cause(err) == external.ErrNotFound {
			return common.NewNotFoundResponse(common.NewErrorBody(common.NameNotFoundErrorCode, err.Error())), nil
		}
		return common.NewInternalErrorResponse(common.NewErrorBody(common.ETCDErrorCode, err.Error())), nil
	}

	schemeData, err := json.Marshal(plan.Scheme)
	if err != nil {
		return common.NewInternalErrorResponse(common.NewErrorBody(common.BadSchemeErrorCode, err.Error())), nil
	}

	return common.NewOKResponse(schemeData, true), nil
}
