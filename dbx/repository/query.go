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
	if query == nil {
		query = r.defaultSelect()
	}
	return dbx.QueryAll(ctx, r.session, query, r.mapper)
}

func (r *Base[E, S]) ListSpec(ctx context.Context, specs ...Spec) ([]E, error) {
	return r.List(ctx, r.applySpecs(specs...))
}

func (r *Base[E, S]) First(ctx context.Context, query *dbx.SelectQuery) (E, error) {
	var zero E
	if r == nil || r.session == nil {
		return zero, dbx.ErrNilDB
	}
	firstQuery := query
	if firstQuery == nil {
		firstQuery = r.defaultSelect()
	} else {
		firstQuery = firstQuery.Clone()
	}
	items, err := dbx.QueryAll(ctx, r.session, firstQuery.Limit(1), r.mapper)
	if err != nil {
		return zero, err
	}
	if len(items) == 0 {
		return zero, ErrNotFound
	}
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
	countQuery.Items = []dbx.SelectItem{dbx.CountAll().As("count")}
	rows, err := dbx.QueryAll(ctx, r.session, countQuery, dbx.MustStructMapper[countRow]())
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
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
	total, err := r.Count(ctx, query)
	if err != nil {
		return PageResult[E]{}, err
	}
	pagedQuery := query
	if pagedQuery == nil {
		pagedQuery = r.defaultSelect()
	} else {
		pagedQuery = pagedQuery.Clone()
	}
	offset := (page - 1) * pageSize
	items, err := r.List(ctx, pagedQuery.Limit(pageSize).Offset(offset))
	if err != nil {
		return PageResult[E]{}, err
	}
	return PageResult[E]{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (r *Base[E, S]) ListPageSpec(ctx context.Context, page int, pageSize int, specs ...Spec) (PageResult[E], error) {
	return r.ListPage(ctx, r.applySpecs(specs...), page, pageSize)
}
