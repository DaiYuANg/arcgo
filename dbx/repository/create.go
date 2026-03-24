package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/samber/lo"
)

func (r *Base[E, S]) Create(ctx context.Context, entity *E) error {
	if r == nil || r.session == nil {
		return dbx.ErrNilDB
	}
	if entity == nil {
		return &ValidationError{Message: "entity is nil"}
	}
	assignments, err := r.mapper.InsertAssignments(r.session, r.schema, entity)
	if err != nil {
		return err
	}
	_, err = dbx.Exec(ctx, r.session, dbx.InsertInto(r.schema).Values(assignments...))
	return wrapMutationError(err)
}

func (r *Base[E, S]) CreateMany(ctx context.Context, entities ...*E) error {
	if r == nil || r.session == nil {
		return dbx.ErrNilDB
	}
	if len(entities) == 0 {
		return nil
	}
	query := dbx.InsertInto(r.schema)
	for index, entity := range entities {
		if entity == nil {
			return &ValidationError{Message: fmt.Sprintf("entity at index %d is nil", index)}
		}
		assignments, err := r.mapper.InsertAssignments(r.session, r.schema, entity)
		if err != nil {
			return err
		}
		query.Values(assignments...)
	}
	_, err := dbx.Exec(ctx, r.session, query)
	return wrapMutationError(err)
}

func (r *Base[E, S]) Upsert(ctx context.Context, entity *E, conflictColumns ...string) error {
	if r == nil || r.session == nil {
		return dbx.ErrNilDB
	}
	if entity == nil {
		return &ValidationError{Message: "entity is nil"}
	}
	assignments, err := r.mapper.InsertAssignments(r.session, r.schema, entity)
	if err != nil {
		return err
	}
	query := dbx.InsertInto(r.schema).Values(assignments...)
	targets := conflictColumns
	if len(targets) == 0 {
		targets = r.primaryKeyColumns()
	}
	targetColumns := lo.Uniq(lo.FilterMap(targets, func(column string, _ int) (string, bool) {
		name := strings.TrimSpace(column)
		return name, name != ""
	}))
	if len(targetColumns) == 0 {
		return &ValidationError{Message: "upsert requires conflict columns"}
	}
	targetSet := lo.SliceToMap(targetColumns, func(column string) (string, struct{}) {
		return column, struct{}{}
	})
	targetExpressions := lo.Map(targetColumns, func(column string, _ int) dbx.Expression {
		return dbx.NamedColumn[any](r.schema, column)
	})
	updateAssignments := lo.FilterMap(r.mapper.Fields(), func(field dbx.MappedField, _ int) (dbx.Assignment, bool) {
		_, conflictTarget := targetSet[field.Column]
		if conflictTarget {
			return nil, false
		}
		return dbx.NamedColumn[any](r.schema, field.Column).SetExcluded(), true
	})
	if len(updateAssignments) == 0 {
		query.OnConflict(targetExpressions...).DoNothing()
	} else {
		query.OnConflict(targetExpressions...).DoUpdateSet(updateAssignments...)
	}
	_, err = dbx.Exec(ctx, r.session, query)
	return wrapMutationError(err)
}

