package dbx

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
)

func benchmarkSQLFetch(
	b *testing.B,
	statement *SQLStatement,
	dataSQL []string,
	runQuery func(context.Context, Session, *SQLStatement, any, StructMapper[UserSummary]) error,
) {
	b.Helper()

	run := func(b *testing.B, sqlDB *sql.DB) {
		b.Helper()
		db := New(sqlDB, testSQLiteDialect{})
		mapper := MustStructMapper[UserSummary]()
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			if err := runQuery(context.Background(), db, statement, nil, mapper); err != nil {
				b.Fatal(err)
			}
		}
	}

	b.Run("Memory", func(b *testing.B) {
		sqlDB, cleanup := OpenBenchmarkSQLiteMemoryWithSchema(b, dataSQL...)
		defer cleanup()
		run(b, sqlDB)
	})
	b.Run("IO", func(b *testing.B) {
		sqlDB, cleanup := OpenBenchmarkSQLiteWithSchema(b, dataSQL...)
		defer cleanup()
		run(b, sqlDB)
	})
}

func BenchmarkSQLList(b *testing.B) {
	statement := NewSQLStatement("user.list", func(_ any) (BoundQuery, error) {
		return BoundQuery{SQL: `SELECT "id", "username" FROM "users" WHERE "status" = ?`, Args: []any{int64(1)}}, nil
	})
	dataSQL := []string{
		`INSERT INTO "roles" ("id","name") VALUES (1,'r')`,
		`INSERT INTO "users" ("username","email_address","status","role_id") VALUES ('alice','a@x.com',1,1),('bob','b@x.com',1,1)`,
	}

	benchmarkSQLFetch(b, statement, dataSQL, func(ctx context.Context, session Session, query *SQLStatement, params any, mapper StructMapper[UserSummary]) error {
		_, err := SQLList[UserSummary](ctx, session, query, params, mapper)
		if err != nil {
			return fmt.Errorf("SQLList returned error: %w", err)
		}
		return nil
	})
}

func BenchmarkSQLGet(b *testing.B) {
	statement := NewSQLStatement("user.get", func(_ any) (BoundQuery, error) {
		return BoundQuery{SQL: `SELECT "id", "username" FROM "users" WHERE "id" = ?`, Args: []any{int64(1)}}, nil
	})
	dataSQL := []string{
		`INSERT INTO "roles" ("id","name") VALUES (1,'r')`,
		`INSERT INTO "users" ("id","username","email_address","status","role_id") VALUES (1,'alice','a@x.com',1,1)`,
	}

	benchmarkSQLFetch(b, statement, dataSQL, func(ctx context.Context, session Session, query *SQLStatement, params any, mapper StructMapper[UserSummary]) error {
		_, err := SQLGet[UserSummary](ctx, session, query, params, mapper)
		if err != nil {
			return fmt.Errorf("SQLGet returned error: %w", err)
		}
		return nil
	})
}

func BenchmarkSQLFind(b *testing.B) {
	statement := NewSQLStatement("user.find", func(_ any) (BoundQuery, error) {
		return BoundQuery{SQL: `SELECT "id", "username" FROM "users" WHERE "id" = ?`, Args: []any{int64(1)}}, nil
	})
	dataSQL := []string{
		`INSERT INTO "roles" ("id","name") VALUES (1,'r')`,
		`INSERT INTO "users" ("id","username","email_address","status","role_id") VALUES (1,'alice','a@x.com',1,1)`,
	}

	run := func(b *testing.B, sqlDB *sql.DB) {
		b.Helper()
		db := New(sqlDB, testSQLiteDialect{})
		mapper := MustStructMapper[UserSummary]()
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			result, err := SQLFind[UserSummary](context.Background(), db, statement, nil, mapper)
			if err != nil {
				b.Fatalf("SQLFind returned error: %v", err)
			}
			if result.IsAbsent() {
				b.Fatal("expected result to be present")
			}
		}
	}

	b.Run("Memory", func(b *testing.B) {
		sqlDB, cleanup := OpenBenchmarkSQLiteMemoryWithSchema(b, dataSQL...)
		defer cleanup()
		run(b, sqlDB)
	})
	b.Run("IO", func(b *testing.B) {
		sqlDB, cleanup := OpenBenchmarkSQLiteWithSchema(b, dataSQL...)
		defer cleanup()
		run(b, sqlDB)
	})
}
