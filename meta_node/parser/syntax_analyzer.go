package parser

import (
	"github.com/pkg/errors"

	"github.com/GDVFox/gostreaming/meta_node/recognizer"
)

// Возможные синтаксические ошибки
var (
	ErrUnexpectedToken = errors.New("unexpected token")
)

// SyntaxAnalyzer структура представляющая синтаксический анализатор
type SyntaxAnalyzer struct {
	tokens *recognizer.LexicalRecognizer
	sym    recognizer.Token
}

// NewSyntaxAnalyzer создает новый синтаксический анализатор, связанный с переданным лексическим распознавателем.
func NewSyntaxAnalyzer(r *recognizer.LexicalRecognizer) *SyntaxAnalyzer {
	return &SyntaxAnalyzer{
		tokens: r,
	}
}

// Parse выполняет построение дерева разбора.
func (a *SyntaxAnalyzer) Parse() (Node, error) {
	if err := a.nextToken(); err != nil {
		return nil, err
	}

	root, err := a.parseDataflow()
	if err != nil {
		return nil, err
	}

	if a.sym.Domain() != recognizer.EndDomain {
		return nil, buildTokenError(a.sym, ErrUnexpectedToken)
	}

	return root, nil
}

func (a *SyntaxAnalyzer) parseDataflow() (Node, error) {
	alternativeNode1, err := a.parseAlternative()
	if err != nil {
		return nil, err
	}

	var operationRoot *OperaionNode
	for a.sym.Domain() == recognizer.AlternativeDomain {
		if operationRoot == nil {
			operationRoot = &OperaionNode{
				abstractNode: abstractNode{coords: a.sym.Coords()},
				Op:           AlternativeOperation,
				Next:         make([]Node, 0, 2), // ожидаем как минимум 2.
			}
			operationRoot.Next = append(operationRoot.Next, alternativeNode1)
		}

		if err = a.nextToken(); err != nil {
			return nil, err
		}

		alternativeNodeN, err := a.parseAlternative()
		if err != nil {
			return nil, err
		}

		operationRoot.Next = append(operationRoot.Next, alternativeNodeN)
	}

	if operationRoot != nil {
		return operationRoot, nil
	}

	return alternativeNode1, nil
}

func (a *SyntaxAnalyzer) parseAlternative() (Node, error) {
	actionNode1, err := a.parseActionBlock()
	if err != nil {
		return nil, err
	}

	var operationRoot *OperaionNode
	currentLeft := actionNode1
	prevDomain := recognizer.UnknownDomain
	for a.sym.Domain() == recognizer.SequentialDomain ||
		a.sym.Domain() == recognizer.ParallelDomain {

		// В случае смены операции текущий корень становится самым левым операндом
		// для новой операции.
		if prevDomain == recognizer.UnknownDomain || a.sym.Domain() != prevDomain {
			newOperation := &OperaionNode{
				abstractNode: abstractNode{coords: a.sym.Coords()},
				Op:           operaionDomains[a.sym.Domain()],
				Next:         make([]Node, 0, 2), // ожидаем как минимум 2.
			}
			newOperation.Next = append(newOperation.Next, currentLeft)

			operationRoot = newOperation
			currentLeft = operationRoot

			prevDomain = a.sym.Domain()
		}

		if err = a.nextToken(); err != nil {
			return nil, err
		}

		actionNodeN, err := a.parseActionBlock()
		if err != nil {
			return nil, err
		}

		operationRoot.Next = append(operationRoot.Next, actionNodeN)
	}

	if operationRoot != nil {
		return operationRoot, nil
	}
	return actionNode1, nil
}

func (a *SyntaxAnalyzer) parseActionBlock() (Node, error) {
	var err error

	if a.sym.Domain() == recognizer.IdentifierDomain {
		newNode := &ActionNode{
			abstractNode: abstractNode{coords: a.sym.Coords()},
			Name:         a.sym.Attr().(string),
		}
		if err = a.nextToken(); err != nil {
			return nil, err
		}
		return newNode, nil
	}

	if a.sym.Domain() != recognizer.LBracketDomain {
		return nil, buildTokenError(a.sym, ErrUnexpectedToken)
	}
	if err = a.nextToken(); err != nil {
		return nil, err
	}

	nextNode, err := a.parseDataflow()
	if err != nil {
		return nil, err
	}

	if a.sym.Domain() != recognizer.RBracketDomain {
		return nil, buildTokenError(a.sym, ErrUnexpectedToken)
	}
	if err = a.nextToken(); err != nil {
		return nil, err
	}

	return nextNode, nil
}

func (a *SyntaxAnalyzer) nextToken() error {
	var err error
	a.sym, err = a.tokens.NextToken()
	if err != nil {
		return buildUnknownError(err)
	}

	for a.sym.Domain() == recognizer.WhitespaceDomain {
		a.sym, err = a.tokens.NextToken()
		if err != nil {
			return buildUnknownError(err)
		}
	}

	return nil
}

func buildTokenError(t recognizer.Token, err error) error {
	return errors.Wrapf(err, "syntax error (%d %d)", t.Coords().Starting.Line, t.Coords().Starting.Pos)
}

func buildUnknownError(err error) error {
	return errors.Wrap(err, "syntax error")
}
