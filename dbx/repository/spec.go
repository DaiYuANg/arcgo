package repository

import "github.com/DaiYuANg/arcgo/dbx"

// Spec mutates a select query before repository execution.
type Spec interface {
	Apply(query *dbx.SelectQuery) *dbx.SelectQuery
}

// SpecFunc adapts a function into a Spec.
type SpecFunc func(query *dbx.SelectQuery) *dbx.SelectQuery

// Apply applies the wrapped query mutation.
func (f SpecFunc) Apply(query *dbx.SelectQuery) *dbx.SelectQuery { return f(query) }

// Where appends a predicate to the query.
func Where(predicate dbx.Predicate) Spec {
	return SpecFunc(func(query *dbx.SelectQuery) *dbx.SelectQuery { return query.Where(predicate) })
}

// OrderBy appends one or more order clauses to the query.
func OrderBy(orders ...dbx.Order) Spec {
	return SpecFunc(func(query *dbx.SelectQuery) *dbx.SelectQuery { return query.OrderBy(orders...) })
}

// Page applies a normalized page request to the query.
func Page(page, pageSize int) Spec {
	return dbx.NewPageRequest(page, pageSize)
}

// PageByRequest applies an existing page request to the query.
func PageByRequest(request PageRequest) Spec {
	return request.Normalize()
}

// Limit applies a row limit to the query.
func Limit(limit int) Spec {
	return SpecFunc(func(query *dbx.SelectQuery) *dbx.SelectQuery { return query.Limit(limit) })
}

// Offset applies a row offset to the query.
func Offset(offset int) Spec {
	return SpecFunc(func(query *dbx.SelectQuery) *dbx.SelectQuery { return query.Offset(offset) })
}
