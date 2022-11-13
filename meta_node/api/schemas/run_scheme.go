package schemas

import (
	"fmt"
	"net/http"

	"github.com/GDVFox/gostreaming/meta_node/api/common"
	"github.com/GDVFox/gostreaming/meta_node/external"
	"github.com/GDVFox/gostreaming/meta_node/watcher"
	"github.com/GDVFox/gostreaming/util/httplib"
	"github.com/GDVFox/gostreaming/util/storage"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// RunScheme запускает схему с указанной конфигурацией.
func RunScheme(r *http.Request) (*httplib.Response, error) {
	vars := mux.Vars(r)
	schemeName := vars["scheme_name"]
	if schemeName == "" {
		return httplib.NewBadRequestResponse(httplib.NewErrorBody(common.BadNameErrorCode, "scheme_name must be not empty")), nil
	}

	plan, err := external.ETCD.LoadPlan(r.Context(), schemeName)
	if err != nil {
		if errors.Cause(err) == storage.ErrNotFound {
			return httplib.NewNotFoundResponse(httplib.NewErrorBody(common.NameNotFoundErrorCode, err.Error())), nil
		}
		return httplib.NewInternalErrorResponse(httplib.NewErrorBody(common.ETCDErrorCode, err.Error())), nil
	}

	if err := watcher.Watcher.RunPlan(plan); err != nil {
		if errors.Cause(err) == watcher.ErrNoAction || errors.Cause(err) == watcher.ErrNoHost {
			return httplib.NewBadRequestResponse(httplib.NewErrorBody(common.BadNameErrorCode, err.Error())), nil
		}
		return httplib.NewInternalErrorResponse(httplib.NewErrorBody(common.MachineErrorCode,
			fmt.Sprintf("unknown error: %s", err.Error()))), nil
	}

	return httplib.NewOKResponse(nil, false), nil
}
