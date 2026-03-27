package sqlite_test

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	"github.com/DaiYuANg/arcgo/dbx"
	sqlitedialect "github.com/DaiYuANg/arcgo/dbx/dialect/sqlite"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestBuildCreateTable(t *testing.T) {
	bound, err := sqlitedialect.New().BuildCreateTable(dbx.TableSpec{
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

	expected := `CREATE TABLE IF NOT EXISTS "users" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "username" TEXT NOT NULL, "email_address" TEXT NOT NULL, "role_id" INTEGER NOT NULL, "status" INTEGER NOT NULL, CONSTRAINT "fk_users_role_id" FOREIGN KEY ("role_id") REFERENCES "roles" ("id") ON DELETE CASCADE, CONSTRAINT "ck_users_status" CHECK (status >= 0))`
	require.Equal(t, expected, bound.SQL)
}

func TestInspectTable(t *testing.T) {
	ctx := context.Background()
	db := openSQLiteDB(t)

	execSQLiteStatements(t, ctx, db,
		"PRAGMA foreign_keys = ON",
		`CREATE TABLE roles (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT NOT NULL, email_address TEXT NOT NULL, role_id INTEGER NOT NULL, status INTEGER NOT NULL, CONSTRAINT fk_users_role_id FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE, CONSTRAINT ck_users_status CHECK (status >= 0))`,
		`CREATE INDEX idx_users_username ON users(username)`,
		`CREATE UNIQUE INDEX ux_users_email_address ON users(email_address)`,
	)

	dialect := sqlitedialect.New()
	core := dbx.New(db, dialect)
	state, err := dialect.InspectTable(ctx, core, "users")
	require.NoError(t, err)

	require.True(t, state.Exists)
	require.Len(t, state.Columns, 5)
	require.Len(t, state.Indexes, 2)

	require.NotNil(t, state.PrimaryKey)
	require.Equal(t, []string{"id"}, state.PrimaryKey.Columns)

	require.Len(t, state.ForeignKeys, 1)
	require.Equal(t, "roles", state.ForeignKeys[0].TargetTable)

	require.Len(t, state.Checks, 1)
	require.Equal(t, "status >= 0", state.Checks[0].Expression)
}

func openSQLiteDB(tb testing.TB) *sql.DB {
	tb.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(tb, err)

	tb.Cleanup(func() {
		if closeErr := db.Close(); closeErr != nil {
			tb.Errorf("close sqlite db: %v", closeErr)
		}
	})

	return db
}

func execSQLiteStatements(tb testing.TB, ctx context.Context, db *sql.DB, statements ...string) {
	tb.Helper()

	for _, statement := range statements {
		_, err := db.ExecContext(ctx, statement)
		require.NoError(tb, err)
	}
}
