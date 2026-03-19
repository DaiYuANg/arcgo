package sqltmplx

import (
	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/DaiYuANg/arcgo/dbx/dialect"
	"github.com/DaiYuANg/arcgo/dbx/sqltmplx/parse"
	"github.com/DaiYuANg/arcgo/dbx/sqltmplx/render"
	"github.com/DaiYuANg/arcgo/dbx/sqltmplx/scan"
	"github.com/DaiYuANg/arcgo/dbx/sqltmplx/validate"
)

type Template struct {
	name      string
	nodes     []parse.Node
	dialect   dialect.Contract
	validator validate.Validator
}

func compileTemplate(name string, tpl string, d dialect.Contract, cfg config) (*Template, error) {
	tokens, err := scan.Scan(tpl)
	if err != nil {
		return nil, err
	}
	nodes, err := parse.Build(tokens)
	if err != nil {
		return nil, err
	}
	return &Template{name: name, nodes: nodes, dialect: d, validator: cfg.validator}, nil
}

func (t *Template) StatementName() string {
	if t == nil {
		return ""
	}
	return t.name
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

func (t *Template) Bind(params any) (dbx.BoundQuery, error) {
	if t == nil {
		return dbx.BoundQuery{}, dbx.ErrNilStatement
	}

	bound, err := t.Render(params)
	if err != nil {
		return dbx.BoundQuery{}, err
	}
	return dbx.BoundQuery{
		Name: t.name,
		SQL:  bound.Query,
		Args: append([]any(nil), bound.Args...),
	}, nil
}
