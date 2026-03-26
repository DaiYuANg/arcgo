package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
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
	targetColumns := normalizeConflictColumns(conflictColumns, r.primaryKeyColumns())
	if len(targetColumns) == 0 {
		return &ValidationError{Message: "upsert requires conflict columns"}
	}
	targetExpressions := lo.Map(targetColumns, func(column string, _ int) dbx.Expression {
		return dbx.NamedColumn[any](r.schema, column)
	})
	updateAssignments := upsertUpdateAssignments(r.schema, r.mapper.Fields(), targetColumns)
	if len(updateAssignments) == 0 {
		query.OnConflict(targetExpressions...).DoNothing()
	} else {
		query.OnConflict(targetExpressions...).DoUpdateSet(updateAssignments...)
	}
	_, err = dbx.Exec(ctx, r.session, query)
	return wrapMutationError(err)
}

func normalizeConflictColumns(columns []string, fallback []string) []string {
	if len(columns) == 0 {
		columns = fallback
	}
	ordered := collectionx.NewOrderedSet[string]()
	for _, column := range columns {
		if name := strings.TrimSpace(column); name != "" {
			ordered.Add(name)
		}
	}
	return ordered.Values()
}

func upsertUpdateAssignments[S dbx.TableSource](schema S, fields []dbx.MappedField, conflictColumns []string) []dbx.Assignment {
	conflictSet := collectionx.NewSetWithCapacity[string](len(conflictColumns))
	conflictSet.Add(conflictColumns...)
	return lo.FilterMap(fields, func(field dbx.MappedField, _ int) (dbx.Assignment, bool) {
		if conflictSet.Contains(field.Column) {
			return nil, false
		}
		return dbx.NamedColumn[any](schema, field.Column).SetExcluded(), true
	})
}
