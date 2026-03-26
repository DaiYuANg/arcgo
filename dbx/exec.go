package dbx

import (
	"context"
	"database/sql"

	"github.com/DaiYuANg/arcgo/dbx/dialect"
)

type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *Row
}

type Scanner[T any] func(rows *sql.Rows) (T, error)

type Session interface {
	Executor
	Dialect() dialect.Dialect
	QueryBoundContext(ctx context.Context, bound BoundQuery) (*sql.Rows, error)
	ExecBoundContext(ctx context.Context, bound BoundQuery) (sql.Result, error)
	// SQL returns an executor for templated SQL. DB and Tx implement this for unified execution entry.
	SQL() *SQLExecutor
}

type QueryBuilder interface {
	Build(d dialect.Dialect) (BoundQuery, error)
}

// Build compiles a QueryBuilder into BoundQuery using the session's dialect.
// For "build once, execute many" reuse: call Build once, then pass the result to
// ExecBound, QueryAllBound, QueryCursorBound, or QueryEachBound in a loop.
func Build(session Session, query QueryBuilder) (BoundQuery, error) {
	if session == nil {
		return BoundQuery{}, ErrNilDB
	}
	if session.Dialect() == nil {
		return BoundQuery{}, ErrNilDialect
	}
	if query == nil {
		logRuntimeNode(session, "build.skip_nil_query")
		return BoundQuery{}, nil
	}
	logRuntimeNode(session, "build.start")
	bound, err := query.Build(session.Dialect())
	if err != nil {
		logRuntimeNode(session, "build.error", "error", err)
		return BoundQuery{}, err
	}
	logRuntimeNode(session, "build.done", "sql_empty", bound.SQL == "", "args_count", len(bound.Args))
	return bound, nil
}

func Exec(ctx context.Context, session Session, query QueryBuilder) (sql.Result, error) {
	bound, err := Build(session, query)
	if err != nil {
		return nil, err
	}
	logRuntimeNode(session, "exec.bound_ready", "statement", bound.Name, "args_count", len(bound.Args))
	return ExecBound(ctx, session, bound)
}

// ExecBound executes a pre-built BoundQuery. Use with Build for reuse when
// executing the same query multiple times (e.g. in a loop).
func ExecBound(ctx context.Context, session Session, bound BoundQuery) (sql.Result, error) {
	if session == nil {
		return nil, ErrNilDB
	}
	logRuntimeNode(session, "exec_bound.start", "statement", bound.Name, "args_count", len(bound.Args))
	return session.ExecBoundContext(ctx, bound)
}

func QueryAll[E any](ctx context.Context, session Session, query QueryBuilder, mapper RowsScanner[E]) ([]E, error) {
	if mapper == nil {
		return nil, ErrNilMapper
	}
	bound, err := Build(session, query)
	if err != nil {
		return nil, err
	}
	return QueryAllBound(ctx, session, bound, mapper)
}

// QueryAllBound executes a pre-built BoundQuery and maps all rows. Use with Build
// for reuse when executing the same query multiple times.
// When bound.CapacityHint > 0 and mapper implements CapacityHintScanner, uses
// pre-allocated slice to reduce append growth.
func QueryAllBound[E any](ctx context.Context, session Session, bound BoundQuery, mapper RowsScanner[E]) ([]E, error) {
	if mapper == nil {
		return nil, ErrNilMapper
	}
	if session == nil {
		return nil, ErrNilDB
	}
	rows, err := session.QueryBoundContext(ctx, bound)
	if err != nil {
		logRuntimeNode(session, "query_all_bound.query_error", "statement", bound.Name, "error", err)
		return nil, err
	}
	defer rows.Close()
	if bound.CapacityHint > 0 {
		if withCap, ok := any(mapper).(CapacityHintScanner[E]); ok {
			logRuntimeNode(session, "query_all_bound.scan_with_capacity", "capacity_hint", bound.CapacityHint)
			items, scanErr := withCap.ScanRowsWithCapacity(rows, bound.CapacityHint)
			if scanErr != nil {
				logRuntimeNode(session, "query_all_bound.scan_error", "error", scanErr)
				return nil, scanErr
			}
			logRuntimeNode(session, "query_all_bound.scan_done", "items", len(items))
			return items, nil
		}
	}
	logRuntimeNode(session, "query_all_bound.scan")
	items, scanErr := mapper.ScanRows(rows)
	if scanErr != nil {
		logRuntimeNode(session, "query_all_bound.scan_error", "error", scanErr)
		return nil, scanErr
	}
	logRuntimeNode(session, "query_all_bound.scan_done", "items", len(items))
	return items, nil
}
