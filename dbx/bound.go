package dbx

type BoundQuery struct {
	Name string
	SQL  string
	Args []any
}
