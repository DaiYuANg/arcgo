package user

import (
	"context"
	"strings"

	"github.com/DaiYuANg/arcgo/examples/rbac_backend/internal/entity"
	modeluser "github.com/DaiYuANg/arcgo/examples/rbac_backend/internal/model/user"
	repouser "github.com/DaiYuANg/arcgo/examples/rbac_backend/internal/repository/user"
)

type CreateCommand struct {
	Username  string
	Password  string
	RoleCodes []string
}

type UpdateCommand struct {
	Username  string
	Password  string
	RoleCodes []string
}

type Service struct {
	repo repouser.Repository
}

func NewService(repo repouser.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context) ([]modeluser.Item, error) {
	rows, err := s.repo.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]modeluser.Item, 0, len(rows))
	for _, row := range rows {
		item, buildErr := s.buildUserItem(ctx, row)
		if buildErr != nil {
			return nil, buildErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Service) Get(ctx context.Context, id int64) (modeluser.Item, error) {
	row, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return modeluser.Item{}, err
	}
	return s.buildUserItem(ctx, row)
}

func (s *Service) Create(ctx context.Context, cmd CreateCommand) (modeluser.Item, error) {
	row, err := s.repo.CreateUser(ctx, strings.TrimSpace(cmd.Username), cmd.Password)
	if err != nil {
		return modeluser.Item{}, err
	}
	if err := s.repo.ReplaceUserRoles(ctx, row.ID, cmd.RoleCodes); err != nil {
		return modeluser.Item{}, err
	}
	return s.Get(ctx, row.ID)
}

func (s *Service) Update(ctx context.Context, id int64, cmd UpdateCommand) (modeluser.Item, error) {
	if _, err := s.repo.UpdateUser(ctx, id, strings.TrimSpace(cmd.Username), cmd.Password); err != nil {
		return modeluser.Item{}, err
	}
	if err := s.repo.ReplaceUserRoles(ctx, id, cmd.RoleCodes); err != nil {
		return modeluser.Item{}, err
	}
	return s.Get(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id int64) (bool, error) {
	return s.repo.DeleteUser(ctx, id)
}

func (s *Service) buildUserItem(ctx context.Context, row entity.UserModel) (modeluser.Item, error) {
	roles, err := s.repo.UserRoles(ctx, row.ID)
	if err != nil {
		return modeluser.Item{}, err
	}
	return modeluser.Item{
		ID:        row.ID,
		Username:  row.Username,
		Roles:     roles,
		CreatedAt: row.CreatedAt,
	}, nil
}
