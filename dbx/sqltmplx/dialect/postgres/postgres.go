package postgres

import "strconv"

type Dialect struct{}

func (Dialect) BindVar(n int) string { return "$" + strconv.Itoa(n) }
func (Dialect) Name() string         { return "postgres" }
