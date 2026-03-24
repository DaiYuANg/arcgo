package repository

import (
	"context"
	"database/sql"

	"github.com/DaiYuANg/arcgo/dbx"
)

func (r *Base[E, S]) Update(ctx context.Context, query *dbx.UpdateQuery) (sql.Result, error) {
	if r == nil || r.session == nil {
		return nil, dbx.ErrNilDB
	}
	if query == nil {
		return nil, ErrNilMutation
	}
	result, err := dbx.Exec(ctx, r.session, query)
	return result, wrapMutationError(err)
}

func (r *Base[E, S]) Delete(ctx context.Context, query *dbx.DeleteQuery) (sql.Result, error) {
	if r == nil || r.session == nil {
		return nil, dbx.ErrNilDB
	}
	if query == nil {
		return nil, ErrNilMutation
	}
	result, err := dbx.Exec(ctx, r.session, query)
	return result, wrapMutationError(err)
}

func (r *Base[E, S]) UpdateByID(ctx context.Context, id any, assignments ...dbx.Assignment) (sql.Result, error) {
	if len(assignments) == 0 {
		return nil, ErrNilMutation
	}
	pk := r.primaryColumnName()
	result, err := r.Update(ctx, dbx.Update(r.schema).Set(assignments...).Where(dbx.NamedColumn[any](r.schema, pk).Eq(id)))
	if err != nil {
		return nil, err
	}
	if r.byIDNotFoundAsError && !hasAffectedRows(result) {
		return nil, ErrNotFound
	}
	return result, nil
}

func (r *Base[E, S]) DeleteByID(ctx context.Context, id any) (sql.Result, error) {
	pk := r.primaryColumnName()
	result, err := r.Delete(ctx, dbx.DeleteFrom(r.schema).Where(dbx.NamedColumn[any](r.schema, pk).Eq(id)))
	if err != nil {
		return nil, err
	}
	if r.byIDNotFoundAsError && !hasAffectedRows(result) {
		return nil, ErrNotFound
	}
	return result, nil
}

func (r *Base[E, S]) UpdateByVersion(ctx context.Context, key Key, currentVersion int64, assignments ...dbx.Assignment) (sql.Result, error) {
	if len(key) == 0 {
		return nil, &ValidationError{Message: "key is empty"}
	}
	if len(assignments) == 0 {
		return nil, ErrNilMutation
	}
	predicate := dbx.And(keyPredicate(r.schema, key), dbx.NamedColumn[any](r.schema, "version").Eq(currentVersion))
	nextVersion := currentVersion + 1
	assignments = append(assignments, dbx.NamedColumn[any](r.schema, "version").Set(nextVersion))
	result, err := r.Update(ctx, dbx.Update(r.schema).Set(assignments...).Where(predicate))
	if err != nil {
		return nil, err
	}
	if !hasAffectedRows(result) {
		return nil, &VersionConflictError{Err: ErrVersionConflict}
	}
	return result, nil
}

