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
	dbx.LogRuntimeNode(r.session, "repository.create.start", "table", r.schema.TableName())
	assignments, err := r.mapper.InsertAssignments(r.session, r.schema, entity)
	if err != nil {
		dbx.LogRuntimeNode(r.session, "repository.create.error", "table", r.schema.TableName(), "stage", "assignments", "error", err)
		return err
	}
	_, err = dbx.Exec(ctx, r.session, dbx.InsertInto(r.schema).Values(assignments...))
	if err != nil {
		wrapped := wrapMutationError(err)
		dbx.LogRuntimeNode(r.session, "repository.create.error", "table", r.schema.TableName(), "stage", "exec", "error", wrapped)
		return wrapped
	}
	dbx.LogRuntimeNode(r.session, "repository.create.done", "table", r.schema.TableName())
	return nil
}

func (r *Base[E, S]) CreateMany(ctx context.Context, entities ...*E) error {
	if r == nil || r.session == nil {
		return dbx.ErrNilDB
	}
	if len(entities) == 0 {
		return nil
	}
	dbx.LogRuntimeNode(r.session, "repository.create_many.start", "table", r.schema.TableName(), "entities", len(entities))
	query := dbx.InsertInto(r.schema)
	for index, entity := range entities {
		if entity == nil {
			return &ValidationError{Message: fmt.Sprintf("entity at index %d is nil", index)}
		}
		assignments, err := r.mapper.InsertAssignments(r.session, r.schema, entity)
		if err != nil {
			dbx.LogRuntimeNode(r.session, "repository.create_many.error", "table", r.schema.TableName(), "stage", "assignments", "index", index, "error", err)
			return err
		}
		query.Values(assignments...)
	}
	_, err := dbx.Exec(ctx, r.session, query)
	if err != nil {
		wrapped := wrapMutationError(err)
		dbx.LogRuntimeNode(r.session, "repository.create_many.error", "table", r.schema.TableName(), "stage", "exec", "error", wrapped)
		return wrapped
	}
	dbx.LogRuntimeNode(r.session, "repository.create_many.done", "table", r.schema.TableName(), "entities", len(entities))
	return nil
}

func (r *Base[E, S]) Upsert(ctx context.Context, entity *E, conflictColumns ...string) error {
	if r == nil || r.session == nil {
		return dbx.ErrNilDB
	}
	if entity == nil {
		return &ValidationError{Message: "entity is nil"}
	}
	dbx.LogRuntimeNode(r.session, "repository.upsert.start", "table", r.schema.TableName(), "conflict_columns", conflictColumns)
	assignments, err := r.mapper.InsertAssignments(r.session, r.schema, entity)
	if err != nil {
		dbx.LogRuntimeNode(r.session, "repository.upsert.error", "table", r.schema.TableName(), "stage", "assignments", "error", err)
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
	if err != nil {
		wrapped := wrapMutationError(err)
		dbx.LogRuntimeNode(r.session, "repository.upsert.error", "table", r.schema.TableName(), "stage", "exec", "error", wrapped)
		return wrapped
	}
	dbx.LogRuntimeNode(r.session, "repository.upsert.done", "table", r.schema.TableName(), "conflict_columns", targetColumns)
	return nil
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
