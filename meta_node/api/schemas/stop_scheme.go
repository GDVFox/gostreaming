package schemas

import (
	"fmt"
	"net/http"

	"github.com/GDVFox/gostreaming/meta_node/api/common"
	"github.com/GDVFox/gostreaming/meta_node/external"
	"github.com/GDVFox/gostreaming/util/httplib"
	"github.com/GDVFox/gostreaming/util/storage"
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

	plan, err := external.ETCD.LoadPlan(r.Context(), schemeName)
	if err != nil {
		if errors.Cause(err) == storage.ErrNotFound {
			return httplib.NewNotFoundResponse(httplib.NewErrorBody(common.NameNotFoundErrorCode, err.Error())), nil
		}
		return httplib.NewInternalErrorResponse(httplib.NewErrorBody(common.ETCDErrorCode, err.Error())), nil
	}

	// останавливаем в обратном порядке.
	for i := len(plan.Nodes) - 1; i >= 0; i-- {
		node := plan.Nodes[i]

		if err := external.Machines.SendStopAction(r.Context(), plan.Scheme.Name, node); err != nil {
			if errors.Cause(err) == external.ErrNoAction {
				return httplib.NewBadRequestResponse(httplib.NewErrorBody(common.BadNameErrorCode,
					fmt.Sprintf("scheme contains unknown action: %s", node.Action))), nil
			} else if errors.Cause(err) == external.ErrNoHost {
				return httplib.NewBadRequestResponse(httplib.NewErrorBody(common.BadNameErrorCode,
					fmt.Sprintf("scheme contains unknown host: %s", node.Host))), nil
			}

			return httplib.NewInternalErrorResponse(httplib.NewErrorBody(common.MachineErrorCode,
				fmt.Sprintf("unknown error: %s", err.Error()))), nil
		}
	}

	return httplib.NewOKResponse(nil, false), nil
}
