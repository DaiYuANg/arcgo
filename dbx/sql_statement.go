package dbx

type SQLStatementSource interface {
	StatementName() string
	Bind(params any) (BoundQuery, error)
}

type SQLStatement struct {
	name   string
	binder func(params any) (BoundQuery, error)
}

func NewSQLStatement(name string, binder func(params any) (BoundQuery, error)) *SQLStatement {
	return &SQLStatement{name: name, binder: binder}
}

func (s *SQLStatement) StatementName() string {
	if s == nil {
		return ""
	}
	return s.name
}

func (s *SQLStatement) Bind(params any) (BoundQuery, error) {
	if s == nil || s.binder == nil {
		return BoundQuery{}, ErrNilStatement
	}

	bound, err := s.binder(params)
	if err != nil {
		return BoundQuery{}, err
	}
	if bound.Name == "" {
		bound.Name = s.name
	}
	if bound.Args.Len() > 0 {
		bound.Args = bound.Args.Clone()
	}
	return bound, nil
}
