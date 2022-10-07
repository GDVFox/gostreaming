package actions

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"

	"github.com/GDVFox/gostreaming/meta_node/api/common"
	"github.com/GDVFox/gostreaming/meta_node/external"
)

// GetAction получает описание бинарный файл из хранилища.
func GetAction(r *http.Request) (*common.Response, error) {
	vars := mux.Vars(r)
	actionName := vars["action_name"]
	if actionName == "" {
		return common.NewBadRequestResponse(common.NewErrorBody(common.BadNameErrorCode, "action_name must be not empty")), nil
	}

	action, err := external.ETCD.LoadAction(r.Context(), actionName)
	if err != nil {
		if errors.Cause(err) == external.ErrNotFound {
			return common.NewNotFoundResponse(common.NewErrorBody(common.NameNotFoundErrorCode, err.Error())), nil
		}
		return common.NewInternalErrorResponse(common.NewErrorBody(common.ETCDErrorCode, err.Error())), nil
	}

	return common.NewOKResponse(action, false), nil
}
