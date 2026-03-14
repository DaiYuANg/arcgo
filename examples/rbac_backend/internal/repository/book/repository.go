package book

import (
	"context"
	"time"

	"github.com/DaiYuANg/arcgo/examples/rbac_backend/internal/entity"
	repocore "github.com/DaiYuANg/arcgo/examples/rbac_backend/internal/repository/core"
)

type Repository interface {
	ListBooks(ctx context.Context) ([]entity.BookModel, error)
	CreateBook(ctx context.Context, title string, author string, createdBy int64) (entity.BookModel, error)
	DeleteBook(ctx context.Context, id int64) (bool, error)
}

type bunRepository struct {
	base repocore.BaseRepository[entity.BookModel]
}

func NewRepository(store *repocore.Store) Repository {
	return &bunRepository{base: repocore.NewBaseRepository[entity.BookModel](store.DB())}
}

func (r *bunRepository) ListBooks(ctx context.Context) ([]entity.BookModel, error) {
	return r.base.List(ctx, "b.id ASC")
}

func (r *bunRepository) CreateBook(ctx context.Context, title string, author string, createdBy int64) (entity.BookModel, error) {
	now := time.Now()
	row := entity.BookModel{Title: title, Author: author, CreatedBy: createdBy, CreatedAt: now, UpdatedAt: now}
	if err := r.base.Create(ctx, &row); err != nil {
		return entity.BookModel{}, err
	}
	return row, nil
}

func (r *bunRepository) DeleteBook(ctx context.Context, id int64) (bool, error) {
	return r.base.DeleteByID(ctx, id)
}
