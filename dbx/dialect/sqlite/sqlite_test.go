package sqlite

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	"github.com/DaiYuANg/arcgo/dbx"
	_ "modernc.org/sqlite"
)

func TestBuildCreateTable(t *testing.T) {
	bound, err := Dialect{}.BuildCreateTable(dbx.TableSpec{
		Name:        "users",
		Columns:     []dbx.ColumnMeta{{Name: "id", Table: "users", GoType: reflect.TypeFor[int64](), PrimaryKey: true, AutoIncrement: true}, {Name: "username", Table: "users", GoType: reflect.TypeFor[string]()}, {Name: "email_address", Table: "users", GoType: reflect.TypeFor[string]()}, {Name: "role_id", Table: "users", GoType: reflect.TypeFor[int64]()}, {Name: "status", Table: "users", GoType: reflect.TypeFor[int]()}},
		PrimaryKey:  &dbx.PrimaryKeyMeta{Name: "pk_users", Table: "users", Columns: []string{"id"}},
		ForeignKeys: []dbx.ForeignKeyMeta{{Name: "fk_users_role_id", Table: "users", Columns: []string{"role_id"}, TargetTable: "roles", TargetColumns: []string{"id"}, OnDelete: dbx.ReferentialCascade}},
		Checks:      []dbx.CheckMeta{{Name: "ck_users_status", Table: "users", Expression: "status >= 0"}},
	})
	if err != nil {
		t.Fatalf("BuildCreateTable returned error: %v", err)
	}
	expected := `CREATE TABLE IF NOT EXISTS "users" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "username" TEXT NOT NULL, "email_address" TEXT NOT NULL, "role_id" INTEGER NOT NULL, "status" INTEGER NOT NULL, CONSTRAINT "fk_users_role_id" FOREIGN KEY ("role_id") REFERENCES "roles" ("id") ON DELETE CASCADE, CONSTRAINT "ck_users_status" CHECK (status >= 0))`
	if bound.SQL != expected {
		t.Fatalf("unexpected create table sql:\nwant: %s\n got: %s", expected, bound.SQL)
	}
}

func TestInspectTable(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("PRAGMA foreign_keys: %v", err)
	}
	ddl := []string{
		`CREATE TABLE roles (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT NOT NULL, email_address TEXT NOT NULL, role_id INTEGER NOT NULL, status INTEGER NOT NULL, CONSTRAINT fk_users_role_id FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE, CONSTRAINT ck_users_status CHECK (status >= 0))`,
		`CREATE INDEX idx_users_username ON users(username)`,
		`CREATE UNIQUE INDEX ux_users_email_address ON users(email_address)`,
	}
	for _, stmt := range ddl {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("exec ddl %q: %v", stmt, err)
		}
	}

	core := dbx.New(db, Dialect{})
	state, err := Dialect{}.InspectTable(context.Background(), core, "users")
	if err != nil {
		t.Fatalf("InspectTable returned error: %v", err)
	}
	if !state.Exists || len(state.Columns) != 5 || len(state.Indexes) != 2 {
		t.Fatalf("unexpected table state: %+v", state)
	}
	if state.PrimaryKey == nil || len(state.PrimaryKey.Columns) != 1 || state.PrimaryKey.Columns[0] != "id" {
		t.Fatalf("unexpected primary key state: %+v", state.PrimaryKey)
	}
	if len(state.ForeignKeys) != 1 || state.ForeignKeys[0].TargetTable != "roles" {
		t.Fatalf("unexpected foreign key state: %+v", state.ForeignKeys)
	}
	if len(state.Checks) != 1 || state.Checks[0].Expression != "status >= 0" {
		t.Fatalf("unexpected check state: %+v", state.Checks)
	}
}
