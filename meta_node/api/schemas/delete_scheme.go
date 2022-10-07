package schemas

import (
	"net/http"

	"github.com/GDVFox/gostreaming/meta_node/api/common"
	"github.com/GDVFox/gostreaming/meta_node/external"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// DeleteScheme удаляем описание схемы, если оно существует.
func DeleteScheme(r *http.Request) (*common.Response, error) {
	vars := mux.Vars(r)
	schemeName := vars["scheme_name"]
	if schemeName == "" {
		return common.NewBadRequestResponse(common.NewErrorBody(common.BadNameErrorCode, "scheme_name must be not empty")), nil
	}

	if err := external.ETCD.DeletePlan(r.Context(), schemeName); err != nil {
		if errors.Cause(err) == external.ErrNotFound {
			return common.NewNotFoundResponse(common.NewErrorBody(common.NameNotFoundErrorCode, err.Error())), nil
		}
		return common.NewInternalErrorResponse(common.NewErrorBody(common.ETCDErrorCode, err.Error())), nil
	}

	return common.NewOKResponse(nil, false), nil
}
