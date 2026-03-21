package dbx

import (
	"context"
	"database/sql"

	scanlib "github.com/stephenafamo/scan"
)

type Cursor[T any] interface {
	Close() error
	Next() bool
	Get() (T, error)
	Err() error
}

type rowCursorScanner[E any] interface {
	scanCursor(ctx context.Context, rows *sql.Rows) (Cursor[E], error)
}

type scanCursor[E any] struct {
	cursor scanlib.ICursor[E]
}

func (c scanCursor[E]) Close() error {
	return c.cursor.Close()
}

func (c scanCursor[E]) Next() bool {
	return c.cursor.Next()
}

func (c scanCursor[E]) Get() (E, error) {
	return c.cursor.Get()
}

func (c scanCursor[E]) Err() error {
	return c.cursor.Err()
}

type sliceCursor[E any] struct {
	items []E
	index int
}

func newSliceCursor[E any](items []E) Cursor[E] {
	return &sliceCursor[E]{items: items, index: -1}
}

func (c *sliceCursor[E]) Close() error {
	return nil
}

func (c *sliceCursor[E]) Next() bool {
	if c.index+1 >= len(c.items) {
		return false
	}
	c.index++
	return true
}

func (c *sliceCursor[E]) Get() (E, error) {
	if c.index < 0 || c.index >= len(c.items) {
		var zero E
		return zero, sql.ErrNoRows
	}
	return c.items[c.index], nil
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
		return nil, err
	}

	if cursorMapper, ok := mapper.(rowCursorScanner[E]); ok {
		cursor, err := cursorMapper.scanCursor(ctx, rows)
		if err != nil {
			_ = rows.Close()
			return nil, err
		}
		return cursor, nil
	}

	defer rows.Close()
	items, err := mapper.ScanRows(rows)
	if err != nil {
		return nil, err
	}
	return newSliceCursor(items), nil
}

func QueryEach[E any](ctx context.Context, session Session, query QueryBuilder, mapper RowsScanner[E]) func(func(E, error) bool) {
	return func(yield func(E, error) bool) {
		cursor, err := QueryCursor(ctx, session, query, mapper)
		if err != nil {
			var zero E
			yield(zero, err)
			return
		}
		defer cursor.Close()

		for cursor.Next() {
			item, err := cursor.Get()
			if !yield(item, err) {
				return
			}
			if err != nil {
				return
			}
		}
		if err := cursor.Err(); err != nil {
			var zero E
			yield(zero, err)
		}
	}
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

	if cursorMapper, ok := mapper.(rowCursorScanner[E]); ok {
		cursor, err := cursorMapper.scanCursor(ctx, rows)
		if err != nil {
			_ = rows.Close()
			return nil, err
		}
		return cursor, nil
	}

	defer rows.Close()
	items, err := mapper.ScanRows(rows)
	if err != nil {
		return nil, err
	}
	return newSliceCursor(items), nil
}

// QueryEachBound is the BoundQuery variant of QueryEach. Use with Build for reuse.
func QueryEachBound[E any](ctx context.Context, session Session, bound BoundQuery, mapper RowsScanner[E]) func(func(E, error) bool) {
	return func(yield func(E, error) bool) {
		cursor, err := QueryCursorBound(ctx, session, bound, mapper)
		if err != nil {
			var zero E
			yield(zero, err)
			return
		}
		defer cursor.Close()

		for cursor.Next() {
			item, err := cursor.Get()
			if !yield(item, err) {
				return
			}
			if err != nil {
				return
			}
		}
		if err := cursor.Err(); err != nil {
			var zero E
			yield(zero, err)
		}
	}
}

func SQLEach[E any](ctx context.Context, session Session, statement SQLStatementSource, params any, mapper RowsScanner[E]) func(func(E, error) bool) {
	return func(yield func(E, error) bool) {
		cursor, err := SQLCursor(ctx, session, statement, params, mapper)
		if err != nil {
			var zero E
			yield(zero, err)
			return
		}
		defer cursor.Close()

		for cursor.Next() {
			item, err := cursor.Get()
			if !yield(item, err) {
				return
			}
			if err != nil {
				return
			}
		}
		if err := cursor.Err(); err != nil {
			var zero E
			yield(zero, err)
		}
	}
}
