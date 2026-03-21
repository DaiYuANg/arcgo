package dbx

import (
	"context"
	"database/sql"

	"github.com/DaiYuANg/arcgo/dbx/dialect"
)

type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
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
		return BoundQuery{}, nil
	}
	return query.Build(session.Dialect())
}

func Exec(ctx context.Context, session Session, query QueryBuilder) (sql.Result, error) {
	bound, err := Build(session, query)
	if err != nil {
		return nil, err
	}
	return ExecBound(ctx, session, bound)
}

// ExecBound executes a pre-built BoundQuery. Use with Build for reuse when
// executing the same query multiple times (e.g. in a loop).
func ExecBound(ctx context.Context, session Session, bound BoundQuery) (sql.Result, error) {
	if session == nil {
		return nil, ErrNilDB
	}
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
func QueryAllBound[E any](ctx context.Context, session Session, bound BoundQuery, mapper RowsScanner[E]) ([]E, error) {
	if mapper == nil {
		return nil, ErrNilMapper
	}
	if session == nil {
		return nil, ErrNilDB
	}
	rows, err := session.QueryBoundContext(ctx, bound)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return mapper.ScanRows(rows)
}
