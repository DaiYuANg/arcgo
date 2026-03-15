package render

import (
	"fmt"
	"strings"

	"github.com/DaiYuANg/archgo/sqltmplx/dialect"
	"github.com/DaiYuANg/archgo/sqltmplx/parse"
	"github.com/expr-lang/expr/vm"
)

func Render(nodes []parse.Node, params any, d dialect.Dialect) (Result, error) {
	st := newState(params, d)
	query, err := renderNodes(nodes, st)
	if err != nil {
		return Result{}, err
	}
	return Result{Query: compactWhitespace(query), Args: st.args}, nil
}

func renderNodes(nodes []parse.Node, st *state) (string, error) {
	var sb strings.Builder
	for _, node := range nodes {
		switch n := node.(type) {
		case parse.TextNode:
			text, err := bindText(n.Text, st)
			if err != nil {
				return "", err
			}
			sb.WriteString(text)
		case *parse.IfNode:
			ok, err := evalIf(n.Program, st.params)
			if err != nil {
				return "", err
			}
			if ok {
				text, err := renderNodes(n.Body, st)
				if err != nil {
					return "", err
				}
				sb.WriteString(text)
			}
		case *parse.WhereNode:
			text, err := renderNodes(n.Body, st)
			if err != nil {
				return "", err
			}
			cleaned := cleanupWhere(text)
			if cleaned != "" {
				sb.WriteByte(' ')
				sb.WriteString(cleaned)
				sb.WriteByte(' ')
			}
		case *parse.SetNode:
			text, err := renderNodes(n.Body, st)
			if err != nil {
				return "", err
			}
			cleaned := cleanupSet(text)
			if cleaned != "" {
				sb.WriteByte(' ')
				sb.WriteString(cleaned)
				sb.WriteByte(' ')
			}
		default:
			return "", fmt.Errorf("sqltmplx: unsupported node %T", node)
		}
	}
	return sb.String(), nil
}

func evalIf(program *vm.Program, params any) (bool, error) {
	out, err := exprRun(program, envMap(params))
	if err != nil {
		return false, err
	}
	b, ok := out.(bool)
	if !ok {
		return false, fmt.Errorf("sqltmplx: if expression must return bool")
	}
	return b, nil
}
