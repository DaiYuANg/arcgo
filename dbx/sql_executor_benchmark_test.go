package dbx

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/DaiYuANg/arcgo/dbx/internal/testsql"
)

func BenchmarkSQLList(b *testing.B) {
	statement := NewSQLStatement("user.list", func(_ any) (BoundQuery, error) {
		return BoundQuery{SQL: `SELECT "id", "username" FROM "users" WHERE "status" = ?`, Args: []any{int64(1)}}, nil
	})

	sqlDB, _, cleanup, err := testsql.Open(testsql.Plan{
		Queries: repeatedQueryPlans(b.N, `SELECT "id", "username" FROM "users" WHERE "status" = ?`, []driver.Value{int64(1)}, []string{"id", "username"}, [][]driver.Value{{int64(1), "alice"}, {int64(2), "bob"}}),
	})
	if err != nil {
		b.Fatalf("testsql.Open returned error: %v", err)
	}
	defer cleanup()

	db := New(sqlDB, testSQLiteDialect{})
	mapper := MustStructMapper[UserSummary]()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := SQLList(context.Background(), db, statement, nil, mapper); err != nil {
			b.Fatalf("SQLList returned error: %v", err)
		}
	}
}

func BenchmarkSQLGet(b *testing.B) {
	statement := NewSQLStatement("user.get", func(_ any) (BoundQuery, error) {
		return BoundQuery{SQL: `SELECT "id", "username" FROM "users" WHERE "id" = ?`, Args: []any{int64(1)}}, nil
	})

	sqlDB, _, cleanup, err := testsql.Open(testsql.Plan{
		Queries: repeatedQueryPlans(b.N, `SELECT "id", "username" FROM "users" WHERE "id" = ?`, []driver.Value{int64(1)}, []string{"id", "username"}, [][]driver.Value{{int64(1), "alice"}}),
	})
	if err != nil {
		b.Fatalf("testsql.Open returned error: %v", err)
	}
	defer cleanup()

	db := New(sqlDB, testSQLiteDialect{})
	mapper := MustStructMapper[UserSummary]()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := SQLGet(context.Background(), db, statement, nil, mapper); err != nil {
			b.Fatalf("SQLGet returned error: %v", err)
		}
	}
}

func BenchmarkSQLFind(b *testing.B) {
	statement := NewSQLStatement("user.find", func(_ any) (BoundQuery, error) {
		return BoundQuery{SQL: `SELECT "id", "username" FROM "users" WHERE "id" = ?`, Args: []any{int64(1)}}, nil
	})

	sqlDB, _, cleanup, err := testsql.Open(testsql.Plan{
		Queries: repeatedQueryPlans(b.N, `SELECT "id", "username" FROM "users" WHERE "id" = ?`, []driver.Value{int64(1)}, []string{"id", "username"}, [][]driver.Value{{int64(1), "alice"}}),
	})
	if err != nil {
		b.Fatalf("testsql.Open returned error: %v", err)
	}
	defer cleanup()

	db := New(sqlDB, testSQLiteDialect{})
	mapper := MustStructMapper[UserSummary]()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := SQLFind(context.Background(), db, statement, nil, mapper)
		if err != nil {
			b.Fatalf("SQLFind returned error: %v", err)
		}
		if result.IsAbsent() {
			b.Fatal("expected result to be present")
		}
	}
}
