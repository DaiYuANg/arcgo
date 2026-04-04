package dbx

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dbx/dialect"
)

type Tx struct {
	raw         *sql.Tx
	dialect     dialect.Dialect
	observe     runtimeObserver
	relation    *relationRuntime
	idGenerator IDGenerator
	nodeID      uint16
}

func (tx *Tx) SQLTx() *sql.Tx {
	return tx.raw
}

func (tx *Tx) Dialect() dialect.Dialect {
	return tx.dialect
}

func (tx *Tx) Bound(rawSQL string, args ...any) BoundQuery {
	return BoundQuery{SQL: rawSQL, Args: collectionx.NewList(args...)}
}

func (tx *Tx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return tx.queryContext(ctx, "", query, args...)
}

func (tx *Tx) queryContext(ctx context.Context, statement, query string, args ...any) (*sql.Rows, error) {
	if tx == nil {
		return nil, ErrNilDB
	}
	if tx.raw == nil {
		return nil, ErrNilSQLDB
	}

	return observedQueryContext(ctx, tx.observe, statement, query, args, tx.raw.QueryContext)
}

func (tx *Tx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return tx.execContext(ctx, "", query, args...)
}

func (tx *Tx) execContext(ctx context.Context, statement, query string, args ...any) (sql.Result, error) {
	if tx == nil {
		return nil, ErrNilDB
	}
	if tx.raw == nil {
		return nil, ErrNilSQLDB
	}

	return observedExecContext(ctx, tx.observe, statement, query, args, tx.raw.ExecContext)
}

func (tx *Tx) QueryRowContext(ctx context.Context, query string, args ...any) *Row {
	if tx == nil {
		return errorRow(ErrNilDB)
	}
	if tx.raw == nil {
		return errorRow(ErrNilSQLDB)
	}
	ctx, event, err := tx.observe.before(ctx, HookEvent{Operation: OperationQueryRow, SQL: query, Args: collectionx.NewList(args...)})
	if err != nil {
		tx.observe.after(ctx, event)
		return errorRow(err)
	}
	rows, queryErr := tx.raw.QueryContext(ctx, query, args...)
	if queryErr != nil {
		event.Err = queryErr
		tx.observe.after(ctx, event)
		return errorRow(queryErr)
	}
	return observedRow(ctx, tx.observe, event, rows)
}

func (tx *Tx) QueryBoundContext(ctx context.Context, bound BoundQuery) (*sql.Rows, error) {
	return tx.queryContext(ctx, bound.Name, bound.SQL, bound.Args.Values()...)
}

func (tx *Tx) ExecBoundContext(ctx context.Context, bound BoundQuery) (sql.Result, error) {
	return tx.execContext(ctx, bound.Name, bound.SQL, bound.Args.Values()...)
}

// Commit commits the transaction using a background context.
func (tx *Tx) Commit() error {
	return tx.CommitContext(context.Background())
}

// CommitContext commits the transaction using the provided context.
func (tx *Tx) CommitContext(ctx context.Context) error {
	if tx == nil || tx.raw == nil {
		return ErrNilSQLDB
	}
	ctx, err := requireContext(ctx, "commit transaction")
	if err != nil {
		return err
	}
	ctx, event, err := tx.observe.before(ctx, HookEvent{Operation: OperationCommitTx})
	if err != nil {
		tx.observe.after(ctx, event)
		return err
	}
	commitErr := tx.raw.Commit()
	event.Err = wrapDBError("commit transaction", commitErr)
	tx.observe.after(ctx, event)
	return event.Err
}

// Rollback rolls the transaction back using a background context.
func (tx *Tx) Rollback() error {
	return tx.RollbackContext(context.Background())
}

// RollbackContext rolls the transaction back using the provided context.
func (tx *Tx) RollbackContext(ctx context.Context) error {
	if tx == nil || tx.raw == nil {
		return ErrNilSQLDB
	}
	ctx, err := requireContext(ctx, "rollback transaction")
	if err != nil {
		return err
	}
	ctx, event, err := tx.observe.before(ctx, HookEvent{Operation: OperationRollbackTx})
	if err != nil {
		tx.observe.after(ctx, event)
		return err
	}
	rollbackErr := tx.raw.Rollback()
	event.Err = wrapDBError("rollback transaction", rollbackErr)
	tx.observe.after(ctx, event)
	return event.Err
}

func (tx *Tx) SQL() *SQLExecutor {
	return &SQLExecutor{session: tx}
}

func (tx *Tx) Logger() *slog.Logger {
	return tx.observe.logger
}

func (tx *Tx) Debug() bool {
	return tx.observe.debug
}

func (tx *Tx) IDGenerator() IDGenerator {
	if tx == nil {
		return nil
	}
	return tx.idGenerator
}

func (tx *Tx) NodeID() uint16 {
	if tx == nil {
		return 0
	}
	return tx.nodeID
}

// RelationRuntime returns the relation load runtime for this Tx.
func (tx *Tx) RelationRuntime() *relationRuntime {
	if tx == nil || tx.relation == nil {
		return defaultRelationRuntime
	}
	return tx.relation
}
