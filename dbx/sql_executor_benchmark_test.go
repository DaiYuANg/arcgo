package dbx

import (
	"context"
	"testing"
)

func BenchmarkSQLList(b *testing.B) {
	statement := NewSQLStatement("user.list", func(_ any) (BoundQuery, error) {
		return BoundQuery{SQL: `SELECT "id", "username" FROM "users" WHERE "status" = ?`, Args: []any{int64(1)}}, nil
	})

	sqlDB, cleanup := OpenTestSQLiteWithSchema(b,
		`INSERT INTO "roles" ("id","name") VALUES (1,'r')`,
		`INSERT INTO "users" ("username","email_address","status","role_id") VALUES ('alice','a@x.com',1,1),('bob','b@x.com',1,1)`,
	)
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

	sqlDB, cleanup := OpenTestSQLiteWithSchema(b,
		`INSERT INTO "roles" ("id","name") VALUES (1,'r')`,
		`INSERT INTO "users" ("id","username","email_address","status","role_id") VALUES (1,'alice','a@x.com',1,1)`,
	)
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

	sqlDB, cleanup := OpenTestSQLiteWithSchema(b,
		`INSERT INTO "roles" ("id","name") VALUES (1,'r')`,
		`INSERT INTO "users" ("id","username","email_address","status","role_id") VALUES (1,'alice','a@x.com',1,1)`,
	)
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
