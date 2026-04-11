package dbx

import (
	"context"
	"database/sql"
	"errors"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/mo"
	"github.com/samber/oops"
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
		return BoundQuery{}, oops.In("dbx").
			With("op", "sql_bind").
			Wrapf(ErrNilStatement, "validate sql statement")
	}

	bound, err := statement.Bind(params)
	if err != nil {
		logRuntimeNode(x.session, "sql.bind.error", "statement", statement.StatementName(), "error", err)
		return BoundQuery{}, wrapDBError("bind sql statement", err)
	}
	if bound.Name == "" {
		bound.Name = statement.StatementName()
	}
	if bound.Args.Len() > 0 {
		bound.Args = bound.Args.Clone()
	}
	logRuntimeNode(x.session, "sql.bind.done", "statement", bound.Name, "args_count", bound.Args.Len())
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
	logRuntimeNode(session, "sql.exec.start", "statement", bound.Name, "args_count", bound.Args.Len())
	result, execErr := session.ExecBoundContext(ctx, bound)
	return result, wrapDBError("execute sql statement", execErr)
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
	logRuntimeNode(session, "sql.query.start", "statement", bound.Name, "args_count", bound.Args.Len())
	rows, queryErr := session.QueryBoundContext(ctx, bound)
	return rows, wrapDBError("query sql statement", queryErr)
}

func (x *SQLExecutor) sessionOrErr() (Session, error) {
	if x == nil || x.session == nil {
		return nil, oops.In("dbx").
			With("op", "sql_session").
			Wrapf(ErrNilDB, "validate sql executor session")
	}
	return x.session, nil
}

func SQLList[E any](ctx context.Context, session Session, statement SQLStatementSource, params any, mapper RowsScanner[E]) (collectionx.List[E], error) {
	if mapper == nil {
		return nil, oops.In("dbx").
			With("op", "sql_list", "statement", statementName(statement)).
			Wrapf(ErrNilMapper, "validate mapper")
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

	items, scanErr := mapper.ScanRows(rows)
	scanErr = errors.Join(wrapDBError("scan statement rows", scanErr), rowsIterError(rows))
	closeErr := closeRows(rows)
	if scanErr != nil {
		logRuntimeNode(session, "sql.list.error", "stage", "scan_rows", "error", scanErr)
		return nil, errors.Join(scanErr, closeErr)
	}
	if closeErr != nil {
		logRuntimeNode(session, "sql.list.error", "stage", "close_rows", "error", closeErr)
		return nil, closeErr
	}
	logRuntimeNode(session, "sql.list.done", "items", items.Len())
	return items, nil
}

// SQLQueryList executes a SQL statement source and returns mapped rows as a collectionx.List.
// This is the collectionx.List companion to SQLList.
func SQLQueryList[E any](ctx context.Context, session Session, statement SQLStatementSource, params any, mapper RowsScanner[E]) (collectionx.List[E], error) {
	return SQLList(ctx, session, statement, params, mapper)
}

func SQLGet[E any](ctx context.Context, session Session, statement SQLStatementSource, params any, mapper RowsScanner[E]) (E, error) {
	if mapper == nil {
		var zero E
		return zero, oops.In("dbx").
			With("op", "sql_get", "statement", statementName(statement)).
			Wrapf(ErrNilMapper, "validate mapper")
	}

	exec, err := sessionExecutor(session)
	if err != nil {
		var zero E
		return zero, err
	}
	if one, ok := mapper.(oneRowScanner[E]); ok {
		value, found, scanErr := scanStatementOne(ctx, exec, statement, params, one)
		if scanErr != nil {
			logRuntimeNode(session, "sql.get.error", "stage", "scan_one", "error", scanErr)
			var zero E
			return zero, scanErr
		}
		if !found {
			logRuntimeNode(session, "sql.get.not_found")
			var zero E
			return zero, oops.In("dbx").
				With("op", "sql_get", "statement", statementName(statement)).
				Wrapf(sql.ErrNoRows, "sql get returned no rows")
		}
		logRuntimeNode(session, "sql.get.done")
		return value, nil
	}

	items, err := SQLList(ctx, session, statement, params, mapper)
	if err != nil {
		var zero E
		return zero, err
	}

	switch items.Len() {
	case 0:
		logRuntimeNode(session, "sql.get.not_found")
		var zero E
		return zero, oops.In("dbx").
			With("op", "sql_get", "statement", statementName(statement)).
			Wrapf(sql.ErrNoRows, "sql get returned no rows")
	case 1:
		logRuntimeNode(session, "sql.get.done")
		item, _ := items.GetFirst()
		return item, nil
	default:
		logRuntimeNode(session, "sql.get.error", "stage", "too_many_rows")
		var zero E
		return zero, oops.In("dbx").
			With("op", "sql_get", "statement", statementName(statement)).
			Wrapf(ErrTooManyRows, "sql get returned too many rows")
	}
}

func SQLFind[E any](ctx context.Context, session Session, statement SQLStatementSource, params any, mapper RowsScanner[E]) (mo.Option[E], error) {
	if mapper == nil {
		return mo.None[E](), oops.In("dbx").
			With("op", "sql_find", "statement", statementName(statement)).
			Wrapf(ErrNilMapper, "validate mapper")
	}

	exec, err := sessionExecutor(session)
	if err != nil {
		return mo.None[E](), err
	}
	if one, ok := mapper.(oneRowScanner[E]); ok {
		value, found, scanErr := scanStatementOne(ctx, exec, statement, params, one)
		if scanErr != nil {
			logRuntimeNode(session, "sql.find.error", "stage", "scan_one", "error", scanErr)
			return mo.None[E](), scanErr
		}
		logRuntimeNode(session, "sql.find.done", "found", found)
		if found {
			return mo.Some(value), nil
		}
		return mo.None[E](), nil
	}

	items, err := SQLList(ctx, session, statement, params, mapper)
	if err != nil {
		return mo.None[E](), err
	}

	switch items.Len() {
	case 0:
		logRuntimeNode(session, "sql.find.done", "found", false)
		return mo.None[E](), nil
	case 1:
		logRuntimeNode(session, "sql.find.done", "found", true)
		item, _ := items.GetFirst()
		return mo.Some(item), nil
	default:
		logRuntimeNode(session, "sql.find.error", "stage", "too_many_rows")
		return mo.None[E](), oops.In("dbx").
			With("op", "sql_find", "statement", statementName(statement)).
			Wrapf(ErrTooManyRows, "sql find returned too many rows")
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
		return zero, oops.In("dbx").
			With("op", "sql_scalar", "statement", statementName(statement)).
			Wrapf(sql.ErrNoRows, "sql scalar returned no rows")
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

	value, err := scanlib.OneFromRows[T](ctx, scanlib.SingleColumnMapper[T], rows)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			closeErr := closeRows(rows)
			var zero T
			return zero, false, closeErr
		}
		closeErr := closeRows(rows)
		var zero T
		return zero, false, errors.Join(wrapDBError("scan scalar row", err), closeErr)
	}

	if rows.Next() {
		closeErr := closeRows(rows)
		var zero T
		return zero, false, errors.Join(
			oops.In("dbx").
				With("op", "sql_scalar", "statement", statementName(statement)).
				Wrapf(ErrTooManyRows, "sql scalar returned too many rows"),
			closeErr,
		)
	}
	if err := rowsIterError(rows); err != nil {
		closeErr := closeRows(rows)
		var zero T
		return zero, false, errors.Join(err, closeErr)
	}
	closeErr := closeRows(rows)
	if closeErr != nil {
		var zero T
		return zero, false, closeErr
	}
	return value, true, nil
}
