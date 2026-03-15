package auth

import (
	"context"
	"database/sql"
	"errors"

	"github.com/DaiYuANg/archgo/examples/rbac_backend/internal/entity"
	repocore "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/repository/core"
	"github.com/samber/lo"
	"github.com/uptrace/bun"
)

// Repository provides authentication-related data access.
type Repository interface {
	// GetUserByUsername fetches a user record by username only.
	// Password verification (bcrypt) is the caller's responsibility.
	GetUserByUsername(ctx context.Context, username string) (entity.UserModel, error)

	// GetUserRoles returns the role codes assigned to a user.
	GetUserRoles(ctx context.Context, userID int64) ([]string, error)
}

// AuthorizationRepository provides RBAC permission checks.
type AuthorizationRepository interface {
	Can(ctx context.Context, userID int64, action string, resource string) (bool, error)
}

// ---- authentication repository ----

type bunRepository struct {
	db *bun.DB
}

func NewRepository(store *repocore.Store) Repository {
	return &bunRepository{db: store.DB()}
}

func (r *bunRepository) GetUserByUsername(ctx context.Context, username string) (entity.UserModel, error) {
	var user entity.UserModel
	err := r.db.NewSelect().
		Model(&user).
		Where("username = ?", username).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.UserModel{}, sql.ErrNoRows
		}
		return entity.UserModel{}, err
	}
	return user, nil
}

func (r *bunRepository) GetUserRoles(ctx context.Context, userID int64) ([]string, error) {
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
	return lo.Map(rows, func(item entity.RoleModel, _ int) string {
		return item.Code
	}), nil
}

// ---- authorization repository ----

type bunAuthorizationRepository struct {
	db *bun.DB
}

func NewAuthorizationRepository(store *repocore.Store) AuthorizationRepository {
	return &bunAuthorizationRepository{db: store.DB()}
}

func (r *bunAuthorizationRepository) Can(ctx context.Context, userID int64, action string, resource string) (bool, error) {
	count, err := r.db.NewSelect().
		Model((*entity.PermissionModel)(nil)).
		Join("JOIN rbac_role_permissions rp ON rp.permission_id = p.id").
		Join("JOIN rbac_user_roles ur ON ur.role_id = rp.role_id").
		Where("ur.user_id = ?", userID).
		Where("p.action = ?", action).
		Where("p.resource = ?", resource).
		Count(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
