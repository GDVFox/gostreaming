package recognizer

import (
	"github.com/pkg/errors"
)

// Возможные ошибки лексического анализа
var (
	ErrUnknownDomain = errors.New("unknown domain")
)

// LexicalRecognizer конечный автомат, который выделяет токены в программе
type LexicalRecognizer struct {
	state int
	table JumpTable
	final map[int]Domain

	text  string
	index int
	line  int
	pos   int
}

// NewLexicalRecognizer создает LexicalRecognizer
func NewLexicalRecognizer(prog string) *LexicalRecognizer {
	return newLexicalRecognizer(prog, NewJumpTable(jumpTable, factorization), finalStates)
}

func newLexicalRecognizer(prog string, table JumpTable, final map[int]Domain) *LexicalRecognizer {
	return &LexicalRecognizer{
		state: 0,
		table: table,
		text:  prog,
		final: final,
		index: 0,
		line:  1,
		pos:   1,
	}
}

func (l *LexicalRecognizer) constructToken(tag Domain, endLine, endPos, endIndex int) (Token, error) {
	var token Token

	abstractToken := newToken(tag,
		l.line, l.pos, l.index,
		endLine, endPos, endIndex)

	switch tag {
	case WhitespaceDomain:
		token = &WhitespaceToken{
			token: abstractToken,
		}
	case IdentifierDomain:
		token = &IdentifierToken{
			token: abstractToken,
			ident: l.text[l.index:endIndex],
		}
	case LBracketDomain, RBracketDomain:
		token = &BracketToken{
			token: abstractToken,
		}
	case AlternativeDomain, SequentialDomain, ParallelDomain:
		token = &OperatorToken{
			token: abstractToken,
		}
	default:
		return nil, ErrUnknownDomain
	}

	return token, nil
}

func (l *LexicalRecognizer) resetState(newPos, newLine, newIndex int) {
	if newIndex == l.index {
		newIndex++
	}
	if newPos == l.pos {
		newPos++
	}

	l.index = newIndex
	l.pos = newPos
	l.line = newLine
	l.state = 0
}

// NextToken возвращает следующий токен или синтаксическую ошибку
func (l *LexicalRecognizer) NextToken() (Token, error) {
	if l.index == len(l.text) {
		return &EndToken{
			token: newToken(EndDomain, l.line, l.pos, l.index, l.line, l.pos, l.index),
		}, nil
	}

	endIndex := l.index
	endLine := l.line
	endPos := l.pos
	for endIndex != len(l.text) {
		nextState := l.table.Next(l.state, l.text[endIndex])
		if nextState == -1 {
			break
		}

		l.state = nextState
		if l.text[endIndex] == '\n' {
			endLine++
			endPos = 0
		}
		endIndex++
		endPos++
	}

	// defer нужен чтобы сохранять координаты начала при конструировании токенов и ошибок
	// и при этом состояние в любом случае сбросилось для разбора следующей лексемы.
	defer l.resetState(endPos, endLine, endIndex)

	domainTag, ok := l.final[l.state]
	if !ok || domainTag == UnknownDomain {
		return nil, errors.Wrapf(ErrUnknownDomain, "(%d %d)", l.line, l.pos)
	}

	token, err := l.constructToken(domainTag, endLine, endPos, endIndex)
	if err != nil {
		return nil, errors.Wrapf(err, "(%d %d)", l.line, l.pos)
	}

	return token, err
}
