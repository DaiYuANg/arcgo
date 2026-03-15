package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/DaiYuANg/archgo/bunx"
	"github.com/DaiYuANg/archgo/examples/rbac_backend/internal/entity"
	repocore "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/repository/core"
	"github.com/samber/lo"
	"github.com/uptrace/bun"
)

type Repository interface {
	ListUsers(ctx context.Context) ([]entity.UserModel, error)
	GetUserByID(ctx context.Context, id int64) (entity.UserModel, error)
	CreateUser(ctx context.Context, username string, password string) (entity.UserModel, error)
	UpdateUser(ctx context.Context, id int64, username string, password string) (entity.UserModel, error)
	DeleteUser(ctx context.Context, id int64) (bool, error)
	ReplaceUserRoles(ctx context.Context, userID int64, roleCodes []string) error
	UserRoles(ctx context.Context, userID int64) ([]string, error)
}

type bunRepository struct {
	base bunx.BaseRepository[entity.UserModel]
	db   *bun.DB
}

func NewRepository(store *repocore.Store) Repository {
	return &bunRepository{
		base: bunx.NewBaseRepository[entity.UserModel](store.DB(), store.Logger()),
		db:   store.DB(),
	}
}

func (r *bunRepository) ListUsers(ctx context.Context) ([]entity.UserModel, error) {
	return r.base.List(ctx, "u.id ASC")
}

func (r *bunRepository) GetUserByID(ctx context.Context, id int64) (entity.UserModel, error) {
	return r.base.GetByID(ctx, id)
}

func (r *bunRepository) CreateUser(ctx context.Context, username string, password string) (entity.UserModel, error) {
	row := entity.UserModel{Username: strings.TrimSpace(username), Password: password}
	if err := r.base.Create(ctx, &row); err != nil {
		return entity.UserModel{}, err
	}
	return r.GetUserByID(ctx, row.ID)
}

func (r *bunRepository) UpdateUser(ctx context.Context, id int64, username string, password string) (entity.UserModel, error) {
	fields := map[string]any{
		"username": strings.TrimSpace(username),
	}
	// Only overwrite the stored hash when a new password is explicitly provided.
	if password != "" {
		fields["password"] = password
	}
	updated, err := r.base.UpdateByID(ctx, id, fields)
	if err != nil {
		return entity.UserModel{}, err
	}
	if !updated {
		return entity.UserModel{}, sql.ErrNoRows
	}
	return r.GetUserByID(ctx, id)
}

func (r *bunRepository) DeleteUser(ctx context.Context, id int64) (bool, error) {
	removed := false
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().Model((*entity.UserRoleModel)(nil)).Where("user_id = ?", id).Exec(ctx); err != nil {
			return fmt.Errorf("delete user roles: %w", err)
		}
		res, err := tx.NewDelete().Model((*entity.UserModel)(nil)).Where("id = ?", id).Exec(ctx)
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

func (r *bunRepository) ReplaceUserRoles(ctx context.Context, userID int64, roleCodes []string) error {
	codes := lo.FilterMap(roleCodes, func(code string, _ int) (string, bool) {
		trimmed := strings.TrimSpace(code)
		if trimmed == "" {
			return "", false
		}
		return trimmed, true
	})
	codes = lo.Uniq(codes)

	roles := make([]entity.RoleModel, 0, len(codes))
	if len(codes) > 0 {
		err := r.db.NewSelect().Model(&roles).Where("code IN (?)", bun.List(codes)).Scan(ctx)
		if err != nil {
			return err
		}
		if len(roles) != len(codes) {
			return errors.New("some role codes do not exist")
		}
	}

	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().Model((*entity.UserRoleModel)(nil)).Where("user_id = ?", userID).Exec(ctx); err != nil {
			return err
		}
		if len(roles) == 0 {
			return nil
		}
		if _, err := tx.NewInsert().Model(new(lo.Map(roles, func(role entity.RoleModel, _ int) entity.UserRoleModel {
			return entity.UserRoleModel{UserID: userID, RoleID: role.ID}
		}))).Exec(ctx); err != nil {
			return err
		}
		return nil
	})
}

func (r *bunRepository) UserRoles(ctx context.Context, userID int64) ([]string, error) {
	rows := make([]entity.RoleModel, 0)
	err := r.db.NewSelect().
		Model(&rows).
		Join("JOIN rbac_user_roles ur ON ur.role_id = r.id").
		Where("ur.user_id = ?", userID).
		OrderExpr("r.id ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return lo.Map(rows, func(item entity.RoleModel, _ int) string { return item.Code }), nil
}
