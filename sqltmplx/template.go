package sqltmplx

import (
	"github.com/DaiYuANg/archgo/sqltmplx/dialect"
	"github.com/DaiYuANg/archgo/sqltmplx/parse"
	"github.com/DaiYuANg/archgo/sqltmplx/render"
	"github.com/DaiYuANg/archgo/sqltmplx/scan"
)

type Template struct {
	nodes     []parse.Node
	dialect   dialect.Dialect
	validator interface{ Validate(string) error }
}

func compileTemplate(tpl string, d dialect.Dialect, cfg config) (*Template, error) {
	tokens, err := scan.Scan(tpl)
	if err != nil {
		return nil, err
	}
	nodes, err := parse.Build(tokens)
	if err != nil {
		return nil, err
	}
	return &Template{nodes: nodes, dialect: d, validator: cfg.validator}, nil
}

func (t *Template) Render(params any) (BoundSQL, error) {
	bound, err := render.Render(t.nodes, params, t.dialect)
	if err != nil {
		return BoundSQL{}, err
	}
	if t.validator != nil {
		if err := t.validator.Validate(bound.Query); err != nil {
			return BoundSQL{}, err
		}
	}
	return BoundSQL{Query: bound.Query, Args: bound.Args}, nil
}
