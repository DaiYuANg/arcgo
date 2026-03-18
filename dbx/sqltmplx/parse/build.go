package parse

import (
	"fmt"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dbx/sqltmplx/scan"
	"github.com/expr-lang/expr"
)

type frameKind int

const (
	frameRoot frameKind = iota
	frameIf
	frameWhere
	frameSet
)

type frame struct {
	kind frameKind
	out  *[]Node
}

func Build(tokens []scan.Token) ([]Node, error) {
	var nodes []Node
	stack := collectionx.NewListWithCapacity[frame](4, frame{kind: frameRoot, out: &nodes})

	appendNode := func(n Node) {
		current, _ := stack.Get(stack.Len() - 1)
		out := current.out
		*out = append(*out, n)
	}

	for _, tok := range tokens {
		switch tok.Kind {
		case scan.Text:
			appendNode(TextNode{Text: tok.Value})
		case scan.Directive:
			d, err := parseDirective(tok.Value)
			if err != nil {
				return nil, err
			}
			switch {
			case d.If != nil:
				program, err := expr.Compile(d.If.Expr)
				if err != nil {
					return nil, fmt.Errorf("sqltmplx: compile expr %q: %w", d.If.Expr, err)
				}
				node := &IfNode{RawExpr: d.If.Expr, Program: program}
				appendNode(node)
				stack.Add(frame{kind: frameIf, out: &node.Body})
			case d.Where != nil:
				node := &WhereNode{}
				appendNode(node)
				stack.Add(frame{kind: frameWhere, out: &node.Body})
			case d.Set != nil:
				node := &SetNode{}
				appendNode(node)
				stack.Add(frame{kind: frameSet, out: &node.Body})
			case d.End != nil:
				if stack.Len() == 1 {
					return nil, fmt.Errorf("sqltmplx: unexpected end")
				}
				_, _ = stack.RemoveAt(stack.Len() - 1)
			}
		}
	}

	if stack.Len() != 1 {
		return nil, fmt.Errorf("sqltmplx: unclosed block")
	}
	return nodes, nil
}
