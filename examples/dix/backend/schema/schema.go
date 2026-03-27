// Package schema defines dbx schemas for the backend example.
package schema

import (
	"time"

	"github.com/DaiYuANg/arcgo/dbx"
)

// UserRow is the dbx row model for the users table.
type UserRow struct {
	ID        int64     `dbx:"id"`
	Name      string    `dbx:"name"`
	Email     string    `dbx:"email"`
	Age       int       `dbx:"age"`
	CreatedAt time.Time `dbx:"created_at,codec=rfc3339_time"`
	UpdatedAt time.Time `dbx:"updated_at,codec=rfc3339_time"`
}

// UserSchema describes the users table for the backend example.
type UserSchema struct {
	dbx.Schema[UserRow]
	ID        dbx.Column[UserRow, int64]     `dbx:"id,pk,auto"`
	Name      dbx.Column[UserRow, string]    `dbx:"name"`
	Email     dbx.Column[UserRow, string]    `dbx:"email,unique"`
	Age       dbx.Column[UserRow, int]       `dbx:"age"`
	CreatedAt dbx.Column[UserRow, time.Time] `dbx:"created_at,codec=rfc3339_time"`
	UpdatedAt dbx.Column[UserRow, time.Time] `dbx:"updated_at,codec=rfc3339_time"`
}
