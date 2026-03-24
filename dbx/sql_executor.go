package dbx

import (
	"context"
	"database/sql"
	"errors"
	"slices"

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
		logRuntimeNode(x.session, "sql.bind.error", "statement", statement.StatementName(), "error", err)
		return BoundQuery{}, err
	}
	if bound.Name == "" {
		bound.Name = statement.StatementName()
	}
	if len(bound.Args) > 0 {
		bound.Args = slices.Clone(bound.Args)
	}
	logRuntimeNode(x.session, "sql.bind.done", "statement", bound.Name, "args_count", len(bound.Args))
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
	logRuntimeNode(session, "sql.exec.start", "statement", bound.Name, "args_count", len(bound.Args))
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
	logRuntimeNode(session, "sql.query.start", "statement", bound.Name, "args_count", len(bound.Args))
	return session.QueryBoundContext(ctx, bound)
}

func (x *SQLExecutor) sessionOrErr() (Session, error) {
	if x == nil || x.session == nil {
		return nil, ErrNilDB
	}
	return x.session, nil
}

func sessionExecutor(session Session) (*SQLExecutor, error) {
	if session == nil {
		return nil, ErrNilDB
	}
	exec := session.SQL()
	if exec == nil {
		return nil, ErrNilDB
	}
	return exec, nil
}

func SQLList[E any](ctx context.Context, session Session, statement SQLStatementSource, params any, mapper RowsScanner[E]) ([]E, error) {
	if mapper == nil {
		return nil, ErrNilMapper
	}

	exec, err := sessionExecutor(session)
	if err != nil {
		return nil, err
	}
	rows, err := queryStatementRows(ctx, exec, statement, params)
	if err != nil {
		logRuntimeNode(session, "sql.list.error", "stage", "query_rows", "error", err)
		return nil, err
	}
	defer rows.Close()

	items, scanErr := mapper.ScanRows(rows)
	if scanErr != nil {
		logRuntimeNode(session, "sql.list.error", "stage", "scan_rows", "error", scanErr)
		return nil, scanErr
	}
	logRuntimeNode(session, "sql.list.done", "items", len(items))
	return items, nil
}

func SQLGet[E any](ctx context.Context, session Session, statement SQLStatementSource, params any, mapper RowsScanner[E]) (E, error) {
	if mapper == nil {
		var zero E
		return zero, ErrNilMapper
	}

	exec, err := sessionExecutor(session)
	if err != nil {
		var zero E
		return zero, err
	}
	if one, ok := mapper.(oneRowScanner[E]); ok {
		value, found, err := scanStatementOne(ctx, exec, statement, params, one)
		if err != nil {
			logRuntimeNode(session, "sql.get.error", "stage", "scan_one", "error", err)
			var zero E
			return zero, err
		}
		if !found {
			logRuntimeNode(session, "sql.get.not_found")
			var zero E
			return zero, sql.ErrNoRows
		}
		logRuntimeNode(session, "sql.get.done")
		return value, nil
	}

	items, err := SQLList(ctx, session, statement, params, mapper)
	if err != nil {
		var zero E
		return zero, err
	}

	switch len(items) {
	case 0:
		logRuntimeNode(session, "sql.get.not_found")
		var zero E
		return zero, sql.ErrNoRows
	case 1:
		logRuntimeNode(session, "sql.get.done")
		return items[0], nil
	default:
		logRuntimeNode(session, "sql.get.error", "stage", "too_many_rows")
		var zero E
		return zero, ErrTooManyRows
	}
}

func SQLFind[E any](ctx context.Context, session Session, statement SQLStatementSource, params any, mapper RowsScanner[E]) (mo.Option[E], error) {
	if mapper == nil {
		return mo.None[E](), ErrNilMapper
	}

	exec, err := sessionExecutor(session)
	if err != nil {
		return mo.None[E](), err
	}
	if one, ok := mapper.(oneRowScanner[E]); ok {
		value, found, err := scanStatementOne(ctx, exec, statement, params, one)
		if err != nil {
			logRuntimeNode(session, "sql.find.error", "stage", "scan_one", "error", err)
			return mo.None[E](), err
		}
		logRuntimeNode(session, "sql.find.done", "found", found)
		return mo.TupleToOption(value, found), nil
	}

	items, err := SQLList(ctx, session, statement, params, mapper)
	if err != nil {
		return mo.None[E](), err
	}

	switch len(items) {
	case 0:
		logRuntimeNode(session, "sql.find.done", "found", false)
		return mo.None[E](), nil
	case 1:
		logRuntimeNode(session, "sql.find.done", "found", true)
		return mo.Some(items[0]), nil
	default:
		logRuntimeNode(session, "sql.find.error", "stage", "too_many_rows")
		return mo.None[E](), ErrTooManyRows
	}
}

func SQLScalar[T any](ctx context.Context, session Session, statement SQLStatementSource, params any) (T, error) {
	value, found, err := sqlScalar[T](ctx, session, statement, params)
	if err != nil {
		logRuntimeNode(session, "sql.scalar.error", "error", err)
		var zero T
		return zero, err
	}
	if !found {
		logRuntimeNode(session, "sql.scalar.not_found")
		var zero T
		return zero, sql.ErrNoRows
	}
	logRuntimeNode(session, "sql.scalar.done")
	return value, nil
}

func SQLScalarOption[T any](ctx context.Context, session Session, statement SQLStatementSource, params any) (mo.Option[T], error) {
	value, found, err := sqlScalar[T](ctx, session, statement, params)
	if err != nil {
		logRuntimeNode(session, "sql.scalar_option.error", "error", err)
		return mo.None[T](), err
	}
	if !found {
		logRuntimeNode(session, "sql.scalar_option.done", "found", false)
		return mo.None[T](), nil
	}
	logRuntimeNode(session, "sql.scalar_option.done", "found", true)
	return mo.Some(value), nil
}

func sqlScalar[T any](ctx context.Context, session Session, statement SQLStatementSource, params any) (T, bool, error) {
	exec, err := sessionExecutor(session)
	if err != nil {
		var zero T
		return zero, false, err
	}
	rows, err := queryStatementRows(ctx, exec, statement, params)
	if err != nil {
		var zero T
		return zero, false, err
	}
	defer rows.Close()

	value, err := scanlib.OneFromRows[T](ctx, scanlib.SingleColumnMapper[T], rows)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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

func scanStatementOne[E any](ctx context.Context, exec *SQLExecutor, statement SQLStatementSource, params any, mapper oneRowScanner[E]) (E, bool, error) {
	rows, err := queryStatementRows(ctx, exec, statement, params)
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
