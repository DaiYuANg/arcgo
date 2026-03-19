package dialect

type Contract interface {
	Name() string
	BindVar(n int) string
}

type Dialect interface {
	Contract
	QuoteIdent(ident string) string
	RenderLimitOffset(limit, offset *int) (string, error)
}
