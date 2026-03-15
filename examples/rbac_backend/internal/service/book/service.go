package book

import (
	"context"

	"github.com/DaiYuANg/archgo/examples/rbac_backend/internal/entity"
	modelbook "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/model/book"
	repobook "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/repository/book"
	"github.com/samber/lo"
)

type Service struct {
	repo repobook.Repository
}

func NewService(repo repobook.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context) ([]modelbook.Item, error) {
	rows, err := s.repo.ListBooks(ctx)
	if err != nil {
		return nil, err
	}
	return lo.Map(rows, func(row entity.BookModel, _ int) modelbook.Item {
		return modelbook.Item{
			ID:        row.ID,
			Title:     row.Title,
			Author:    row.Author,
			CreatedBy: row.CreatedBy,
		}
	}), nil
}

func (s *Service) Create(ctx context.Context, title string, author string, actorID int64) (modelbook.Item, error) {
	row, err := s.repo.CreateBook(ctx, title, author, actorID)
	if err != nil {
		return modelbook.Item{}, err
	}
	return modelbook.Item{
		ID:        row.ID,
		Title:     row.Title,
		Author:    row.Author,
		CreatedBy: row.CreatedBy,
	}, nil
}

func (s *Service) Delete(ctx context.Context, id int64) (bool, error) {
	return s.repo.DeleteBook(ctx, id)
}
