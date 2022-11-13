package planner

import (
	"strconv"

	"github.com/pkg/errors"

	"github.com/GDVFox/gostreaming/meta_node/parser"
)

// Возможные ошибки
var (
	ErrNodeAlreadyExists = errors.New("node with same name already exists")
	ErrUnknownNodeName   = errors.New("unknown node name")
	ErrUnknownOperation  = errors.New("unknown operation")
	ErrUnknownNode       = errors.New("unknown node")
	ErrFoundCycle        = errors.New("found cycle")
	ErrAlreadyUsed       = errors.New("node already used in dataflow")
)

// Plan содержит информацию, необходимую для запуска обработки потока на серверах.
type Plan struct {
	Name   string      `json:"name"`
	Scheme *Scheme     `json:"scheme"`
	Nodes  []*NodePlan `json:"nodes"`
}

// NodePlan описание узла, предназначенного для запуска на сервере.
type NodePlan struct {
	Name      string             `json:"name"`
	Action    string             `json:"action"`
	Host      string             `json:"host"`
	Port      int                `json:"port"`
	In        []string           `json:"in"`
	Out       []string           `json:"out"`
	Args      []string           `json:"args"`
	Env       map[string]string  `json:"env"`
	Addresses []*AddrDescription `json:"addresses"`
}

// node вершина в дереве связей узлов.
// Содежит список адресов, от которых данный узел принимает данные,
// и список адресов, на которые отправляет данные.
type node struct {
	In  []string
	Out []string
}

// Planner преобразует дерево разбора программы потока данных
// в план выполнения действий на серверах.
type Planner struct {
	root parser.Node

	scheme *Scheme
	nodes  map[string]*NodeDescription
	used   map[string]struct{}

	nodeConnections map[string]*node
}

// NewPlanner создает новый объект Planner
func NewPlanner(r parser.Node, scheme *Scheme) (*Planner, error) {
	if err := scheme.Check(); err != nil {
		return nil, errors.Wrap(err, "scheme validation failed")
	}

	nodes := make(map[string]*NodeDescription, len(scheme.Nodes))
	nodeConnections := make(map[string]*node, len(scheme.Nodes))
	for _, n := range scheme.Nodes {
		if _, ok := nodes[n.Name]; ok {
			return nil, errors.Wrapf(ErrNodeAlreadyExists, "%s", n.Name)
		}
		nodes[n.Name] = n
		nodeConnections[n.Name] = &node{
			In:  make([]string, 0),
			Out: make([]string, 0),
		}
	}

	return &Planner{
		root:            r,
		scheme:          scheme,
		nodes:           nodes,
		used:            make(map[string]struct{}),
		nodeConnections: nodeConnections,
	}, nil
}

// Plan строит план выполнения пайплайна.
func (s *Planner) Plan() (*Plan, error) {
	root, err := s.scheduleNode(s.root)
	if err != nil {
		return nil, err
	}
	orderedNodePlans := make([]*NodePlan, 0)
	colors := make(map[string]byte, 0)

	stack := make([]string, 0)
	stack = append(stack, root.In...)
	for len(stack) > 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if colors[node] == 'g' {
			colors[node] = 'b'
			nodeDescr := s.nodes[node]
			in := make([]string, len(s.nodeConnections[node].In))
			for i, n := range s.nodeConnections[node].In {
				in[i] = s.scheme.Name + "_" + s.nodes[n].Name
			}
			out := make([]string, len(s.nodeConnections[node].Out))
			// По умолчанию используется первый адрес.
			for i, n := range s.nodeConnections[node].Out {
				out[i] = s.nodes[n].Addresses[0].Host + ":" + strconv.Itoa(s.nodes[n].Addresses[0].Port)
			}
			orderedNodePlans = append(orderedNodePlans, &NodePlan{
				Name:      nodeDescr.Name,
				Action:    nodeDescr.Action,
				Host:      nodeDescr.Addresses[0].Host,
				Port:      nodeDescr.Addresses[0].Port,
				In:        in,
				Out:       out,
				Args:      nodeDescr.Args,
				Env:       nodeDescr.Env,
				Addresses: nodeDescr.Addresses,
			})
			continue
		}

		colors[node] = 'g'
		stack = append(stack, node)
		for _, n := range s.nodeConnections[node].Out {
			if colors[n] == 'b' {
				continue
			}
			if colors[n] == 'g' {
				// такое поведение невозможно, но оставляем ошибку, чтобы поймать баг в алгоритме.
				return nil, errors.Wrapf(ErrFoundCycle, "%s -> %s", node, n)
			}
			stack = append(stack, n)
		}
	}

	return &Plan{
		Name:   s.scheme.Name,
		Scheme: s.scheme,
		Nodes:  orderedNodePlans,
	}, nil
}

func (s *Planner) scheduleNode(r parser.Node) (*node, error) {
	switch v := r.(type) {
	case *parser.ActionNode:
		return s.scheduleActionNode(v)
	case *parser.OperaionNode:
		return s.scheduleOperationNode(v)
	default:
		return nil, errors.Wrapf(ErrUnknownNode, "(%d, %d)", r.Coords().Starting.Line, r.Coords().Starting.Pos)
	}
}

func (s *Planner) scheduleActionNode(r *parser.ActionNode) (*node, error) {
	if _, ok := s.nodes[r.Name]; !ok {
		return nil, errors.Wrapf(ErrUnknownNodeName,
			"(%d, %d) %s", r.Coords().Starting.Line, r.Coords().Starting.Pos, r.Name)
	}
	if _, ok := s.used[r.Name]; ok {
		return nil, errors.Wrapf(ErrAlreadyUsed,
			"(%d, %d) %s", r.Coords().Starting.Line, r.Coords().Starting.Pos, r.Name)
	}

	s.used[r.Name] = struct{}{}
	return &node{
		In:  []string{r.Name},
		Out: []string{r.Name},
	}, nil
}

func (s *Planner) scheduleOperationNode(r *parser.OperaionNode) (*node, error) {
	switch r.Op {
	case parser.ParallelOperation:
		return s.scheduleParallelNode(r)
	case parser.AlternativeOperation:
		return s.scheduleAlternativeNode(r)
	case parser.SequentialOperation:
		return s.scheduleSequentialNode(r)
	default:
		return nil, errors.Wrapf(ErrUnknownOperation,
			"(%d %d) %s", r.Coords().Starting.Line, r.Coords().Starting.Pos, r.Op)
	}
}

func (s *Planner) scheduleParallelNode(r *parser.OperaionNode) (*node, error) {
	in := make([]string, 0)
	out := make([]string, 0)

	for _, node := range r.Next {
		sNode, err := s.scheduleNode(node)
		if err != nil {
			return nil, err
		}
		in = append(in, sNode.In...)
		out = append(out, sNode.Out...)
	}

	return &node{
		In:  in,
		Out: out,
	}, nil
}

func (s *Planner) scheduleAlternativeNode(r *parser.OperaionNode) (*node, error) {
	return s.scheduleParallelNode(r)
}

func (s *Planner) scheduleSequentialNode(r *parser.OperaionNode) (*node, error) {
	in := make([]string, 0)
	out := make([]string, 0)

	var sPrevNode *node
	for i, node := range r.Next {
		sNode, err := s.scheduleNode(node)
		if err != nil {
			return nil, err
		}

		if i == 0 {
			in = append(in, sNode.In...)
			sPrevNode = sNode
			continue
		}
		if i == len(r.Next)-1 {
			out = append(out, sNode.Out...)
		}

		for _, actionNode := range sPrevNode.Out {
			s.nodeConnections[actionNode].Out = append(s.nodeConnections[actionNode].Out, sNode.In...)
		}
		for _, actionNode := range sNode.In {
			s.nodeConnections[actionNode].In = append(s.nodeConnections[actionNode].In, sPrevNode.Out...)
		}

		sPrevNode = sNode
	}

	return &node{
		In:  in,
		Out: out,
	}, nil
}
