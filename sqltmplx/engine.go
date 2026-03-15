package sqltmplx

import "github.com/DaiYuANg/archgo/sqltmplx/dialect"

type Engine struct {
	dialect dialect.Dialect
	cfg     config
}

func New(d dialect.Dialect, opts ...Option) *Engine {
	cfg := config{}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &Engine{dialect: d, cfg: cfg}
}

func (e *Engine) Compile(tpl string) (*Template, error) {
	return compileTemplate(tpl, e.dialect, e.cfg)
}

func (e *Engine) Render(tpl string, params any) (BoundSQL, error) {
	t, err := e.Compile(tpl)
	if err != nil {
		return BoundSQL{}, err
	}
	return t.Render(params)
}
