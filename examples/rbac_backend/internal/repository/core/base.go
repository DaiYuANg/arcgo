package core

import (
	"context"
	"errors"
	"strings"

	"github.com/uptrace/bun"
)

type BaseRepository[T any] struct {
	db *bun.DB
}

func NewBaseRepository[T any](db *bun.DB) BaseRepository[T] {
	return BaseRepository[T]{db: db}
}

func (r BaseRepository[T]) List(ctx context.Context, orderExpr string) ([]T, error) {
	rows := make([]T, 0)
	q := r.db.NewSelect().Model(&rows)
	if strings.TrimSpace(orderExpr) != "" {
		q = q.OrderExpr(orderExpr)
	}
	if err := q.Scan(ctx); err != nil {
		return nil, err
	}
	return rows, nil
}

func (r BaseRepository[T]) GetByID(ctx context.Context, id int64) (T, error) {
	var row T
	err := r.db.NewSelect().
		Model(&row).
		Where("id = ?", id).
		Limit(1).
		Scan(ctx)
	if err != nil {
		var zero T
		return zero, err
	}
	return row, nil
}

func (r BaseRepository[T]) Create(ctx context.Context, row *T) error {
	if row == nil {
		return errors.New("nil row")
	}
	_, err := r.db.NewInsert().Model(row).Exec(ctx)
	return err
}

func (r BaseRepository[T]) UpdateByID(ctx context.Context, id int64, setters map[string]any) (bool, error) {
	q := r.db.NewUpdate().
		Model((*T)(nil)).
		Where("id = ?", id)

	for column, value := range setters {
		q = q.Set(column+" = ?", value)
	}

	res, err := q.Exec(ctx)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r BaseRepository[T]) DeleteByID(ctx context.Context, id int64) (bool, error) {
	res, err := r.db.NewDelete().
		Model((*T)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}
