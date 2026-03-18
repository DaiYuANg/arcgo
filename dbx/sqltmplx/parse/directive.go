package parse

type Directive struct {
	If    *IfDirective
	Where *WhereDirective
	Set   *SetDirective
	End   *EndDirective
}

type IfDirective struct {
	Keyword string
	Expr    string
}

type WhereDirective struct {
	Keyword string
}

type SetDirective struct {
	Keyword string
}

type EndDirective struct {
	Keyword string
}
