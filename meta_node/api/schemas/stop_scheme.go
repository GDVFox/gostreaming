package schemas

import (
	"fmt"
	"net/http"

	"github.com/GDVFox/gostreaming/meta_node/api/common"
	"github.com/GDVFox/gostreaming/meta_node/watcher"
	"github.com/GDVFox/gostreaming/util/httplib"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// StopScheme останавливает схему.
func StopScheme(r *http.Request) (*httplib.Response, error) {
	vars := mux.Vars(r)
	schemeName := vars["scheme_name"]
	if schemeName == "" {
		return httplib.NewBadRequestResponse(httplib.NewErrorBody(common.BadNameErrorCode, "scheme_name must be not empty")), nil
	}

	if err := watcher.Watcher.StopPlan(schemeName); err != nil {
		if errors.Cause(err) == watcher.ErrNoAction || errors.Cause(err) == watcher.ErrNoHost {
			return httplib.NewBadRequestResponse(httplib.NewErrorBody(common.BadNameErrorCode, err.Error())), nil
		} else if errors.Cause(err) == watcher.ErrUnknownPlan {
			return httplib.NewNotFoundResponse(httplib.NewErrorBody(common.BadSchemeErrorCode, err.Error())), nil
		}
		return httplib.NewInternalErrorResponse(httplib.NewErrorBody(common.MachineErrorCode,
			fmt.Sprintf("unknown error: %s", err.Error()))), nil
	}

	return httplib.NewOKResponse(nil, false), nil
}
