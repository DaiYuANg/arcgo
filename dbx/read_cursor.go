package dbx

import (
	"context"
	"database/sql"
	"errors"

	"github.com/DaiYuANg/arcgo/collectionx"
	scanlib "github.com/stephenafamo/scan"
)

type Cursor[T any] interface {
	Close() error
	Next() bool
	Get() (T, error)
	Err() error
}

type scanCursor[E any] struct {
	cursor scanlib.ICursor[E]
}

func (c scanCursor[E]) Close() error {
	return wrapDBError("close scan cursor", c.cursor.Close())
}

func (c scanCursor[E]) Next() bool {
	return c.cursor.Next()
}

func (c scanCursor[E]) Get() (E, error) {
	value, err := c.cursor.Get()
	return value, wrapDBError("get scan cursor value", err)
}

func (c scanCursor[E]) Err() error {
	return wrapDBError("read scan cursor error", c.cursor.Err())
}

type sliceCursor[E any] struct {
	items collectionx.List[E]
	index int
}

func newSliceCursor[E any](items collectionx.List[E]) Cursor[E] {
	return &sliceCursor[E]{items: items, index: -1}
}

func (c *sliceCursor[E]) Close() error {
	return nil
}

func (c *sliceCursor[E]) Next() bool {
	if c.index+1 >= c.items.Len() {
		return false
	}
	c.index++
	return true
}

func (c *sliceCursor[E]) Get() (E, error) {
	if c.index < 0 || c.index >= c.items.Len() {
		var zero E
		return zero, sql.ErrNoRows
	}
	item, _ := c.items.Get(c.index)
	return item, nil
}

func (c *sliceCursor[E]) Err() error {
	return nil
}

func QueryCursor[E any](ctx context.Context, session Session, query QueryBuilder, mapper RowsScanner[E]) (Cursor[E], error) {
	if mapper == nil {
		return nil, ErrNilMapper
	}
	bound, err := Build(session, query)
	if err != nil {
		return nil, err
	}
	return QueryCursorBound(ctx, session, bound, mapper)
}

// QueryCursorBound executes a pre-built BoundQuery and returns a cursor. Use with Build
// for reuse when executing the same query multiple times.
func QueryCursorBound[E any](ctx context.Context, session Session, bound BoundQuery, mapper RowsScanner[E]) (Cursor[E], error) {
	if mapper == nil {
		return nil, ErrNilMapper
	}
	if session == nil {
		return nil, ErrNilDB
	}
	rows, err := session.QueryBoundContext(ctx, bound)
	if err != nil {
		logRuntimeNode(session, "query_cursor_bound.query_error", "statement", bound.Name, "error", err)
		return nil, wrapDBError("query cursor rows", err)
	}

	if cursor, ok, err := structMapperCursor(ctx, rows, mapper); ok {
		logRuntimeNode(session, "query_cursor_bound.scan_cursor")
		if err != nil {
			err = errors.Join(wrapDBError("scan cursor rows", err), closeRows(rows))
			logRuntimeNode(session, "query_cursor_bound.scan_cursor_error", "error", err)
			return nil, err
		}
		logRuntimeNode(session, "query_cursor_bound.scan_cursor_done")
		return cursor, nil
	}

	logRuntimeNode(session, "query_cursor_bound.scan_slice")
	items, scanErr := mapper.ScanRows(rows)
	scanErr = errors.Join(wrapDBError("scan cursor rows into slice", scanErr), rowsIterError(rows))
	closeErr := closeRows(rows)
	if scanErr != nil {
		scanErr = errors.Join(scanErr, closeErr)
		logRuntimeNode(session, "query_cursor_bound.scan_slice_error", "error", scanErr)
		return nil, scanErr
	}
	if closeErr != nil {
		logRuntimeNode(session, "query_cursor_bound.scan_slice_error", "error", closeErr)
		return nil, closeErr
	}
	logRuntimeNode(session, "query_cursor_bound.scan_slice_done", "items", items.Len())
	return newSliceCursor(items), nil
}

func QueryEach[E any](ctx context.Context, session Session, query QueryBuilder, mapper RowsScanner[E]) func(func(E, error) bool) {
	return iterateCursor(func() (Cursor[E], error) {
		return QueryCursor(ctx, session, query, mapper)
	})
}

func SQLCursor[E any](ctx context.Context, session Session, statement SQLStatementSource, params any, mapper RowsScanner[E]) (Cursor[E], error) {
	if mapper == nil {
		return nil, ErrNilMapper
	}

	exec, err := sessionExecutor(session)
	if err != nil {
		return nil, err
	}
	rows, err := queryStatementRows(ctx, exec, statement, params)
	if err != nil {
		return nil, err
	}

	if cursor, ok, err := structMapperCursor(ctx, rows, mapper); ok {
		if err != nil {
			return nil, errors.Join(wrapDBError("scan sql cursor rows", err), closeRows(rows))
		}
		return cursor, nil
	}

	items, scanErr := mapper.ScanRows(rows)
	scanErr = errors.Join(wrapDBError("scan sql rows", scanErr), rowsIterError(rows))
	closeErr := closeRows(rows)
	if scanErr != nil {
		return nil, errors.Join(scanErr, closeErr)
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return newSliceCursor(items), nil
}

// QueryEachBound is the BoundQuery variant of QueryEach. Use with Build for reuse.
func QueryEachBound[E any](ctx context.Context, session Session, bound BoundQuery, mapper RowsScanner[E]) func(func(E, error) bool) {
	return iterateCursor(func() (Cursor[E], error) {
		return QueryCursorBound(ctx, session, bound, mapper)
	})
}

func SQLEach[E any](ctx context.Context, session Session, statement SQLStatementSource, params any, mapper RowsScanner[E]) func(func(E, error) bool) {
	return iterateCursor(func() (Cursor[E], error) {
		return SQLCursor(ctx, session, statement, params, mapper)
	})
}

func structMapperCursor[E any](ctx context.Context, rows *sql.Rows, mapper RowsScanner[E]) (Cursor[E], bool, error) {
	switch typed := any(mapper).(type) {
	case StructMapper[E]:
		cursor, err := typed.scanCursor(ctx, rows)
		return cursor, true, err
	case *StructMapper[E]:
		if typed == nil {
			return nil, true, ErrNilMapper
		}
		cursor, err := typed.scanCursor(ctx, rows)
		return cursor, true, err
	default:
		return nil, false, nil
	}
}

func iterateCursor[E any](open func() (Cursor[E], error)) func(func(E, error) bool) {
	return func(yield func(E, error) bool) {
		cursor, ok := openCursorOrYieldError(open, yield)
		if !ok {
			return
		}
		defer yieldCursorCloseError(cursor, yield)
		if !drainCursor(cursor, yield) {
			return
		}
		yieldCursorErr(cursor, yield)
	}
}

func openCursorOrYieldError[E any](open func() (Cursor[E], error), yield func(E, error) bool) (Cursor[E], bool) {
	cursor, err := open()
	if err == nil {
		return cursor, true
	}
	var zero E
	yield(zero, err)
	return nil, false
}

func drainCursor[E any](cursor Cursor[E], yield func(E, error) bool) bool {
	for cursor.Next() {
		if !yieldCursorItem(cursor, yield) {
			return false
		}
	}
	return true
}

func yieldCursorItem[E any](cursor Cursor[E], yield func(E, error) bool) bool {
	item, itemErr := cursor.Get()
	if !yield(item, itemErr) {
		return false
	}
	return itemErr == nil
}

func yieldCursorErr[E any](cursor Cursor[E], yield func(E, error) bool) {
	if err := cursor.Err(); err != nil {
		var zero E
		yield(zero, err)
	}
}

func yieldCursorCloseError[E any](cursor Cursor[E], yield func(E, error) bool) {
	if closeErr := closeCursor(cursor); closeErr != nil {
		var zero E
		yield(zero, closeErr)
	}
}
