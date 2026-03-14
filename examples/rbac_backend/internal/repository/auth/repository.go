package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/DaiYuANg/arcgo/examples/rbac_backend/internal/entity"
	repocore "github.com/DaiYuANg/arcgo/examples/rbac_backend/internal/repository/core"
	"github.com/samber/lo"
	"github.com/uptrace/bun"
)

type Repository interface {
	Login(ctx context.Context, username string, password string) (entity.Principal, error)
}

type AuthorizationRepository interface {
	Can(ctx context.Context, userID int64, action string, resource string) (bool, error)
}

type bunRepository struct {
	db *bun.DB
}

func NewRepository(store *repocore.Store) Repository {
	return &bunRepository{db: store.DB()}
}

func (r *bunRepository) Login(ctx context.Context, username string, password string) (entity.Principal, error) {
	var user entity.UserModel
	err := r.db.NewSelect().
		Model(&user).
		Where("username = ?", strings.TrimSpace(username)).
		Where("password = ?", password).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.Principal{}, errors.New("invalid username or password")
		}
		return entity.Principal{}, err
	}

	roles, err := r.userRoles(ctx, user.ID)
	if err != nil {
		return entity.Principal{}, err
	}
	return entity.Principal{UserID: user.ID, Username: user.Username, Roles: roles}, nil
}

func (r *bunRepository) userRoles(ctx context.Context, userID int64) ([]string, error) {
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
