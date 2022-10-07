package actions

import (
	"io/ioutil"
	"net/http"

	"github.com/GDVFox/gostreaming/meta_node/api/common"
	"github.com/GDVFox/gostreaming/meta_node/external"
	"github.com/pkg/errors"
)

const (
	maxFormSize = 256 * 1024 * 1024 // 256MB
)

// CreateScheme создает описание схемы.
func CreateScheme(r *http.Request) (*common.Response, error) {
	if err := r.ParseMultipartForm(maxFormSize); err != nil {
		return common.NewBadRequestResponse(common.NewErrorBody(common.BadActionErrorCode, err.Error())), nil
	}
	name := r.FormValue("name")
	if name == "" {
		return common.NewBadRequestResponse(common.NewErrorBody(common.BadNameErrorCode, "expected non empty name")), nil
	}
	actionFile, _, err := r.FormFile("action")
	if err != nil {
		return common.NewBadRequestResponse(common.NewErrorBody(common.BadActionErrorCode, err.Error())), nil
	}
	action, err := ioutil.ReadAll(actionFile)
	if err != nil {
		return common.NewBadRequestResponse(common.NewErrorBody(common.BadActionErrorCode, err.Error())), nil
	}

	if err := external.ETCD.RegisterAction(r.Context(), name, action); err != nil {
		if errors.Cause(err) == external.ErrAlreadyExists {
			return common.NewConflictResponse(common.NewErrorBody(common.NameAlreadyExistsErrorCode, err.Error())), nil
		}
		return common.NewInternalErrorResponse(common.NewErrorBody(common.ETCDErrorCode, err.Error())), nil
	}

	return common.NewOKResponse(nil, false), nil
}
