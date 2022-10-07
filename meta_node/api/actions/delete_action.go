package actions

import (
	"net/http"

	"github.com/GDVFox/gostreaming/meta_node/api/common"
	"github.com/GDVFox/gostreaming/meta_node/external"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// DeleteAction удаляем действие, если оно существует.
func DeleteAction(r *http.Request) (*common.Response, error) {
	vars := mux.Vars(r)
	actionName := vars["action_name"]
	if actionName == "" {
		return common.NewBadRequestResponse(common.NewErrorBody(common.BadNameErrorCode, "action_name must be not empty")), nil
	}

	if err := external.ETCD.DeleteAction(r.Context(), actionName); err != nil {
		if errors.Cause(err) == external.ErrNotFound {
			return common.NewNotFoundResponse(common.NewErrorBody(common.NameNotFoundErrorCode, err.Error())), nil
		}
		return common.NewInternalErrorResponse(common.NewErrorBody(common.ETCDErrorCode, err.Error())), nil
	}

	return common.NewOKResponse(nil, false), nil
}
