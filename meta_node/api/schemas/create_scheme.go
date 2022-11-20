package schemas

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"

	"github.com/GDVFox/gostreaming/meta_node/api/common"
	"github.com/GDVFox/gostreaming/meta_node/external"
	"github.com/GDVFox/gostreaming/meta_node/parser"
	"github.com/GDVFox/gostreaming/meta_node/planner"
	"github.com/GDVFox/gostreaming/meta_node/recognizer"
	"github.com/GDVFox/gostreaming/util/httplib"
	"github.com/GDVFox/gostreaming/util/storage"
)

// CreateScheme создает описание схемы.
func CreateScheme(r *http.Request) (*httplib.Response, error) {
	scheme := &planner.Scheme{}
	if err := json.NewDecoder(r.Body).Decode(&scheme); err != nil {
		return httplib.NewBadRequestResponse(httplib.NewErrorBody(common.BadUnmarshalRequestErrorCode, err.Error())), nil
	}

	plan, err := buildPlan(scheme)
	if err != nil {
		return httplib.NewBadRequestResponse(httplib.NewErrorBody(common.BadSchemeErrorCode, err.Error())), nil
	}

	if err := external.ETCD.RegisterPlan(r.Context(), plan); err != nil {
		if errors.Cause(err) == storage.ErrAlreadyExists {
			return httplib.NewConflictResponse(httplib.NewErrorBody(common.NameAlreadyExistsErrorCode, err.Error())), nil
		}
		return httplib.NewInternalErrorResponse(httplib.NewErrorBody(common.ETCDErrorCode, err.Error())), nil
	}

	return httplib.NewOKResponse(nil, httplib.ContentTypeRaw), nil
}

func buildPlan(scheme *planner.Scheme) (*planner.Plan, error) {
	analyzer := parser.NewSyntaxAnalyzer(recognizer.NewLexicalRecognizer(scheme.Dataflow))
	root, err := analyzer.Parse()
	if err != nil {
		return nil, err
	}

	pln, err := planner.NewPlanner(root, scheme)
	if err != nil {
		return nil, err
	}

	return pln.Plan()
}
