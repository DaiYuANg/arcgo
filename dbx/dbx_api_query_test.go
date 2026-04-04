package dbx_test

import (
	"context"
	"fmt"

	"github.com/DaiYuANg/arcgo/collectionx"
	dbx "github.com/DaiYuANg/arcgo/dbx"
	"github.com/samber/mo"
)

func Alias[S dbx.TableSource](schema S, alias string) S {
	return dbx.Alias(schema, alias)
}

func CaseWhen[T any](predicate Predicate, value any) *CaseBuilder[T] {
	return dbx.CaseWhen[T](predicate, value)
}

func Count[E any, T any](expr Column[E, T]) Aggregate[int64] {
	return dbx.Count(expr)
}

func Like[E any](column Column[E, string], pattern string) Predicate {
	return dbx.Like(column, pattern)
}

func MustMapper[E any](schema SchemaResource) Mapper[E] {
	return dbx.MustMapper[E](schema)
}

func MustSchema[S any](name string, schema S) S {
	return dbx.MustSchema(name, schema)
}

func MustStructMapper[E any]() StructMapper[E] {
	return dbx.MustStructMapper[E]()
}

func NamedColumn[T any](source TableSource, name string) Column[struct{}, T] {
	return dbx.NamedColumn[T](source, name)
}

func NewCodec[T any](name string, decode func(any) (T, error), encode func(T) (any, error)) Codec {
	return dbx.NewCodec(name, decode, encode)
}

func NewStructMapper[E any]() (StructMapper[E], error) {
	mapper, err := dbx.NewStructMapper[E]()
	if err != nil {
		var zero StructMapper[E]
		return zero, fmt.Errorf("new struct mapper: %w", err)
	}
	return mapper, nil
}

func NewStructMapperWithOptions[E any](opts ...MapperOption) (StructMapper[E], error) {
	mapper, err := dbx.NewStructMapperWithOptions[E](opts...)
	if err != nil {
		var zero StructMapper[E]
		return zero, fmt.Errorf("new struct mapper with options: %w", err)
	}
	return mapper, nil
}

func QueryAll[E any](ctx context.Context, session Session, query QueryBuilder, mapper RowsScanner[E]) ([]E, error) {
	items, err := dbx.QueryAll(ctx, session, query, mapper)
	if err != nil {
		return nil, fmt.Errorf("query all: %w", err)
	}
	return items, nil
}

func QueryAllList[E any](ctx context.Context, session Session, query QueryBuilder, mapper RowsScanner[E]) (collectionx.List[E], error) {
	items, err := dbx.QueryAllList(ctx, session, query, mapper)
	if err != nil {
		return nil, fmt.Errorf("query all list: %w", err)
	}
	return items, nil
}

func QueryAllBound[E any](ctx context.Context, session Session, bound BoundQuery, mapper RowsScanner[E]) ([]E, error) {
	items, err := dbx.QueryAllBound(ctx, session, bound, mapper)
	if err != nil {
		return nil, fmt.Errorf("query all bound: %w", err)
	}
	return items, nil
}

func QueryAllBoundList[E any](ctx context.Context, session Session, bound BoundQuery, mapper RowsScanner[E]) (collectionx.List[E], error) {
	items, err := dbx.QueryAllBoundList(ctx, session, bound, mapper)
	if err != nil {
		return nil, fmt.Errorf("query all bound list: %w", err)
	}
	return items, nil
}

func QueryCursor[E any](ctx context.Context, session Session, query QueryBuilder, mapper RowsScanner[E]) (Cursor[E], error) {
	cursor, err := dbx.QueryCursor(ctx, session, query, mapper)
	if err != nil {
		return nil, fmt.Errorf("query cursor: %w", err)
	}
	return cursor, nil
}

func QueryEach[E any](ctx context.Context, session Session, query QueryBuilder, mapper RowsScanner[E]) func(func(E, error) bool) {
	return dbx.QueryEach(ctx, session, query, mapper)
}

func ResultColumn[T any](name string) Column[struct{}, T] {
	return dbx.ResultColumn[T](name)
}

func SQLCursor[E any](ctx context.Context, session Session, statement SQLStatementSource, params any, mapper RowsScanner[E]) (Cursor[E], error) {
	cursor, err := dbx.SQLCursor(ctx, session, statement, params, mapper)
	if err != nil {
		return nil, fmt.Errorf("sql cursor: %w", err)
	}
	return cursor, nil
}

func SQLEach[E any](ctx context.Context, session Session, statement SQLStatementSource, params any, mapper RowsScanner[E]) func(func(E, error) bool) {
	return dbx.SQLEach(ctx, session, statement, params, mapper)
}

func SQLFind[E any](ctx context.Context, session Session, statement SQLStatementSource, params any, mapper RowsScanner[E]) (mo.Option[E], error) {
	item, err := dbx.SQLFind(ctx, session, statement, params, mapper)
	if err != nil {
		return mo.None[E](), fmt.Errorf("sql find: %w", err)
	}
	return item, nil
}

func SQLGet[E any](ctx context.Context, session Session, statement SQLStatementSource, params any, mapper RowsScanner[E]) (E, error) {
	item, err := dbx.SQLGet(ctx, session, statement, params, mapper)
	if err != nil {
		var zero E
		return zero, fmt.Errorf("sql get: %w", err)
	}
	return item, nil
}

func SQLList[E any](ctx context.Context, session Session, statement SQLStatementSource, params any, mapper RowsScanner[E]) ([]E, error) {
	items, err := dbx.SQLList(ctx, session, statement, params, mapper)
	if err != nil {
		return nil, fmt.Errorf("sql list: %w", err)
	}
	return items, nil
}

func SQLQueryList[E any](ctx context.Context, session Session, statement SQLStatementSource, params any, mapper RowsScanner[E]) (collectionx.List[E], error) {
	items, err := dbx.SQLQueryList(ctx, session, statement, params, mapper)
	if err != nil {
		return nil, fmt.Errorf("sql query list: %w", err)
	}
	return items, nil
}

func SQLScalar[T any](ctx context.Context, session Session, statement SQLStatementSource, params any) (T, error) {
	item, err := dbx.SQLScalar[T](ctx, session, statement, params)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("sql scalar: %w", err)
	}
	return item, nil
}

func SQLScalarOption[T any](ctx context.Context, session Session, statement SQLStatementSource, params any) (mo.Option[T], error) {
	item, err := dbx.SQLScalarOption[T](ctx, session, statement, params)
	if err != nil {
		return mo.None[T](), fmt.Errorf("sql scalar option: %w", err)
	}
	return item, nil
}

func StructMapperScanPlanForTest[E any](mapper StructMapper[E], columns []string) error {
	if err := dbx.StructMapperScanPlanForTest(mapper, columns); err != nil {
		return fmt.Errorf("struct mapper scan plan: %w", err)
	}
	return nil
}
