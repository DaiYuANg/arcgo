package sqltmplx

import (
	"github.com/DaiYuANg/arcgo/dbx/dialect"
	"github.com/samber/lo"
)

type Engine struct {
	dialect dialect.Contract
	cfg     config
}

func New(d dialect.Contract, opts ...Option) *Engine {
	cfg := config{}
	lo.ForEach(opts, func(opt Option, _ int) {
		if opt != nil {
			opt(&cfg)
		}
	})
	return &Engine{dialect: d, cfg: cfg}
}

func (e *Engine) Compile(tpl string) (*Template, error) {
	return e.CompileNamed("", tpl)
}

func (e *Engine) CompileNamed(name string, tpl string) (*Template, error) {
	return compileTemplate(name, tpl, e.dialect, e.cfg)
}

func (e *Engine) Render(tpl string, params any) (BoundSQL, error) {
	t, err := e.Compile(tpl)
	if err != nil {
		return BoundSQL{}, err
	}
	return t.Render(params)
}
