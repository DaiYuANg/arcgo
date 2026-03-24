package dbx

import (
	"context"
	"database/sql"
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
		if _, err := mapper.InsertAssignments(New(nil, testSQLiteDialect{}), accounts, entity); err != nil {
			b.Fatalf("InsertAssignments returned error: %v", err)
		}
	}
}

type benchmarkDBAutoIDRecord struct {
	ID    int64  `dbx:"id"`
	Label string `dbx:"label"`
}

type benchmarkSnowflakeIDRecord struct {
	ID    int64  `dbx:"id"`
	Label string `dbx:"label"`
}

type benchmarkUUIDIDRecord struct {
	ID    string `dbx:"id"`
	Label string `dbx:"label"`
}

type benchmarkDBAutoIDSchema struct {
	Schema[benchmarkDBAutoIDRecord]
	ID    Column[benchmarkDBAutoIDRecord, int64]  `dbx:"id,pk"`
	Label Column[benchmarkDBAutoIDRecord, string] `dbx:"label"`
}

type benchmarkSnowflakeIDSchema struct {
	Schema[benchmarkSnowflakeIDRecord]
	ID    IDColumn[benchmarkSnowflakeIDRecord, int64, IDSnowflake] `dbx:"id,pk"`
	Label Column[benchmarkSnowflakeIDRecord, string]               `dbx:"label"`
}

type benchmarkUUIDIDSchema struct {
	Schema[benchmarkUUIDIDRecord]
	ID    IDColumn[benchmarkUUIDIDRecord, string, IDUUIDv7] `dbx:"id,pk"`
	Label Column[benchmarkUUIDIDRecord, string]             `dbx:"label"`
}

func BenchmarkMapperInsertAssignmentsIDStrategy(b *testing.B) {
	dbAutoSchema := MustSchema("benchmark_db_auto_records", benchmarkDBAutoIDSchema{})
	dbAutoMapper := MustMapper[benchmarkDBAutoIDRecord](dbAutoSchema)
	dbAutoEntity := &benchmarkDBAutoIDRecord{Label: "admin"}

	snowflakeSchema := MustSchema("benchmark_snowflake_records", benchmarkSnowflakeIDSchema{})
	snowflakeMapper := MustMapper[benchmarkSnowflakeIDRecord](snowflakeSchema)
	snowflakeEntity := &benchmarkSnowflakeIDRecord{Label: "admin"}

	uuidSchema := MustSchema("benchmark_uuid_records", benchmarkUUIDIDSchema{})
	uuidMapper := MustMapper[benchmarkUUIDIDRecord](uuidSchema)
	uuidEntity := &benchmarkUUIDIDRecord{Label: "admin"}

	b.Run("DBAuto", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dbAutoEntity.ID = 0
			if _, err := dbAutoMapper.InsertAssignments(New(nil, testSQLiteDialect{}), dbAutoSchema, dbAutoEntity); err != nil {
				b.Fatalf("InsertAssignments returned error: %v", err)
			}
		}
	})

	b.Run("Snowflake", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			snowflakeEntity.ID = 0
			if _, err := snowflakeMapper.InsertAssignments(New(nil, testSQLiteDialect{}), snowflakeSchema, snowflakeEntity); err != nil {
				b.Fatalf("InsertAssignments returned error: %v", err)
			}
		}
	})

	b.Run("UUIDv7", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			uuidEntity.ID = ""
			if _, err := uuidMapper.InsertAssignments(New(nil, testSQLiteDialect{}), uuidSchema, uuidEntity); err != nil {
				b.Fatalf("InsertAssignments returned error: %v", err)
			}
		}
	})
}

func BenchmarkQueryAllStructMapper(b *testing.B) {
	accounts := MustSchema("accounts", accountSchema{})
	mapper := MustStructMapper[accountRecord]()
	query := Select(accounts.AllColumns()...).From(accounts)
	ddl := []string{mapperScanAccountsDDL, `INSERT INTO "accounts" ("id","nickname","bio","label") VALUES (1,'ally','hello','admin'),(2,NULL,NULL,'reader')`}

	run := func(b *testing.B, sqlDB *sql.DB) {
		core := New(sqlDB, testSQLiteDialect{})
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := QueryAll(context.Background(), core, query, mapper); err != nil {
				b.Fatalf("QueryAll returned error: %v", err)
			}
		}
	}

	b.Run("Memory", func(b *testing.B) {
		sqlDB, cleanup := OpenBenchmarkSQLiteMemory(b, ddl...)
		defer cleanup()
		run(b, sqlDB)
	})
	b.Run("IO", func(b *testing.B) {
		sqlDB, cleanup := OpenBenchmarkSQLite(b, ddl...)
		defer cleanup()
		run(b, sqlDB)
	})
}

func BenchmarkQueryAllStructMapperWithLimit(b *testing.B) {
	accounts := MustSchema("accounts", accountSchema{})
	mapper := MustStructMapper[accountRecord]()
	query := Select(accounts.AllColumns()...).From(accounts).Limit(20)
	ddl := []string{mapperScanAccountsDDL, `INSERT INTO "accounts" ("id","nickname","bio","label") VALUES (1,'ally','hello','admin'),(2,NULL,NULL,'reader')`}

	run := func(b *testing.B, sqlDB *sql.DB) {
		core := New(sqlDB, testSQLiteDialect{})
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := QueryAll(context.Background(), core, query, mapper); err != nil {
				b.Fatalf("QueryAll returned error: %v", err)
			}
		}
	}

	b.Run("Memory", func(b *testing.B) {
		sqlDB, cleanup := OpenBenchmarkSQLiteMemory(b, ddl...)
		defer cleanup()
		run(b, sqlDB)
	})
	b.Run("IO", func(b *testing.B) {
		sqlDB, cleanup := OpenBenchmarkSQLite(b, ddl...)
		defer cleanup()
		run(b, sqlDB)
	})
}

func BenchmarkQueryCursorStructMapper(b *testing.B) {
	accounts := MustSchema("accounts", accountSchema{})
	mapper := MustStructMapper[accountRecord]()
	query := Select(accounts.AllColumns()...).From(accounts)
	ddl := []string{mapperScanAccountsDDL, `INSERT INTO "accounts" ("id","nickname","bio","label") VALUES (1,'ally','hello','admin'),(2,NULL,NULL,'reader')`}

	run := func(b *testing.B, sqlDB *sql.DB) {
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

	b.Run("Memory", func(b *testing.B) {
		sqlDB, cleanup := OpenBenchmarkSQLiteMemory(b, ddl...)
		defer cleanup()
		run(b, sqlDB)
	})
	b.Run("IO", func(b *testing.B) {
		sqlDB, cleanup := OpenBenchmarkSQLite(b, ddl...)
		defer cleanup()
		run(b, sqlDB)
	})
}

func BenchmarkSQLScalar(b *testing.B) {
	statement := NewSQLStatement("user.count", func(_ any) (BoundQuery, error) {
		return BoundQuery{SQL: `SELECT count(*) FROM "users"`}, nil
	})
	dataSQL := []string{
		`INSERT INTO "roles" ("id","name") VALUES (1,'r')`,
		`INSERT INTO "users" ("username","email_address","status","role_id") VALUES ('a','a@x.com',1,1),('b','b@x.com',1,1)`,
	}

	run := func(b *testing.B, sqlDB *sql.DB) {
		db := New(sqlDB, testSQLiteDialect{})
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := SQLScalar[int64](context.Background(), db, statement, nil); err != nil {
				b.Fatalf("SQLScalar returned error: %v", err)
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

func BenchmarkQueryAllStructMapperJSONCodec(b *testing.B) {
	registerCSVCodecBenchmark()
	codecAccounts := MustSchema("codec_accounts", codecSchema{})
	mapper := MustStructMapper[codecRecord]()
	query := Select(codecAccounts.AllColumns()...).From(codecAccounts)
	ddl := []string{mapperCodecExtraDDL, `INSERT INTO "codec_accounts" ("id","preferences","tags") VALUES (1,'{"theme":"dark","flags":["alpha","beta"]}','go,dbx,orm')`}

	run := func(b *testing.B, sqlDB *sql.DB) {
		core := New(sqlDB, testSQLiteDialect{})
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := QueryAll(context.Background(), core, query, mapper); err != nil {
				b.Fatalf("QueryAll returned error: %v", err)
			}
		}
	}

	b.Run("Memory", func(b *testing.B) {
		sqlDB, cleanup := OpenBenchmarkSQLiteMemory(b, ddl...)
		defer cleanup()
		run(b, sqlDB)
	})
	b.Run("IO", func(b *testing.B) {
		sqlDB, cleanup := OpenBenchmarkSQLite(b, ddl...)
		defer cleanup()
		run(b, sqlDB)
	})
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
		if _, err := mapper.InsertAssignments(New(nil, testSQLiteDialect{}), accounts, entity); err != nil {
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
