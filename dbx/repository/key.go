package repository

import (
	"context"
	"database/sql"

	"github.com/DaiYuANg/arcgo/dbx"
)

type Key map[string]any

func (r *Base[E, S]) GetByID(ctx context.Context, id any) (E, error) {
	pk := r.primaryColumnName()
	query := r.defaultSelect().Where(dbx.NamedColumn[any](r.schema, pk).Eq(id))
	return r.First(ctx, query)
}

func (r *Base[E, S]) GetByKey(ctx context.Context, key Key) (E, error) {
	if len(key) == 0 {
		var zero E
		return zero, &ValidationError{Message: "key is empty"}
	}
	return r.First(ctx, r.defaultSelect().Where(keyPredicate(r.schema, key)))
}

func (r *Base[E, S]) UpdateByKey(ctx context.Context, key Key, assignments ...dbx.Assignment) (sql.Result, error) {
	if len(key) == 0 {
		return nil, &ValidationError{Message: "key is empty"}
	}
	if len(assignments) == 0 {
		return nil, ErrNilMutation
	}
	result, err := r.Update(ctx, dbx.Update(r.schema).Set(assignments...).Where(keyPredicate(r.schema, key)))
	if err != nil {
		return nil, err
	}
	if r.byIDNotFoundAsError && !hasAffectedRows(result) {
		return nil, ErrNotFound
	}
	return result, nil
}

func (r *Base[E, S]) DeleteByKey(ctx context.Context, key Key) (sql.Result, error) {
	if len(key) == 0 {
		return nil, &ValidationError{Message: "key is empty"}
	}
	result, err := r.Delete(ctx, dbx.DeleteFrom(r.schema).Where(keyPredicate(r.schema, key)))
	if err != nil {
		return nil, err
	}
	if r.byIDNotFoundAsError && !hasAffectedRows(result) {
		return nil, ErrNotFound
	}
	return result, nil
}

