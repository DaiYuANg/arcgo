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
}

type QueryBuilder interface {
	Build(d dialect.Dialect) (BoundQuery, error)
}

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

	rows, err := session.QueryBoundContext(ctx, bound)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return mapper.ScanRows(rows)
}
