package role

import (
	"context"
	"strings"

	"github.com/DaiYuANg/arcgo/examples/rbac_backend/internal/entity"
	modelrole "github.com/DaiYuANg/arcgo/examples/rbac_backend/internal/model/role"
	reporole "github.com/DaiYuANg/arcgo/examples/rbac_backend/internal/repository/role"
	"github.com/samber/lo"
)

type CreateCommand struct {
	Code string
	Name string
}

type UpdateCommand struct {
	Code string
	Name string
}

type Service struct {
	repo reporole.Repository
}

func NewService(repo reporole.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context) ([]modelrole.Item, error) {
	rows, err := s.repo.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	return lo.Map(rows, func(row entity.RoleModel, _ int) modelrole.Item {
		return toRoleItem(row)
	}), nil
}

func (s *Service) Get(ctx context.Context, id int64) (modelrole.Item, error) {
	row, err := s.repo.GetRoleByID(ctx, id)
	if err != nil {
		return modelrole.Item{}, err
	}
	return toRoleItem(row), nil
}

func (s *Service) Create(ctx context.Context, cmd CreateCommand) (modelrole.Item, error) {
	row, err := s.repo.CreateRole(ctx, strings.TrimSpace(cmd.Code), strings.TrimSpace(cmd.Name))
	if err != nil {
		return modelrole.Item{}, err
	}
	return toRoleItem(row), nil
}

func (s *Service) Update(ctx context.Context, id int64, cmd UpdateCommand) (modelrole.Item, error) {
	row, err := s.repo.UpdateRole(ctx, id, strings.TrimSpace(cmd.Code), strings.TrimSpace(cmd.Name))
	if err != nil {
		return modelrole.Item{}, err
	}
	return toRoleItem(row), nil
}

func (s *Service) Delete(ctx context.Context, id int64) (bool, error) {
	return s.repo.DeleteRole(ctx, id)
}

func toRoleItem(row entity.RoleModel) modelrole.Item {
	return modelrole.Item{ID: row.ID, Code: row.Code, Name: row.Name}
}
