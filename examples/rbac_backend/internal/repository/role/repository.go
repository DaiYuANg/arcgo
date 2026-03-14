package role

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/DaiYuANg/arcgo/examples/rbac_backend/internal/entity"
	repocore "github.com/DaiYuANg/arcgo/examples/rbac_backend/internal/repository/core"
	"github.com/uptrace/bun"
)

type Repository interface {
	ListRoles(ctx context.Context) ([]entity.RoleModel, error)
	GetRoleByID(ctx context.Context, id int64) (entity.RoleModel, error)
	CreateRole(ctx context.Context, code string, name string) (entity.RoleModel, error)
	UpdateRole(ctx context.Context, id int64, code string, name string) (entity.RoleModel, error)
	DeleteRole(ctx context.Context, id int64) (bool, error)
}

type bunRepository struct {
	base repocore.BaseRepository[entity.RoleModel]
	db   *bun.DB
}

func NewRepository(store *repocore.Store) Repository {
	return &bunRepository{
		base: repocore.NewBaseRepository[entity.RoleModel](store.DB()),
		db:   store.DB(),
	}
}

func (r *bunRepository) ListRoles(ctx context.Context) ([]entity.RoleModel, error) {
	return r.base.List(ctx, "r.id ASC")
}

func (r *bunRepository) GetRoleByID(ctx context.Context, id int64) (entity.RoleModel, error) {
	return r.base.GetByID(ctx, id)
}

func (r *bunRepository) CreateRole(ctx context.Context, code string, name string) (entity.RoleModel, error) {
	row := entity.RoleModel{Code: strings.TrimSpace(code), Name: strings.TrimSpace(name)}
	if err := r.base.Create(ctx, &row); err != nil {
		return entity.RoleModel{}, err
	}
	return r.GetRoleByID(ctx, row.ID)
}

func (r *bunRepository) UpdateRole(ctx context.Context, id int64, code string, name string) (entity.RoleModel, error) {
	updated, err := r.base.UpdateByID(ctx, id, map[string]any{
		"code": strings.TrimSpace(code),
		"name": strings.TrimSpace(name),
	})
	if err != nil {
		return entity.RoleModel{}, err
	}
	if !updated {
		return entity.RoleModel{}, sql.ErrNoRows
	}
	return r.GetRoleByID(ctx, id)
}

func (r *bunRepository) DeleteRole(ctx context.Context, id int64) (bool, error) {
	removed := false
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().Model((*entity.RolePermissionModel)(nil)).Where("role_id = ?", id).Exec(ctx); err != nil {
			return fmt.Errorf("delete role permissions: %w", err)
		}
		if _, err := tx.NewDelete().Model((*entity.UserRoleModel)(nil)).Where("role_id = ?", id).Exec(ctx); err != nil {
			return fmt.Errorf("delete user roles: %w", err)
		}
		res, err := tx.NewDelete().Model((*entity.RoleModel)(nil)).Where("id = ?", id).Exec(ctx)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		removed = affected > 0
		return nil
	})
	if err != nil {
		return false, err
	}
	return removed, nil
}
