package repository

import (
	"context"
	"errors"

	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/samber/mo"
)

func (r *Base[E, S]) GetByIDOption(ctx context.Context, id any) (mo.Option[E], error) {
	item, err := r.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return mo.None[E](), nil
		}
		return mo.None[E](), err
	}
	return mo.Some(item), nil
}

func (r *Base[E, S]) GetByKeyOption(ctx context.Context, key Key) (mo.Option[E], error) {
	item, err := r.GetByKey(ctx, key)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return mo.None[E](), nil
		}
		return mo.None[E](), err
	}
	return mo.Some(item), nil
}

func (r *Base[E, S]) FirstOption(ctx context.Context, query *dbx.SelectQuery) (mo.Option[E], error) {
	item, err := r.First(ctx, query)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return mo.None[E](), nil
		}
		return mo.None[E](), err
	}
	return mo.Some(item), nil
}

func (r *Base[E, S]) FirstSpecOption(ctx context.Context, specs ...Spec) (mo.Option[E], error) {
	return r.FirstOption(ctx, r.applySpecs(specs...))
}

