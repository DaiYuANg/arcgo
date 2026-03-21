package dbx

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func BenchmarkNewStructMapperCached(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := NewStructMapper[auditedUser](); err != nil {
			b.Fatalf("NewStructMapper returned error: %v", err)
		}
	}
}

func BenchmarkStructMapperScanPlanCached(b *testing.B) {
	mapper := MustStructMapper[accountRecord]()
	columns := []string{"id", "nickname", "bio", "label"}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := mapper.scanPlan(columns); err != nil {
			b.Fatalf("scanPlan returned error: %v", err)
		}
	}
}

func BenchmarkStructMapperScanPlanAliasFallback(b *testing.B) {
	mapper := MustStructMapper[auditedUser]()
	columns := []string{`"users"."id"`, `"CREATED_BY"`, `"UPDATED_BY"`}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := mapper.scanPlan(columns); err != nil {
			b.Fatalf("scanPlan returned error: %v", err)
		}
	}
}

func BenchmarkMapperInsertAssignments(b *testing.B) {
	accounts := MustSchema("accounts", accountSchema{})
	mapper := MustMapper[accountRecord](accounts)
	entity := &accountRecord{
		Label: "ADMIN",
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := mapper.InsertAssignments(accounts, entity); err != nil {
			b.Fatalf("InsertAssignments returned error: %v", err)
		}
	}
}

func BenchmarkQueryAllStructMapper(b *testing.B) {
	accounts := MustSchema("accounts", accountSchema{})
	mapper := MustStructMapper[accountRecord]()
	query := Select(accounts.AllColumns()...).From(accounts)

	sqlDB, cleanup := OpenTestSQLite(b, mapperScanAccountsDDL,
		`INSERT INTO "accounts" ("id","nickname","bio","label") VALUES (1,'ally','hello','admin'),(2,NULL,NULL,'reader')`,
	)
	defer cleanup()

	core := New(sqlDB, testSQLiteDialect{})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := QueryAll(context.Background(), core, query, mapper); err != nil {
			b.Fatalf("QueryAll returned error: %v", err)
		}
	}
}

func BenchmarkQueryCursorStructMapper(b *testing.B) {
	accounts := MustSchema("accounts", accountSchema{})
	mapper := MustStructMapper[accountRecord]()
	query := Select(accounts.AllColumns()...).From(accounts)

	sqlDB, cleanup := OpenTestSQLite(b, mapperScanAccountsDDL,
		`INSERT INTO "accounts" ("id","nickname","bio","label") VALUES (1,'ally','hello','admin'),(2,NULL,NULL,'reader')`,
	)
	defer cleanup()

	core := New(sqlDB, testSQLiteDialect{})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cursor, err := QueryCursor(context.Background(), core, query, mapper)
		if err != nil {
			b.Fatalf("QueryCursor returned error: %v", err)
		}
		for cursor.Next() {
			if _, err := cursor.Get(); err != nil {
				_ = cursor.Close()
				b.Fatalf("cursor.Get returned error: %v", err)
			}
		}
		if err := cursor.Err(); err != nil {
			_ = cursor.Close()
			b.Fatalf("cursor.Err returned error: %v", err)
		}
		if err := cursor.Close(); err != nil {
			b.Fatalf("cursor.Close returned error: %v", err)
		}
	}
}

func BenchmarkSQLScalar(b *testing.B) {
	statement := NewSQLStatement("user.count", func(_ any) (BoundQuery, error) {
		return BoundQuery{SQL: `SELECT count(*) FROM "users"`}, nil
	})

	sqlDB, cleanup := OpenTestSQLiteWithSchema(b,
		`INSERT INTO "roles" ("id","name") VALUES (1,'r')`,
		`INSERT INTO "users" ("username","email_address","status","role_id") VALUES ('a','a@x.com',1,1),('b','b@x.com',1,1)`,
	)
	defer cleanup()

	db := New(sqlDB, testSQLiteDialect{})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := SQLScalar[int64](context.Background(), db, statement, nil); err != nil {
			b.Fatalf("SQLScalar returned error: %v", err)
		}
	}
}

func BenchmarkQueryAllStructMapperJSONCodec(b *testing.B) {
	registerCSVCodecBenchmark()
	codecAccounts := MustSchema("codec_accounts", codecSchema{})
	mapper := MustStructMapper[codecRecord]()
	query := Select(codecAccounts.AllColumns()...).From(codecAccounts)

	sqlDB, cleanup := OpenTestSQLite(b, mapperCodecExtraDDL,
		`INSERT INTO "codec_accounts" ("id","preferences","tags") VALUES (1,'{"theme":"dark","flags":["alpha","beta"]}','go,dbx,orm')`,
	)
	defer cleanup()

	core := New(sqlDB, testSQLiteDialect{})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := QueryAll(context.Background(), core, query, mapper); err != nil {
			b.Fatalf("QueryAll returned error: %v", err)
		}
	}
}

func BenchmarkMapperInsertAssignmentsCodec(b *testing.B) {
	registerCSVCodecBenchmark()
	accounts := MustSchema("codec_accounts", codecSchema{})
	mapper := MustMapper[codecRecord](accounts)
	entity := &codecRecord{
		Preferences: codecPreferences{Theme: "dark", Flags: []string{"admin", "beta"}},
		Tags:        []string{"alpha", "beta"},
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := mapper.InsertAssignments(accounts, entity); err != nil {
			b.Fatalf("InsertAssignments returned error: %v", err)
		}
	}
}

func registerCSVCodecBenchmark() {
	registerCSVCodecOnce.Do(func() {
		MustRegisterCodec(NewCodec[[]string](
			"csv",
			func(src any) ([]string, error) {
				switch value := src.(type) {
				case string:
					return splitCSV(value), nil
				case []byte:
					return splitCSV(string(value)), nil
				default:
					return nil, errors.New("dbx: csv codec only supports string or []byte")
				}
			},
			func(values []string) (any, error) {
				return strings.Join(values, ","), nil
			},
		))
	})
}
