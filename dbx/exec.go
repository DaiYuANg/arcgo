package dbx

import (
	"context"
	"database/sql"
	"errors"

	"github.com/DaiYuANg/arcgo/collectionx"
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
		logRuntimeNode(session, "build.error", "error", ErrNilQuery)
		return BoundQuery{}, ErrNilQuery
	}
	logRuntimeNode(session, "build.start")
	bound, err := query.Build(session.Dialect())
	if err != nil {
		logRuntimeNode(session, "build.error", "error", err)
		return BoundQuery{}, wrapDBError("build query", err)
	}
	logRuntimeNode(session, "build.done", "sql_empty", bound.SQL == "", "args_count", bound.Args.Len())
	return bound, nil
}

func Exec(ctx context.Context, session Session, query QueryBuilder) (sql.Result, error) {
	bound, err := Build(session, query)
	if err != nil {
		return nil, err
	}
	logRuntimeNode(session, "exec.bound_ready", "statement", bound.Name, "args_count", bound.Args.Len())
	return ExecBound(ctx, session, bound)
}

// ExecBound executes a pre-built BoundQuery. Use with Build for reuse when
// executing the same query multiple times (e.g. in a loop).
func ExecBound(ctx context.Context, session Session, bound BoundQuery) (sql.Result, error) {
	if session == nil {
		return nil, ErrNilDB
	}
	logRuntimeNode(session, "exec_bound.start", "statement", bound.Name, "args_count", bound.Args.Len())
	result, err := session.ExecBoundContext(ctx, bound)
	return result, wrapDBError("execute bound query", err)
}

func QueryAll[E any](ctx context.Context, session Session, query QueryBuilder, mapper RowsScanner[E]) (collectionx.List[E], error) {
	if mapper == nil {
		return nil, ErrNilMapper
	}
	bound, err := Build(session, query)
	if err != nil {
		return nil, err
	}
	return QueryAllBound(ctx, session, bound, mapper)
}

// QueryAllList builds a query and maps all rows into a collectionx.List.
func QueryAllList[E any](ctx context.Context, session Session, query QueryBuilder, mapper RowsScanner[E]) (collectionx.List[E], error) {
	if mapper == nil {
		return nil, ErrNilMapper
	}
	bound, err := Build(session, query)
	if err != nil {
		return nil, err
	}
	return QueryAllBoundList(ctx, session, bound, mapper)
}

// QueryAllBound executes a pre-built BoundQuery and maps all rows. Use with Build
// for reuse when executing the same query multiple times.
// When bound.CapacityHint > 0 and mapper implements CapacityHintScanner, uses
// pre-allocated slice to reduce append growth.
func QueryAllBound[E any](ctx context.Context, session Session, bound BoundQuery, mapper RowsScanner[E]) (collectionx.List[E], error) {
	if mapper == nil {
		return nil, ErrNilMapper
	}
	if session == nil {
		return nil, ErrNilDB
	}
	rows, err := session.QueryBoundContext(ctx, bound)
	if err != nil {
		logRuntimeNode(session, "query_all_bound.query_error", "statement", bound.Name, "error", err)
		return nil, wrapDBError("query bound rows", err)
	}
	if withCap, ok := capacityHintScannerFor(mapper, bound.CapacityHint); ok {
		return scanAllBoundWithCapacity(session, rows, bound, withCap)
	}
	logRuntimeNode(session, "query_all_bound.scan")
	items, scanErr := mapper.ScanRows(rows)
	scanErr = errors.Join(wrapDBError("scan rows", scanErr), rowsIterError(rows))
	closeErr := closeRows(rows)
	if scanErr != nil {
		scanErr = errors.Join(scanErr, closeErr)
		logRuntimeNode(session, "query_all_bound.scan_error", "error", scanErr)
		return nil, scanErr
	}
	if closeErr != nil {
		logRuntimeNode(session, "query_all_bound.scan_error", "error", closeErr)
		return nil, closeErr
	}
	logRuntimeNode(session, "query_all_bound.scan_done", "items", items.Len())
	return items, nil
}

// QueryAllBoundList executes a pre-built BoundQuery and maps all rows into a collectionx.List.
func QueryAllBoundList[E any](ctx context.Context, session Session, bound BoundQuery, mapper RowsScanner[E]) (collectionx.List[E], error) {
	items, err := QueryAllBound(ctx, session, bound, mapper)
	if err != nil {
		return nil, err
	}
	return collectionx.NewList(items...), nil
}

func capacityHintScannerFor[E any](mapper RowsScanner[E], capacityHint int) (CapacityHintScanner[E], bool) {
	if capacityHint <= 0 {
		return nil, false
	}
	withCap, ok := any(mapper).(CapacityHintScanner[E])
	return withCap, ok
}

func scanAllBoundWithCapacity[E any](session Session, rows *sql.Rows, bound BoundQuery, mapper CapacityHintScanner[E]) (collectionx.List[E], error) {
	logRuntimeNode(session, "query_all_bound.scan_with_capacity", "capacity_hint", bound.CapacityHint)
	items, scanErr := mapper.ScanRowsWithCapacity(rows, bound.CapacityHint)
	scanErr = errors.Join(wrapDBError("scan rows with capacity", scanErr), rowsIterError(rows))
	closeErr := closeRows(rows)
	if scanErr != nil {
		scanErr = errors.Join(scanErr, closeErr)
		logRuntimeNode(session, "query_all_bound.scan_error", "error", scanErr)
		return nil, scanErr
	}
	if closeErr != nil {
		logRuntimeNode(session, "query_all_bound.scan_error", "error", closeErr)
		return nil, closeErr
	}
	logRuntimeNode(session, "query_all_bound.scan_done", "items", items.Len())
	return items, nil
}
