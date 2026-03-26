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
	dbx.LogRuntimeNode(r.session, "repository.update.start", "table", r.schema.TableName())
	result, err := dbx.Exec(ctx, r.session, query)
	if err != nil {
		wrapped := wrapMutationError(err)
		dbx.LogRuntimeNode(r.session, "repository.update.error", "table", r.schema.TableName(), "error", wrapped)
		return result, wrapped
	}
	dbx.LogRuntimeNode(r.session, "repository.update.done", "table", r.schema.TableName())
	return result, nil
}

func (r *Base[E, S]) Delete(ctx context.Context, query *dbx.DeleteQuery) (sql.Result, error) {
	if r == nil || r.session == nil {
		return nil, dbx.ErrNilDB
	}
	if query == nil {
		return nil, ErrNilMutation
	}
	dbx.LogRuntimeNode(r.session, "repository.delete.start", "table", r.schema.TableName())
	result, err := dbx.Exec(ctx, r.session, query)
	if err != nil {
		wrapped := wrapMutationError(err)
		dbx.LogRuntimeNode(r.session, "repository.delete.error", "table", r.schema.TableName(), "error", wrapped)
		return result, wrapped
	}
	dbx.LogRuntimeNode(r.session, "repository.delete.done", "table", r.schema.TableName())
	return result, nil
}

func (r *Base[E, S]) UpdateByID(ctx context.Context, id any, assignments ...dbx.Assignment) (sql.Result, error) {
	if len(assignments) == 0 {
		return nil, ErrNilMutation
	}
	pk := r.primaryColumnName()
	result, err := r.Update(ctx, dbx.Update(r.schema).Set(assignments...).Where(dbx.NamedColumn[any](r.schema, pk).Eq(id)))
	if err != nil {
		dbx.LogRuntimeNode(r.session, "repository.update_by_id.error", "table", r.schema.TableName(), "error", err)
		return nil, err
	}
	if r.byIDNotFoundAsError && !hasAffectedRows(result) {
		dbx.LogRuntimeNode(r.session, "repository.update_by_id.not_found", "table", r.schema.TableName())
		return nil, ErrNotFound
	}
	dbx.LogRuntimeNode(r.session, "repository.update_by_id.done", "table", r.schema.TableName())
	return result, nil
}

func (r *Base[E, S]) DeleteByID(ctx context.Context, id any) (sql.Result, error) {
	pk := r.primaryColumnName()
	result, err := r.Delete(ctx, dbx.DeleteFrom(r.schema).Where(dbx.NamedColumn[any](r.schema, pk).Eq(id)))
	if err != nil {
		dbx.LogRuntimeNode(r.session, "repository.delete_by_id.error", "table", r.schema.TableName(), "error", err)
		return nil, err
	}
	if r.byIDNotFoundAsError && !hasAffectedRows(result) {
		dbx.LogRuntimeNode(r.session, "repository.delete_by_id.not_found", "table", r.schema.TableName())
		return nil, ErrNotFound
	}
	dbx.LogRuntimeNode(r.session, "repository.delete_by_id.done", "table", r.schema.TableName())
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
		dbx.LogRuntimeNode(r.session, "repository.update_by_version.error", "table", r.schema.TableName(), "error", err)
		return nil, err
	}
	if !hasAffectedRows(result) {
		dbx.LogRuntimeNode(r.session, "repository.update_by_version.conflict", "table", r.schema.TableName(), "version", currentVersion)
		return nil, &VersionConflictError{Err: ErrVersionConflict}
	}
	dbx.LogRuntimeNode(r.session, "repository.update_by_version.done", "table", r.schema.TableName(), "version", nextVersion)
	return result, nil
}
