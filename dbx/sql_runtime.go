package dbx

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/DaiYuANg/arcgo/collectionx"
)

func wrapDBError(op string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("dbx %s: %w", op, err)
}

func requireContext(ctx context.Context, op string) (context.Context, error) {
	if ctx == nil {
		return nil, fmt.Errorf("dbx %s: context is nil", op)
	}

	return ctx, nil
}

func closeRows(rows *sql.Rows) error {
	if rows == nil {
		return nil
	}

	return wrapDBError("close rows", rows.Close())
}

func rowsIterError(rows *sql.Rows) error {
	if rows == nil {
		return nil
	}

	return wrapDBError("iterate rows", rows.Err())
}

func closeCursor[T interface{ Close() error }](cursor T) error {
	return wrapDBError("close cursor", cursor.Close())
}

func observedQueryContext(
	ctx context.Context,
	observe runtimeObserver,
	statement string,
	query string,
	args []any,
	queryFn func(context.Context, string, ...any) (*sql.Rows, error),
) (*sql.Rows, error) {
	ctx, event, err := observe.before(ctx, HookEvent{
		Operation: OperationQuery,
		Statement: statement,
		SQL:       query,
		Args:      collectionx.NewList(args...),
	})
	if err != nil {
		observe.after(ctx, event)
		return nil, err
	}

	rows, queryErr := queryFn(ctx, query, args...)
	queryErr = wrapDBError("query rows", queryErr)
	event.Err = queryErr
	observe.after(ctx, event)

	return rows, queryErr
}

func observedExecContext(
	ctx context.Context,
	observe runtimeObserver,
	statement string,
	query string,
	args []any,
	execFn func(context.Context, string, ...any) (sql.Result, error),
) (sql.Result, error) {
	ctx, event, err := observe.before(ctx, HookEvent{
		Operation: OperationExec,
		Statement: statement,
		SQL:       query,
		Args:      collectionx.NewList(args...),
	})
	if err != nil {
		observe.after(ctx, event)
		return nil, err
	}

	result, execErr := execFn(ctx, query, args...)
	execErr = wrapDBError("execute query", execErr)
	if execErr == nil && result != nil {
		if rowsAffected, rowsErr := result.RowsAffected(); rowsErr == nil {
			event.RowsAffected = rowsAffected
			event.HasRowsAffected = true
		}
	}
	event.Err = execErr
	observe.after(ctx, event)

	return result, execErr
}
