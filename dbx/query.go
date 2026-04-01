package dbx

import "github.com/samber/lo"

type Join struct {
	Type      JoinType
	Table     Table
	Predicate Predicate
}

type CTE struct {
	Name  string
	Query *SelectQuery
}

type UnionClause struct {
	All   bool
	Query *SelectQuery
}

type SelectQuery struct {
	Items     []SelectItem
	FromItem  Table
	Joins     []Join
	WhereExp  Predicate
	Groups    []Expression
	HavingExp Predicate
	Orders    []Order
	LimitN    *int
	OffsetN   *int
	Distinct  bool
	CTEs      []CTE
	Unions    []UnionClause
}

type JoinBuilder struct {
	query *SelectQuery
	index int
}

func Select(items ...SelectItem) *SelectQuery {
	return &SelectQuery{Items: compactSelectItems(items)}
}

func (q *SelectQuery) Clone() *SelectQuery {
	if q == nil {
		return nil
	}
	cloned := *q
	cloned.Items = append([]SelectItem(nil), q.Items...)
	cloned.Joins = append([]Join(nil), q.Joins...)
	cloned.Groups = append([]Expression(nil), q.Groups...)
	cloned.Orders = append([]Order(nil), q.Orders...)
	cloned.CTEs = cloneCTEs(q.CTEs)
	cloned.Unions = cloneUnionClauses(q.Unions)
	cloned.LimitN = cloneInt(q.LimitN)
	cloned.OffsetN = cloneInt(q.OffsetN)
	return &cloned
}

func (q *SelectQuery) WithDistinct() *SelectQuery {
	q.Distinct = true
	return q
}

func (q *SelectQuery) DistinctOn() *SelectQuery {
	q.Distinct = true
	return q
}

func (q *SelectQuery) With(name string, query *SelectQuery) *SelectQuery {
	q.CTEs = lo.Concat(q.CTEs, []CTE{{Name: name, Query: query}})
	return q
}

func (q *SelectQuery) From(source TableSource) *SelectQuery {
	q.FromItem = source.tableRef()
	return q
}

func (q *SelectQuery) Where(predicate Predicate) *SelectQuery {
	q.WhereExp = predicate
	return q
}

func (q *SelectQuery) GroupBy(expressions ...Expression) *SelectQuery {
	q.Groups = lo.Concat(q.Groups, compactExpressions(expressions))
	return q
}

func (q *SelectQuery) Having(predicate Predicate) *SelectQuery {
	q.HavingExp = predicate
	return q
}

func (q *SelectQuery) OrderBy(orders ...Order) *SelectQuery {
	q.Orders = lo.Concat(q.Orders, compactOrders(orders))
	return q
}

func (q *SelectQuery) Limit(limit int) *SelectQuery {
	q.LimitN = &limit
	return q
}

func (q *SelectQuery) Offset(offset int) *SelectQuery {
	q.OffsetN = &offset
	return q
}

func (q *SelectQuery) Union(query *SelectQuery) *SelectQuery {
	q.Unions = lo.Concat(q.Unions, []UnionClause{{Query: query}})
	return q
}

func (q *SelectQuery) UnionAll(query *SelectQuery) *SelectQuery {
	q.Unions = lo.Concat(q.Unions, []UnionClause{{All: true, Query: query}})
	return q
}

func (q *SelectQuery) Join(source TableSource) *JoinBuilder {
	q.Joins = lo.Concat(q.Joins, []Join{{Type: InnerJoin, Table: source.tableRef()}})
	return &JoinBuilder{query: q, index: len(q.Joins) - 1}
}

func (q *SelectQuery) LeftJoin(source TableSource) *JoinBuilder {
	q.Joins = lo.Concat(q.Joins, []Join{{Type: LeftJoin, Table: source.tableRef()}})
	return &JoinBuilder{query: q, index: len(q.Joins) - 1}
}

func (q *SelectQuery) RightJoin(source TableSource) *JoinBuilder {
	q.Joins = lo.Concat(q.Joins, []Join{{Type: RightJoin, Table: source.tableRef()}})
	return &JoinBuilder{query: q, index: len(q.Joins) - 1}
}

func (b *JoinBuilder) On(predicate Predicate) *SelectQuery {
	b.query.Joins[b.index].Predicate = predicate
	return b.query
}

func cloneCTEs(items []CTE) []CTE {
	if len(items) == 0 {
		return nil
	}
	return lo.Map(items, func(item CTE, _ int) CTE {
		return CTE{Name: item.Name, Query: item.Query.Clone()}
	})
}

func cloneUnionClauses(items []UnionClause) []UnionClause {
	if len(items) == 0 {
		return nil
	}
	return lo.Map(items, func(item UnionClause, _ int) UnionClause {
		return UnionClause{All: item.All, Query: item.Query.Clone()}
	})
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func compactSelectItems(items []SelectItem) []SelectItem {
	return lo.Filter(items, func(item SelectItem, _ int) bool {
		return item != nil
	})
}

func compactOrders(orders []Order) []Order {
	return lo.Filter(orders, func(order Order, _ int) bool {
		return order != nil
	})
}
