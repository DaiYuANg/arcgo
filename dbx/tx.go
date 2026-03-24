package dbx

import (
	"context"
	"database/sql"
	"log/slog"
	"slices"

	"github.com/DaiYuANg/arcgo/dbx/dialect"
)

type Tx struct {
	raw      *sql.Tx
	dialect  dialect.Dialect
	observe  runtimeObserver
	relation *relationRuntime
	idGenerator IDGenerator
	nodeID      uint16
}

func (tx *Tx) SQLTx() *sql.Tx {
	return tx.raw
}

func (tx *Tx) Dialect() dialect.Dialect {
	return tx.dialect
}

func (tx *Tx) Bound(sql string, args ...any) BoundQuery {
	return BoundQuery{SQL: sql, Args: slices.Clone(args)}
}

func (tx *Tx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return tx.queryContext(ctx, "", query, args...)
}

func (tx *Tx) queryContext(ctx context.Context, statement string, query string, args ...any) (*sql.Rows, error) {
	if tx == nil {
		return nil, ErrNilDB
	}
	if tx.raw == nil {
		return nil, ErrNilSQLDB
	}
	ctx, event, err := tx.observe.before(ctx, HookEvent{
		Operation: OperationQuery,
		Statement: statement,
		SQL:       query,
		Args:      args,
	})
	if err != nil {
		return nil, err
	}
	rows, queryErr := tx.raw.QueryContext(ctx, query, args...)
	event.Err = queryErr
	tx.observe.after(ctx, event)
	return rows, queryErr
}

func (tx *Tx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return tx.execContext(ctx, "", query, args...)
}

func (tx *Tx) execContext(ctx context.Context, statement string, query string, args ...any) (sql.Result, error) {
	if tx == nil {
		return nil, ErrNilDB
	}
	if tx.raw == nil {
		return nil, ErrNilSQLDB
	}
	ctx, event, err := tx.observe.before(ctx, HookEvent{
		Operation: OperationExec,
		Statement: statement,
		SQL:       query,
		Args:      args,
	})
	if err != nil {
		return nil, err
	}
	result, execErr := tx.raw.ExecContext(ctx, query, args...)
	if execErr == nil && result != nil {
		if rowsAffected, rowsErr := result.RowsAffected(); rowsErr == nil {
			event.RowsAffected = rowsAffected
			event.HasRowsAffected = true
		}
	}
	event.Err = execErr
	tx.observe.after(ctx, event)
	return result, execErr
}

func (tx *Tx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	if tx == nil || tx.raw == nil {
		return nil
	}
	ctx, event, err := tx.observe.before(ctx, HookEvent{Operation: OperationQueryRow, SQL: query, Args: args})
	if err != nil {
		return nil
	}
	row := tx.raw.QueryRowContext(ctx, query, args...)
	tx.observe.after(ctx, event)
	return row
}

func (tx *Tx) QueryBoundContext(ctx context.Context, bound BoundQuery) (*sql.Rows, error) {
	return tx.queryContext(ctx, bound.Name, bound.SQL, bound.Args...)
}

func (tx *Tx) ExecBoundContext(ctx context.Context, bound BoundQuery) (sql.Result, error) {
	return tx.execContext(ctx, bound.Name, bound.SQL, bound.Args...)
}

func (tx *Tx) Commit() error {
	if tx == nil || tx.raw == nil {
		return ErrNilSQLDB
	}
	ctx, event, err := tx.observe.before(context.Background(), HookEvent{Operation: OperationCommitTx})
	if err != nil {
		return err
	}
	commitErr := tx.raw.Commit()
	event.Err = commitErr
	tx.observe.after(ctx, event)
	return commitErr
}

func (tx *Tx) Rollback() error {
	if tx == nil || tx.raw == nil {
		return ErrNilSQLDB
	}
	ctx, event, err := tx.observe.before(context.Background(), HookEvent{Operation: OperationRollbackTx})
	if err != nil {
		return err
	}
	rollbackErr := tx.raw.Rollback()
	event.Err = rollbackErr
	tx.observe.after(ctx, event)
	return rollbackErr
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
