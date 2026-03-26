package repository

import (
	"context"
	"errors"

	"github.com/DaiYuANg/arcgo/dbx"
)

func (r *Base[E, S]) List(ctx context.Context, query *dbx.SelectQuery) ([]E, error) {
	if r == nil || r.session == nil {
		return nil, dbx.ErrNilDB
	}
	listQuery := cloneOrDefault(r, query)
	dbx.LogRuntimeNode(r.session, "repository.list.start", "table", r.schema.TableName(), "has_query", query != nil)
	items, err := dbx.QueryAll(ctx, r.session, listQuery, r.mapper)
	if err != nil {
		dbx.LogRuntimeNode(r.session, "repository.list.error", "table", r.schema.TableName(), "error", err)
		return nil, err
	}
	dbx.LogRuntimeNode(r.session, "repository.list.done", "table", r.schema.TableName(), "items", len(items))
	return items, nil
}

func (r *Base[E, S]) ListSpec(ctx context.Context, specs ...Spec) ([]E, error) {
	return r.List(ctx, r.applySpecs(specs...))
}

func (r *Base[E, S]) First(ctx context.Context, query *dbx.SelectQuery) (E, error) {
	var zero E
	if r == nil || r.session == nil {
		return zero, dbx.ErrNilDB
	}
	firstQuery := cloneOrDefault(r, query)
	dbx.LogRuntimeNode(r.session, "repository.first.start", "table", r.schema.TableName(), "has_query", query != nil)
	items, err := dbx.QueryAll(ctx, r.session, firstQuery.Limit(1), r.mapper)
	if err != nil {
		dbx.LogRuntimeNode(r.session, "repository.first.error", "table", r.schema.TableName(), "error", err)
		return zero, err
	}
	if len(items) == 0 {
		dbx.LogRuntimeNode(r.session, "repository.first.not_found", "table", r.schema.TableName())
		return zero, ErrNotFound
	}
	dbx.LogRuntimeNode(r.session, "repository.first.done", "table", r.schema.TableName())
	return items[0], nil
}

func (r *Base[E, S]) FirstSpec(ctx context.Context, specs ...Spec) (E, error) {
	return r.First(ctx, r.applySpecs(specs...))
}

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
	rows, err := dbx.QueryAll(ctx, r.session, countQuery, dbx.MustStructMapper[countRow]())
	if err != nil {
		dbx.LogRuntimeNode(r.session, "repository.count.error", "table", r.schema.TableName(), "error", err)
		return 0, err
	}
	if len(rows) == 0 {
		dbx.LogRuntimeNode(r.session, "repository.count.done", "table", r.schema.TableName(), "count", 0)
		return 0, nil
	}
	dbx.LogRuntimeNode(r.session, "repository.count.done", "table", r.schema.TableName(), "count", rows[0].Count)
	return rows[0].Count, nil
}

func (r *Base[E, S]) CountSpec(ctx context.Context, specs ...Spec) (int64, error) {
	return r.Count(ctx, r.applySpecs(specs...))
}

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

func (r *Base[E, S]) ExistsSpec(ctx context.Context, specs ...Spec) (bool, error) {
	return r.Exists(ctx, r.applySpecs(specs...))
}

func (r *Base[E, S]) ListPage(ctx context.Context, query *dbx.SelectQuery, page int, pageSize int) (PageResult[E], error) {
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
	dbx.LogRuntimeNode(r.session, "repository.list_page.done", "table", r.schema.TableName(), "items", len(items), "total", total)
	return PageResult[E]{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (r *Base[E, S]) ListPageSpec(ctx context.Context, page int, pageSize int, specs ...Spec) (PageResult[E], error) {
	return r.ListPage(ctx, r.applySpecs(specs...), page, pageSize)
}
