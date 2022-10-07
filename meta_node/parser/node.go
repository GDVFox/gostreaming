package parser

import "github.com/GDVFox/gostreaming/meta_node/recognizer"

// Node интерфейс для представления вершины дерева разбора.
type Node interface {
	Coords() recognizer.Fragment
}

type abstractNode struct {
	coords recognizer.Fragment
}

func (n *abstractNode) Coords() recognizer.Fragment {
	return n.coords
}

// Operation алиас для определения типа операции
type Operation int

// Доступные виды операций
const (
	AlternativeOperation Operation = iota
	SequentialOperation
	ParallelOperation
)

func (o Operation) String() string {
	switch o {
	case AlternativeOperation:
		return "ALT"
	case SequentialOperation:
		return "SEQ"
	case ParallelOperation:
		return "PAR"
	default:
		return "UNK"
	}
}

// OperaionDomains соответствие между доменами распознавателя и операторами
var operaionDomains = map[recognizer.Domain]Operation{
	recognizer.AlternativeDomain: AlternativeOperation,
	recognizer.SequentialDomain:  SequentialOperation,
	recognizer.ParallelDomain:    ParallelOperation,
}

// OperaionNode вершина дерева разбора, представляющая собой операцию.
type OperaionNode struct {
	abstractNode
	Op   Operation
	Next []Node
}

// ActionNode вершина дерева разбора, представляющая собой действие.
type ActionNode struct {
	abstractNode
	Name string
}
