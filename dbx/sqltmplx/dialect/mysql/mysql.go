package mysql

type Dialect struct{}

func (Dialect) BindVar(_ int) string { return "?" }
func (Dialect) Name() string         { return "mysql" }
