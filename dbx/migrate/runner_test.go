package migrate

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/DaiYuANg/arcgo/dbx/internal/testsql"
)

type testDialect struct{}

func (testDialect) Name() string                                         { return "sqlite" }
func (testDialect) BindVar(_ int) string                                 { return "?" }
func (testDialect) QuoteIdent(ident string) string                       { return `"` + ident + `"` }
func (testDialect) RenderLimitOffset(limit, offset *int) (string, error) { return "", nil }

func TestRunnerUpGoCreatesHistoryAndAppliesMigration(t *testing.T) {
	sqlDB, recorder, cleanup, err := testsql.Open(testsql.Plan{
		Execs: []testsql.ExecPlan{
			{SQL: `CREATE TABLE IF NOT EXISTS "schema_history" ("version" VARCHAR(255) NOT NULL, "description" VARCHAR(255) NOT NULL, "kind" VARCHAR(32) NOT NULL, "checksum" VARCHAR(128) NOT NULL, "success" BOOLEAN NOT NULL, "applied_at" VARCHAR(64) NOT NULL, PRIMARY KEY ("version", "kind", "description"))`},
			{SQL: `CREATE TABLE sample (id INTEGER PRIMARY KEY)`},
			{SQL: `DELETE FROM "schema_history" WHERE "version" = ? AND "kind" = ? AND "description" = ?`},
			{SQL: `INSERT INTO "schema_history" ("version", "description", "kind", "checksum", "success", "applied_at") VALUES (?, ?, ?, ?, ?, ?)`},
		},
		Queries: []testsql.QueryPlan{{
			SQL:     `SELECT "version", "description", "kind", "applied_at", "checksum", "success" FROM "schema_history" ORDER BY "applied_at", "version", "description"`,
			Columns: []string{"version", "description", "kind", "applied_at", "checksum", "success"},
			Rows:    nil,
		}},
	})
	if err != nil {
		t.Fatalf("testsql.Open returned error: %v", err)
	}
	defer cleanup()

	runner := NewRunner(sqlDB, testDialect{}, RunnerOptions{})
	report, err := runner.UpGo(context.Background(), NewGoMigration("1", "create sample", func(ctx context.Context, tx *sql.Tx) error {
		_, execErr := tx.ExecContext(ctx, `CREATE TABLE sample (id INTEGER PRIMARY KEY)`)
		return execErr
	}, nil))
	if err != nil {
		t.Fatalf("UpGo returned error: %v", err)
	}
	if len(report.Applied) != 1 || report.Applied[0].Version != "1" || report.Applied[0].Kind != KindGo {
		t.Fatalf("unexpected go migration report: %+v", report)
	}
	if len(recorder.Execs) != 4 {
		t.Fatalf("unexpected exec count: %d", len(recorder.Execs))
	}
	if len(recorder.Execs[3].Args) != 6 {
		t.Fatalf("unexpected history insert args: %#v", recorder.Execs[3].Args)
	}
	if got := recorder.Execs[3].Args[:5]; !equalDriverValues(got, []driver.Value{"1", "create sample", "go", checksumString("go|1|create sample"), true}) {
		t.Fatalf("unexpected history insert args: %#v", recorder.Execs[3].Args)
	}
}

func TestRunnerPendingSQLTracksRepeatableChecksum(t *testing.T) {
	sqlDB, _, cleanup, err := testsql.Open(testsql.Plan{
		Execs: []testsql.ExecPlan{{SQL: `CREATE TABLE IF NOT EXISTS "schema_history" ("version" VARCHAR(255) NOT NULL, "description" VARCHAR(255) NOT NULL, "kind" VARCHAR(32) NOT NULL, "checksum" VARCHAR(128) NOT NULL, "success" BOOLEAN NOT NULL, "applied_at" VARCHAR(64) NOT NULL, PRIMARY KEY ("version", "kind", "description"))`}},
		Queries: []testsql.QueryPlan{{
			SQL:     `SELECT "version", "description", "kind", "applied_at", "checksum", "success" FROM "schema_history" ORDER BY "applied_at", "version", "description"`,
			Columns: []string{"version", "description", "kind", "applied_at", "checksum", "success"},
			Rows:    [][]driver.Value{{"", "refresh cache", "repeatable", "2026-03-19T22:00:00Z", checksumString(strings.Join([]string{"repeatable", "", "refresh cache", "SELECT 2;\n", ""}, "\n--dbx-migrate--\n")), true}},
		}},
	})
	if err != nil {
		t.Fatalf("testsql.Open returned error: %v", err)
	}
	defer cleanup()

	source := FileSource{
		FS: fstest.MapFS{
			"sql/R__refresh_cache.sql": &fstest.MapFile{Data: []byte("SELECT 1;\n")},
		},
		Dir: "sql",
	}
	runner := NewRunner(sqlDB, testDialect{}, RunnerOptions{ValidateHash: true})
	pending, err := runner.PendingSQL(context.Background(), source)
	if err != nil {
		t.Fatalf("PendingSQL returned error: %v", err)
	}
	if len(pending) != 1 || !pending[0].Repeatable {
		t.Fatalf("unexpected pending repeatable migrations: %+v", pending)
	}
}

func TestRunnerUpSQLAppliesVersionedFiles(t *testing.T) {
	sqlDB, _, cleanup, err := testsql.Open(testsql.Plan{
		Execs: []testsql.ExecPlan{
			{SQL: `CREATE TABLE IF NOT EXISTS "schema_history" ("version" VARCHAR(255) NOT NULL, "description" VARCHAR(255) NOT NULL, "kind" VARCHAR(32) NOT NULL, "checksum" VARCHAR(128) NOT NULL, "success" BOOLEAN NOT NULL, "applied_at" VARCHAR(64) NOT NULL, PRIMARY KEY ("version", "kind", "description"))`},
			{SQL: `CREATE TABLE logs (id INTEGER PRIMARY KEY)`},
			{SQL: `DELETE FROM "schema_history" WHERE "version" = ? AND "kind" = ? AND "description" = ?`},
			{SQL: `INSERT INTO "schema_history" ("version", "description", "kind", "checksum", "success", "applied_at") VALUES (?, ?, ?, ?, ?, ?)`},
		},
		Queries: []testsql.QueryPlan{{
			SQL:     `SELECT "version", "description", "kind", "applied_at", "checksum", "success" FROM "schema_history" ORDER BY "applied_at", "version", "description"`,
			Columns: []string{"version", "description", "kind", "applied_at", "checksum", "success"},
			Rows:    nil,
		}},
	})
	if err != nil {
		t.Fatalf("testsql.Open returned error: %v", err)
	}
	defer cleanup()

	source := FileSource{
		FS: fstest.MapFS{
			"sql/V1__create_logs.sql": &fstest.MapFile{Data: []byte("CREATE TABLE logs (id INTEGER PRIMARY KEY)\n")},
		},
		Dir: "sql",
	}
	runner := NewRunner(sqlDB, testDialect{}, RunnerOptions{})
	report, err := runner.UpSQL(context.Background(), source)
	if err != nil {
		t.Fatalf("UpSQL returned error: %v", err)
	}
	if len(report.Applied) != 1 || report.Applied[0].Kind != KindSQL {
		t.Fatalf("unexpected sql migration report: %+v", report)
	}
}

func equalDriverValues(left, right []driver.Value) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
