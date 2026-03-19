package dbx

import (
	"context"
	"database/sql"

	"github.com/samber/mo"
	scanlib "github.com/stephenafamo/scan"
)

type SQLExecutor struct {
	session Session
}

type oneRowScanner[E any] interface {
	scanOneRows(ctx context.Context, rows *sql.Rows) (E, bool, error)
}

func (x *SQLExecutor) Bind(statement SQLStatementSource, params any) (BoundQuery, error) {
	if statement == nil {
		return BoundQuery{}, ErrNilStatement
	}

	bound, err := statement.Bind(params)
	if err != nil {
		return BoundQuery{}, err
	}
	if bound.Name == "" {
		bound.Name = statement.StatementName()
	}
	if len(bound.Args) > 0 {
		bound.Args = append([]any(nil), bound.Args...)
	}
	return bound, nil
}

func (x *SQLExecutor) Exec(ctx context.Context, statement SQLStatementSource, params any) (sql.Result, error) {
	session, err := x.sessionOrErr()
	if err != nil {
		return nil, err
	}

	bound, err := x.Bind(statement, params)
	if err != nil {
		return nil, err
	}
	return session.ExecBoundContext(ctx, bound)
}

func (x *SQLExecutor) Query(ctx context.Context, statement SQLStatementSource, params any) (*sql.Rows, error) {
	session, err := x.sessionOrErr()
	if err != nil {
		return nil, err
	}

	bound, err := x.Bind(statement, params)
	if err != nil {
		return nil, err
	}
	return session.QueryBoundContext(ctx, bound)
}

func (x *SQLExecutor) sessionOrErr() (Session, error) {
	if x == nil || x.session == nil {
		return nil, ErrNilDB
	}
	return x.session, nil
}

func SQLList[E any](ctx context.Context, executor *SQLExecutor, statement SQLStatementSource, params any, mapper RowsScanner[E]) ([]E, error) {
	if mapper == nil {
		return nil, ErrNilMapper
	}

	rows, err := queryStatementRows(ctx, executor, statement, params)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return mapper.ScanRows(rows)
}

func SQLGet[E any](ctx context.Context, executor *SQLExecutor, statement SQLStatementSource, params any, mapper RowsScanner[E]) (E, error) {
	if mapper == nil {
		var zero E
		return zero, ErrNilMapper
	}

	if one, ok := mapper.(oneRowScanner[E]); ok {
		value, found, err := scanStatementOne(ctx, executor, statement, params, one)
		if err != nil {
			var zero E
			return zero, err
		}
		if !found {
			var zero E
			return zero, sql.ErrNoRows
		}
		return value, nil
	}

	items, err := SQLList(ctx, executor, statement, params, mapper)
	if err != nil {
		var zero E
		return zero, err
	}

	switch len(items) {
	case 0:
		var zero E
		return zero, sql.ErrNoRows
	case 1:
		return items[0], nil
	default:
		var zero E
		return zero, ErrTooManyRows
	}
}

func SQLFind[E any](ctx context.Context, executor *SQLExecutor, statement SQLStatementSource, params any, mapper RowsScanner[E]) (mo.Option[E], error) {
	if mapper == nil {
		return mo.None[E](), ErrNilMapper
	}

	if one, ok := mapper.(oneRowScanner[E]); ok {
		value, found, err := scanStatementOne(ctx, executor, statement, params, one)
		if err != nil {
			return mo.None[E](), err
		}
		return mo.TupleToOption(value, found), nil
	}

	items, err := SQLList(ctx, executor, statement, params, mapper)
	if err != nil {
		return mo.None[E](), err
	}

	switch len(items) {
	case 0:
		return mo.None[E](), nil
	case 1:
		return mo.Some(items[0]), nil
	default:
		return mo.None[E](), ErrTooManyRows
	}
}

func SQLScalar[T any](ctx context.Context, executor *SQLExecutor, statement SQLStatementSource, params any) (T, error) {
	value, found, err := sqlScalar[T](ctx, executor, statement, params)
	if err != nil {
		var zero T
		return zero, err
	}
	if !found {
		var zero T
		return zero, sql.ErrNoRows
	}
	return value, nil
}

func SQLScalarOption[T any](ctx context.Context, executor *SQLExecutor, statement SQLStatementSource, params any) (mo.Option[T], error) {
	value, found, err := sqlScalar[T](ctx, executor, statement, params)
	if err != nil {
		return mo.None[T](), err
	}
	if !found {
		return mo.None[T](), nil
	}
	return mo.Some(value), nil
}

func sqlScalar[T any](ctx context.Context, executor *SQLExecutor, statement SQLStatementSource, params any) (T, bool, error) {
	rows, err := queryStatementRows(ctx, executor, statement, params)
	if err != nil {
		var zero T
		return zero, false, err
	}
	defer rows.Close()

	value, err := scanlib.OneFromRows[T](ctx, scanlib.SingleColumnMapper[T], rows)
	if err != nil {
		if err == sql.ErrNoRows {
			var zero T
			return zero, false, nil
		}
		var zero T
		return zero, false, err
	}

	if rows.Next() {
		var zero T
		return zero, false, ErrTooManyRows
	}
	if err := rows.Err(); err != nil {
		var zero T
		return zero, false, err
	}

	return value, true, nil
}

func scanStatementOne[E any](ctx context.Context, executor *SQLExecutor, statement SQLStatementSource, params any, mapper oneRowScanner[E]) (E, bool, error) {
	rows, err := queryStatementRows(ctx, executor, statement, params)
	if err != nil {
		var zero E
		return zero, false, err
	}
	defer rows.Close()

	return mapper.scanOneRows(ctx, rows)
}

func queryStatementRows(ctx context.Context, executor *SQLExecutor, statement SQLStatementSource, params any) (*sql.Rows, error) {
	if executor == nil {
		return nil, ErrNilDB
	}
	return executor.Query(ctx, statement, params)
}
