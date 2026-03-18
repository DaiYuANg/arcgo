package validate

type Noop struct{}

func (Noop) Validate(string) error { return nil }

func (Noop) Analyze(sql string) (*Analysis, error) {
	return &Analysis{
		Dialect:       "",
		StatementType: detectStatementType(sql),
		NormalizedSQL: sql,
		AST:           nil,
	}, nil
}
