package repository

import "github.com/DaiYuANg/arcgo/dbx"

type Spec interface {
	Apply(query *dbx.SelectQuery) *dbx.SelectQuery
}

type SpecFunc func(query *dbx.SelectQuery) *dbx.SelectQuery

func (f SpecFunc) Apply(query *dbx.SelectQuery) *dbx.SelectQuery { return f(query) }

func Where(predicate dbx.Predicate) Spec {
	return SpecFunc(func(query *dbx.SelectQuery) *dbx.SelectQuery { return query.Where(predicate) })
}

func OrderBy(orders ...dbx.Order) Spec {
	return SpecFunc(func(query *dbx.SelectQuery) *dbx.SelectQuery { return query.OrderBy(orders...) })
}

func Limit(limit int) Spec {
	return SpecFunc(func(query *dbx.SelectQuery) *dbx.SelectQuery { return query.Limit(limit) })
}

func Offset(offset int) Spec {
	return SpecFunc(func(query *dbx.SelectQuery) *dbx.SelectQuery { return query.Offset(offset) })
}

