package parse

import "github.com/expr-lang/expr/vm"

// Node represents a parsed template node.
type Node interface {
	node()
}

// TextNode stores literal SQL text.
type TextNode struct {
	Text string
}

func (TextNode) node() {}

// IfNode stores a conditional block.
type IfNode struct {
	RawExpr string
	Program *vm.Program
	Body    []Node
}

func (*IfNode) node() {}

// WhereNode stores a conditional WHERE block.
type WhereNode struct {
	Body []Node
}

func (*WhereNode) node() {}

// SetNode stores a conditional SET block.
type SetNode struct {
	Body []Node
}

func (*SetNode) node() {}
