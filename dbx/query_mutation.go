package dbx

import "github.com/samber/lo"

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

type ConflictBuilder struct {
	query *InsertQuery
}

type UpsertClause struct {
	Targets     []Expression
	DoNothing   bool
	Assignments []Assignment
}

func InsertInto(source TableSource) *InsertQuery {
	return &InsertQuery{Into: source.tableRef()}
}

func (q *InsertQuery) Columns(columns ...Expression) *InsertQuery {
	q.TargetColumns = lo.Concat(q.TargetColumns, compactExpressions(columns))
	return q
}

func (q *InsertQuery) Values(assignments ...Assignment) *InsertQuery {
	row := compactAssignments(assignments)
	q.Rows = lo.Concat(q.Rows, [][]Assignment{row})
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
	q.ReturningItems = lo.Concat(q.ReturningItems, compactSelectItems(items))
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
	q.Assignments = lo.Concat(q.Assignments, compactAssignments(assignments))
	return q
}

func (q *UpdateQuery) Where(predicate Predicate) *UpdateQuery {
	q.WhereExp = predicate
	return q
}

func (q *UpdateQuery) Returning(items ...SelectItem) *UpdateQuery {
	q.ReturningItems = lo.Concat(q.ReturningItems, compactSelectItems(items))
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
	q.ReturningItems = lo.Concat(q.ReturningItems, compactSelectItems(items))
	return q
}

func compactAssignments(assignments []Assignment) []Assignment {
	return lo.Filter(assignments, func(assignment Assignment, _ int) bool {
		return assignment != nil
	})
}
