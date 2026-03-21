package dbx

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// Standard DDL for users/roles schema used by most tests.
const testSchemaDDL = `
CREATE TABLE IF NOT EXISTS "roles" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "name" TEXT NOT NULL UNIQUE);
CREATE TABLE IF NOT EXISTS "users" (
	"id" INTEGER PRIMARY KEY AUTOINCREMENT,
	"username" TEXT NOT NULL,
	"email_address" TEXT NOT NULL,
	"status" INTEGER NOT NULL DEFAULT 1,
	"role_id" INTEGER NOT NULL REFERENCES "roles"("id") ON DELETE CASCADE
);
`

// OpenTestSQLite opens an in-memory SQLite DB, runs ddl statements, and returns the DB plus cleanup.
// Call cleanup() when done.
func OpenTestSQLite(tb testing.TB, ddl ...string) (*sql.DB, func()) {
	tb.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		tb.Fatalf("sql.Open: %v", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		tb.Fatalf("PRAGMA foreign_keys: %v", err)
	}
	for _, s := range ddl {
		if s == "" {
			continue
		}
		if _, err := db.Exec(s); err != nil {
			_ = db.Close()
			tb.Fatalf("exec ddl %q: %v", s, err)
		}
	}
	return db, func() { _ = db.Close() }
}

// OpenTestSQLiteWithSchema opens in-memory SQLite with the standard users/roles schema and optional data SQL.
func OpenTestSQLiteWithSchema(tb testing.TB, dataSQL ...string) (*sql.DB, func()) {
	tb.Helper()
	ddl := make([]string, 0, 1+len(dataSQL))
	ddl = append(ddl, testSchemaDDL)
	ddl = append(ddl, dataSQL...)
	return OpenTestSQLite(tb, ddl...)
}
