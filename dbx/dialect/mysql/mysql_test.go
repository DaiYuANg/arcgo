package mysql_test

import (
	"reflect"
	"testing"

	"github.com/DaiYuANg/arcgo/dbx"
	mysql "github.com/DaiYuANg/arcgo/dbx/dialect/mysql"
	"github.com/stretchr/testify/require"
)

func TestBuildCreateTable(t *testing.T) {
	bound, err := mysql.New().BuildCreateTable(dbx.TableSpec{
		Name: "users",
		Columns: []dbx.ColumnMeta{
			{Name: "id", Table: "users", GoType: reflect.TypeFor[int64](), PrimaryKey: true, AutoIncrement: true},
			{Name: "username", Table: "users", GoType: reflect.TypeFor[string]()},
			{Name: "email_address", Table: "users", GoType: reflect.TypeFor[string]()},
			{Name: "role_id", Table: "users", GoType: reflect.TypeFor[int64]()},
			{Name: "status", Table: "users", GoType: reflect.TypeFor[int]()},
		},
		PrimaryKey: &dbx.PrimaryKeyMeta{
			Name:    "pk_users",
			Table:   "users",
			Columns: []string{"id"},
		},
		ForeignKeys: []dbx.ForeignKeyMeta{
			{
				Name:          "fk_users_role_id",
				Table:         "users",
				Columns:       []string{"role_id"},
				TargetTable:   "roles",
				TargetColumns: []string{"id"},
				OnDelete:      dbx.ReferentialCascade,
			},
		},
		Checks: []dbx.CheckMeta{
			{
				Name:       "ck_users_status",
				Table:      "users",
				Expression: "status >= 0",
			},
		},
	})
	require.NoError(t, err)

	expected := "CREATE TABLE IF NOT EXISTS `users` (`id` BIGINT AUTO_INCREMENT PRIMARY KEY, `username` TEXT NOT NULL, `email_address` TEXT NOT NULL, `role_id` BIGINT NOT NULL, `status` INT NOT NULL, CONSTRAINT `fk_users_role_id` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`) ON DELETE CASCADE, CONSTRAINT `ck_users_status` CHECK (status >= 0))"
	require.Equal(t, expected, bound.SQL)
}

func TestInspectTable(t *testing.T) {
	// InspectTable issues MySQL-specific information_schema queries; it cannot run against SQLite.
	t.Skip("InspectTable requires real mysql")
}
