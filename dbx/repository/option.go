package repository

import (
	"context"

	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/samber/mo"
)

func (r *Base[E, S]) GetByIDOption(ctx context.Context, id any) (mo.Option[E], error) {
	return optionFromResult(r.GetByID(ctx, id))
}

func (r *Base[E, S]) GetByKeyOption(ctx context.Context, key Key) (mo.Option[E], error) {
	return optionFromResult(r.GetByKey(ctx, key))
}

func (r *Base[E, S]) FirstOption(ctx context.Context, query *dbx.SelectQuery) (mo.Option[E], error) {
	return optionFromResult(r.First(ctx, query))
}

func (r *Base[E, S]) FirstSpecOption(ctx context.Context, specs ...Spec) (mo.Option[E], error) {
	return r.FirstOption(ctx, r.applySpecs(specs...))
}
