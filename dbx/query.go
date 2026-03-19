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

type InsertQuery struct {
	Into           Table
	TargetColumns  []Expression
	Assignments    []Assignment
	Rows           [][]Assignment
	Source         *SelectQuery
	Upsert         *UpsertClause
	ReturningItems []SelectItem
}

type UpdateQuery struct {
	Table          Table
	Assignments    []Assignment
	WhereExp       Predicate
	ReturningItems []SelectItem
}

type DeleteQuery struct {
	From           Table
	WhereExp       Predicate
	ReturningItems []SelectItem
}

type JoinBuilder struct {
	query *SelectQuery
	index int
}

type ConflictBuilder struct {
	query *InsertQuery
}

type UpsertClause struct {
	Targets     []Expression
	DoNothing   bool
	Assignments []Assignment
}

func Select(items ...SelectItem) *SelectQuery {
	return &SelectQuery{Items: compactSelectItems(items)}
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
	q.CTEs = append(q.CTEs, CTE{Name: name, Query: query})
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
	q.Groups = append(q.Groups, compactExpressions(expressions)...)
	return q
}

func (q *SelectQuery) Having(predicate Predicate) *SelectQuery {
	q.HavingExp = predicate
	return q
}

func (q *SelectQuery) OrderBy(orders ...Order) *SelectQuery {
	q.Orders = append(q.Orders, compactOrders(orders)...)
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
	q.Unions = append(q.Unions, UnionClause{Query: query})
	return q
}

func (q *SelectQuery) UnionAll(query *SelectQuery) *SelectQuery {
	q.Unions = append(q.Unions, UnionClause{All: true, Query: query})
	return q
}

func (q *SelectQuery) Join(source TableSource) *JoinBuilder {
	q.Joins = append(q.Joins, Join{Type: InnerJoin, Table: source.tableRef()})
	return &JoinBuilder{query: q, index: len(q.Joins) - 1}
}

func (q *SelectQuery) LeftJoin(source TableSource) *JoinBuilder {
	q.Joins = append(q.Joins, Join{Type: LeftJoin, Table: source.tableRef()})
	return &JoinBuilder{query: q, index: len(q.Joins) - 1}
}

func (q *SelectQuery) RightJoin(source TableSource) *JoinBuilder {
	q.Joins = append(q.Joins, Join{Type: RightJoin, Table: source.tableRef()})
	return &JoinBuilder{query: q, index: len(q.Joins) - 1}
}

func (b *JoinBuilder) On(predicate Predicate) *SelectQuery {
	b.query.Joins[b.index].Predicate = predicate
	return b.query
}

func InsertInto(source TableSource) *InsertQuery {
	return &InsertQuery{Into: source.tableRef()}
}

func (q *InsertQuery) Columns(columns ...Expression) *InsertQuery {
	q.TargetColumns = append(q.TargetColumns, compactExpressions(columns)...)
	return q
}

func (q *InsertQuery) Values(assignments ...Assignment) *InsertQuery {
	row := compactAssignments(assignments)
	q.Rows = append(q.Rows, row)
	if len(q.Rows) == 1 {
		q.Assignments = row
	} else {
		q.Assignments = nil
	}
	return q
}

func (q *InsertQuery) FromSelect(query *SelectQuery) *InsertQuery {
	q.Source = query
	return q
}

func (q *InsertQuery) Returning(items ...SelectItem) *InsertQuery {
	q.ReturningItems = append(q.ReturningItems, compactSelectItems(items)...)
	return q
}

func (q *InsertQuery) OnConflict(targets ...Expression) *ConflictBuilder {
	q.Upsert = &UpsertClause{Targets: compactExpressions(targets)}
	return &ConflictBuilder{query: q}
}

func (b *ConflictBuilder) DoNothing() *InsertQuery {
	b.query.Upsert = &UpsertClause{
		Targets:   b.query.Upsert.Targets,
		DoNothing: true,
	}
	return b.query
}

func (b *ConflictBuilder) DoUpdateSet(assignments ...Assignment) *InsertQuery {
	b.query.Upsert = &UpsertClause{
		Targets:     b.query.Upsert.Targets,
		Assignments: compactAssignments(assignments),
	}
	return b.query
}

func Update(source TableSource) *UpdateQuery {
	return &UpdateQuery{Table: source.tableRef()}
}

func (q *UpdateQuery) Set(assignments ...Assignment) *UpdateQuery {
	q.Assignments = append(q.Assignments, compactAssignments(assignments)...)
	return q
}

func (q *UpdateQuery) Where(predicate Predicate) *UpdateQuery {
	q.WhereExp = predicate
	return q
}

func (q *UpdateQuery) Returning(items ...SelectItem) *UpdateQuery {
	q.ReturningItems = append(q.ReturningItems, compactSelectItems(items)...)
	return q
}

func DeleteFrom(source TableSource) *DeleteQuery {
	return &DeleteQuery{From: source.tableRef()}
}

func (q *DeleteQuery) Where(predicate Predicate) *DeleteQuery {
	q.WhereExp = predicate
	return q
}

func (q *DeleteQuery) Returning(items ...SelectItem) *DeleteQuery {
	q.ReturningItems = append(q.ReturningItems, compactSelectItems(items)...)
	return q
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

func compactAssignments(assignments []Assignment) []Assignment {
	return lo.Filter(assignments, func(assignment Assignment, _ int) bool {
		return assignment != nil
	})
}
