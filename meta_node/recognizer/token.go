package recognizer

// Position задает координаты кодовой точки
type Position struct {
	Line  int
	Pos   int
	Index int
}

// Fragment задает координаты фрагмента исходного кода
type Fragment struct {
	Starting Position
	Ending   Position
}

// Domain представляет лексический домен
type Domain int

const (
	// WhitespaceDomain представляет лексический домен пробельных символов.
	WhitespaceDomain Domain = iota
	// IdentifierDomain представляет лексический домен идентификаторов.
	IdentifierDomain
	// LBracketDomain представляет лексический домен левой скобки.
	LBracketDomain
	// RBracketDomain представляет лексический домен правой скобки.
	RBracketDomain
	// AlternativeDomain представляет домен оператора альтернирования.
	AlternativeDomain
	// SequentialDomain представляет домен оператора последовательного выполнения.
	SequentialDomain
	// ParallelDomain представляет домен оператора параллельного выполнения.
	ParallelDomain
	// EndDomain представляет конец файла
	EndDomain
	// UnknownDomain представляет ошибочный домен
	UnknownDomain Domain = -1
)

func (d Domain) String() string {
	switch d {
	case WhitespaceDomain:
		return "SPC"
	case IdentifierDomain:
		return "IDN"
	case LBracketDomain:
		return "LBR"
	case RBracketDomain:
		return "RBR"
	case AlternativeDomain:
		return "ALT"
	case SequentialDomain:
		return "SEQ"
	case ParallelDomain:
		return "PAR"
	case EndDomain:
		return "END"
	default:
		return "UNK"
	}
}

// Token абстрактный токен, содержащий основные методы
type Token interface {
	Domain() Domain
	Coords() Fragment
	Attr() interface{}
}

type token struct {
	tag    Domain
	coords Fragment
}

func newToken(tag Domain, sLine, sPos, sIndex, eLine, ePos, eIndex int) token {
	return token{
		tag: tag,
		coords: Fragment{
			Starting: Position{
				Line:  sLine,
				Pos:   sPos,
				Index: sIndex,
			},
			Ending: Position{
				Line:  eLine,
				Pos:   ePos,
				Index: eIndex,
			},
		},
	}
}

func (t *token) Domain() Domain {
	return t.tag
}

func (t *token) Coords() Fragment {
	return t.coords
}

func (t *token) Attr() interface{} {
	return nil
}

// WhitespaceToken представляет пробельные символы
type WhitespaceToken struct {
	token
}

// IdentifierToken представляет идентификатор
type IdentifierToken struct {
	token
	ident string
}

// Attr возвращает код идентификатора представляемый целым числом
func (t *IdentifierToken) Attr() interface{} {
	return t.ident
}

// BracketToken представляет скобку
type BracketToken struct {
	token
}

// Attr возвращает код идентификатора представляемый целым числом
func (t *BracketToken) Attr() interface{} {
	return nil
}

// OperatorToken представляет оператор языка
type OperatorToken struct {
	token
}

// Attr возвращает символ оператора
func (t *OperatorToken) Attr() interface{} {
	return nil
}

// EndToken представляет конец программы
type EndToken struct {
	token
}
