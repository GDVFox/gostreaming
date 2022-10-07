package schemas

import (
	"encoding/json"
	"net/http"

	"github.com/GDVFox/gostreaming/meta_node/api/common"
	"github.com/GDVFox/gostreaming/meta_node/external"
	"github.com/GDVFox/gostreaming/meta_node/parser"
	"github.com/GDVFox/gostreaming/meta_node/planner"
	"github.com/GDVFox/gostreaming/meta_node/recognizer"
	"github.com/pkg/errors"
)

// CreateScheme создает описание схемы.
func CreateScheme(r *http.Request) (*common.Response, error) {
	scheme := &planner.Scheme{}
	if err := json.NewDecoder(r.Body).Decode(&scheme); err != nil {
		return common.NewBadRequestResponse(common.NewErrorBody(common.BadUnmarshalRequestErrorCode, err.Error())), nil
	}

	plan, err := buildPlan(scheme)
	if err != nil {
		return common.NewBadRequestResponse(common.NewErrorBody(common.BadSchemeErrorCode, err.Error())), nil
	}

	if err := external.ETCD.RegisterPlan(r.Context(), plan); err != nil {
		if errors.Cause(err) == external.ErrAlreadyExists {
			return common.NewConflictResponse(common.NewErrorBody(common.NameAlreadyExistsErrorCode, err.Error())), nil
		}
		return common.NewInternalErrorResponse(common.NewErrorBody(common.ETCDErrorCode, err.Error())), nil
	}

	return common.NewOKResponse(nil, false), nil
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
