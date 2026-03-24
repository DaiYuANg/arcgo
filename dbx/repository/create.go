package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/DaiYuANg/arcgo/dbx"
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
	targetSet := make(map[string]struct{}, len(targets))
	targetExpressions := make([]dbx.Expression, 0, len(targets))
	for _, column := range targets {
		name := strings.TrimSpace(column)
		if name == "" {
			continue
		}
		targetSet[name] = struct{}{}
		targetExpressions = append(targetExpressions, dbx.NamedColumn[any](r.schema, name))
	}
	if len(targetExpressions) == 0 {
		return &ValidationError{Message: "upsert requires conflict columns"}
	}
	updateAssignments := make([]dbx.Assignment, 0, len(r.mapper.Fields()))
	for _, field := range r.mapper.Fields() {
		if _, conflictTarget := targetSet[field.Column]; conflictTarget {
			continue
		}
		updateAssignments = append(updateAssignments, dbx.NamedColumn[any](r.schema, field.Column).SetExcluded())
	}
	if len(updateAssignments) == 0 {
		query.OnConflict(targetExpressions...).DoNothing()
	} else {
		query.OnConflict(targetExpressions...).DoUpdateSet(updateAssignments...)
	}
	_, err = dbx.Exec(ctx, r.session, query)
	return wrapMutationError(err)
}

