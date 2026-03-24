package repository

import (
	"context"
	"database/sql"

	"github.com/DaiYuANg/arcgo/dbx"
)

func (r *Base[E, S]) InTx(ctx context.Context, opts *sql.TxOptions, fn func(tx *dbx.Tx, repo *Base[E, S]) error) error {
	if r == nil || r.db == nil {
		return dbx.ErrNilDB
	}
	if fn == nil {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, opts)
	if err != nil {
		return err
	}
	txRepo := &Base[E, S]{
		db:                  r.db,
		session:             tx,
		schema:              r.schema,
		mapper:              r.mapper,
		byIDNotFoundAsError: r.byIDNotFoundAsError,
	}
	runErr := fn(tx, txRepo)
	if runErr != nil {
		_ = tx.Rollback()
		return runErr
	}
	return tx.Commit()
}

