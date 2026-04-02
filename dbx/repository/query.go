package repository

import (
	"context"
	"errors"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dbx"
)

// List returns every entity matched by the query.
func (r *Base[E, S]) List(ctx context.Context, query *dbx.SelectQuery) (collectionx.List[E], error) {
	if r == nil || r.session == nil {
		return nil, dbx.ErrNilDB
	}
	listQuery := cloneOrDefault(r, query)
	dbx.LogRuntimeNode(r.session, "repository.list.start", "table", r.schema.TableName(), "has_query", query != nil)
	rows, err := dbx.QueryAll[E](ctx, r.session, listQuery, r.mapper)
	if err != nil {
		dbx.LogRuntimeNode(r.session, "repository.list.error", "table", r.schema.TableName(), "error", err)
		return nil, err
	}
	dbx.LogRuntimeNode(r.session, "repository.list.done", "table", r.schema.TableName(), "items", rows.Len())
	return rows, nil
}

// ListSpec returns every entity matched by the provided specs.
func (r *Base[E, S]) ListSpec(ctx context.Context, specs ...Spec) (collectionx.List[E], error) {
	return r.List(ctx, r.applySpecs(specs...))
}

// First returns the first entity matched by the query.
func (r *Base[E, S]) First(ctx context.Context, query *dbx.SelectQuery) (E, error) {
	var zero E
	if r == nil || r.session == nil {
		return zero, dbx.ErrNilDB
	}
	firstQuery := cloneOrDefault(r, query)
	dbx.LogRuntimeNode(r.session, "repository.first.start", "table", r.schema.TableName(), "has_query", query != nil)
	items, err := dbx.QueryAll[E](ctx, r.session, firstQuery.Limit(1), r.mapper)
	if err != nil {
		dbx.LogRuntimeNode(r.session, "repository.first.error", "table", r.schema.TableName(), "error", err)
		return zero, err
	}
	if items.IsEmpty() {
		dbx.LogRuntimeNode(r.session, "repository.first.not_found", "table", r.schema.TableName())
		return zero, ErrNotFound
	}
	dbx.LogRuntimeNode(r.session, "repository.first.done", "table", r.schema.TableName())
	item, _ := items.GetFirst()
	return item, nil
}

// FirstSpec returns the first entity matched by the provided specs.
func (r *Base[E, S]) FirstSpec(ctx context.Context, specs ...Spec) (E, error) {
	return r.First(ctx, r.applySpecs(specs...))
}

// Count returns the number of rows matched by the query.
func (r *Base[E, S]) Count(ctx context.Context, query *dbx.SelectQuery) (int64, error) {
	if r == nil || r.session == nil {
		return 0, dbx.ErrNilDB
	}
	countQuery := r.defaultSelect()
	if query != nil {
		countQuery = cloneForCount(query)
	}
	dbx.LogRuntimeNode(r.session, "repository.count.start", "table", r.schema.TableName(), "has_query", query != nil)
	countQuery.Items = []dbx.SelectItem{dbx.CountAll().As("count")}
	rows, err := dbx.QueryAll[countRow](ctx, r.session, countQuery, dbx.MustStructMapper[countRow]())
	if err != nil {
		dbx.LogRuntimeNode(r.session, "repository.count.error", "table", r.schema.TableName(), "error", err)
		return 0, err
	}
	if rows.IsEmpty() {
		dbx.LogRuntimeNode(r.session, "repository.count.done", "table", r.schema.TableName(), "count", 0)
		return 0, nil
	}
	row, _ := rows.GetFirst()
	dbx.LogRuntimeNode(r.session, "repository.count.done", "table", r.schema.TableName(), "count", row.Count)
	return row.Count, nil
}

// CountSpec returns the number of rows matched by the provided specs.
func (r *Base[E, S]) CountSpec(ctx context.Context, specs ...Spec) (int64, error) {
	return r.Count(ctx, r.applySpecs(specs...))
}

// Exists reports whether the query matches at least one row.
func (r *Base[E, S]) Exists(ctx context.Context, query *dbx.SelectQuery) (bool, error) {
	_, err := r.First(ctx, query)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	return false, err
}

// ExistsSpec reports whether the provided specs match at least one row.
func (r *Base[E, S]) ExistsSpec(ctx context.Context, specs ...Spec) (bool, error) {
	return r.Exists(ctx, r.applySpecs(specs...))
}

// ListPage returns one page of results together with the total row count.
func (r *Base[E, S]) ListPage(ctx context.Context, query *dbx.SelectQuery, page, pageSize int) (PageResult[E], error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	dbx.LogRuntimeNode(r.session, "repository.list_page.start", "table", r.schema.TableName(), "page", page, "page_size", pageSize)
	total, err := r.Count(ctx, query)
	if err != nil {
		dbx.LogRuntimeNode(r.session, "repository.list_page.error", "table", r.schema.TableName(), "stage", "count", "error", err)
		return PageResult[E]{}, err
	}
	pagedQuery := cloneOrDefault(r, query)
	offset := (page - 1) * pageSize
	items, err := r.List(ctx, pagedQuery.Limit(pageSize).Offset(offset))
	if err != nil {
		dbx.LogRuntimeNode(r.session, "repository.list_page.error", "table", r.schema.TableName(), "stage", "list", "error", err)
		return PageResult[E]{}, err
	}
	dbx.LogRuntimeNode(r.session, "repository.list_page.done", "table", r.schema.TableName(), "items", items.Len(), "total", total)
	return PageResult[E]{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

// ListPageSpec returns one page of results for the provided specs.
func (r *Base[E, S]) ListPageSpec(ctx context.Context, page, pageSize int, specs ...Spec) (PageResult[E], error) {
	return r.ListPage(ctx, r.applySpecs(specs...), page, pageSize)
}
