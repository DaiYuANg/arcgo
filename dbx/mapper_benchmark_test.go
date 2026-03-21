package dbx

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	"github.com/DaiYuANg/arcgo/dbx/internal/testsql"
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
	sqlText, err := query.Build(testSQLiteDialect{})
	if err != nil {
		b.Fatalf("build returned error: %v", err)
	}

	sqlDB, _, cleanup, err := testsql.Open(testsql.Plan{
		Queries: repeatedQueryPlans(b.N, sqlText.SQL, nil, []string{"id", "nickname", "bio", "label"}, [][]driver.Value{
			{int64(1), "ally", "hello", "admin"},
			{int64(2), nil, nil, "reader"},
		}),
	})
	if err != nil {
		b.Fatalf("testsql.Open returned error: %v", err)
	}
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
	sqlText, err := query.Build(testSQLiteDialect{})
	if err != nil {
		b.Fatalf("build returned error: %v", err)
	}

	sqlDB, _, cleanup, err := testsql.Open(testsql.Plan{
		Queries: repeatedQueryPlans(b.N, sqlText.SQL, nil, []string{"id", "nickname", "bio", "label"}, [][]driver.Value{
			{int64(1), "ally", "hello", "admin"},
			{int64(2), nil, nil, "reader"},
		}),
	})
	if err != nil {
		b.Fatalf("testsql.Open returned error: %v", err)
	}
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

	sqlDB, _, cleanup, err := testsql.Open(testsql.Plan{
		Queries: repeatedQueryPlans(b.N, `SELECT count(*) FROM "users"`, nil, []string{"count"}, [][]driver.Value{
			{int64(2)},
		}),
	})
	if err != nil {
		b.Fatalf("testsql.Open returned error: %v", err)
	}
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
	sqlText, err := query.Build(testSQLiteDialect{})
	if err != nil {
		b.Fatalf("build returned error: %v", err)
	}

	sqlDB, _, cleanup, err := testsql.Open(testsql.Plan{
		Queries: repeatedQueryPlans(b.N, sqlText.SQL, nil, []string{"id", "preferences", "tags"}, [][]driver.Value{
			{int64(1), `{"theme":"dark","flags":["alpha","beta"]}`, "go,dbx,orm"},
		}),
	})
	if err != nil {
		b.Fatalf("testsql.Open returned error: %v", err)
	}
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

func repeatedQueryPlans(count int, sql string, args []driver.Value, columns []string, rows [][]driver.Value) []testsql.QueryPlan {
	plans := make([]testsql.QueryPlan, count)
	for i := 0; i < count; i++ {
		plans[i] = testsql.QueryPlan{
			SQL:     sql,
			Args:    append([]driver.Value(nil), args...),
			Columns: append([]string(nil), columns...),
			Rows:    cloneDriverRows(rows),
		}
	}
	return plans
}

func cloneDriverRows(rows [][]driver.Value) [][]driver.Value {
	items := make([][]driver.Value, len(rows))
	for i, row := range rows {
		items[i] = append([]driver.Value(nil), row...)
	}
	return items
}
