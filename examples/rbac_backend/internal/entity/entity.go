package entity

import (
	"time"

	"github.com/uptrace/bun"
)

type UserModel struct {
	bun.BaseModel `bun:"table:rbac_users,alias:u"`

	ID        int64     `bun:",pk,autoincrement"`
	Username  string    `bun:",notnull,unique"`
	Password  string    `bun:",notnull"`
	CreatedAt time.Time `bun:",notnull,default:current_timestamp"`
}

type RoleModel struct {
	bun.BaseModel `bun:"table:rbac_roles,alias:r"`

	ID   int64  `bun:",pk,autoincrement"`
	Code string `bun:",notnull,unique"`
	Name string `bun:",notnull"`
}

type PermissionModel struct {
	bun.BaseModel `bun:"table:rbac_permissions,alias:p"`

	ID       int64  `bun:",pk,autoincrement"`
	Action   string `bun:",notnull"`
	Resource string `bun:",notnull"`
}

type UserRoleModel struct {
	bun.BaseModel `bun:"table:rbac_user_roles,alias:ur"`

	UserID int64 `bun:",pk"`
	RoleID int64 `bun:",pk"`
}

type RolePermissionModel struct {
	bun.BaseModel `bun:"table:rbac_role_permissions,alias:rp"`

	RoleID       int64 `bun:",pk"`
	PermissionID int64 `bun:",pk"`
}

type BookModel struct {
	bun.BaseModel `bun:"table:rbac_books,alias:b"`

	ID        int64     `bun:",pk,autoincrement"`
	Title     string    `bun:",notnull"`
	Author    string    `bun:",notnull"`
	CreatedBy int64     `bun:",notnull"`
	CreatedAt time.Time `bun:",notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:",notnull,default:current_timestamp"`
}

type Principal struct {
	UserID   int64
	Username string
	Roles    []string
}
