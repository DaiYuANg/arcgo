package dbx_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DaiYuANg/arcgo/collectionx"
)

func TestSQLListScansStructMapperAndPropagatesStatementName(t *testing.T) {
	sqlDB, cleanup := OpenTestSQLiteWithSchema(t,
		`INSERT INTO "roles" ("id","name") VALUES (1,'r1')`,
		`INSERT INTO "users" ("username","email_address","status","role_id") VALUES ('alice','a@x.com',1,1),('bob','b@x.com',1,1)`,
	)
	defer cleanup()

	var event HookEvent
	core := MustNewWithOptions(sqlDB, testSQLiteDialect{}, WithHooks(HookFuncs{
		AfterFunc: func(_ context.Context, actual *HookEvent) {
			if actual != nil && actual.Operation == OperationQuery {
				event = *actual
			}
		},
	}))

	type params struct {
		Status int64
	}

	statement := NewSQLStatement("user.find_active", func(actual any) (BoundQuery, error) {
		value, ok := actual.(params)
		if !ok {
			return BoundQuery{}, errors.New("dbx: sql statement params must be params")
		}
		return BoundQuery{
			SQL:  `SELECT "id", "username" FROM "users" WHERE "status" = ?`,
			Args: collectionx.NewList[any](value.Status),
		}, nil
	})

	items, err := SQLList(context.Background(), core, statement, params{Status: 1}, MustStructMapper[UserSummary]())
	if err != nil {
		t.Fatalf("SQLList returned error: %v", err)
	}
	first, _ := items.Get(0)
	second, _ := items.Get(1)
	if items.Len() != 2 || first.Username != "alice" || second.ID != 2 {
		t.Fatalf("unexpected items: %+v", items.Values())
	}
	if event.Statement != "user.find_active" {
		t.Fatalf("unexpected statement name in hook event: %+v", event)
	}
	if event.SQL != `SELECT "id", "username" FROM "users" WHERE "status" = ?` {
		t.Fatalf("unexpected sql in hook event: %+v", event)
	}
}

func TestSQLQueryListScansStructMapper(t *testing.T) {
	sqlDB, cleanup := OpenTestSQLiteWithSchema(t,
		`INSERT INTO "roles" ("id","name") VALUES (1,'r1')`,
		`INSERT INTO "users" ("username","email_address","status","role_id") VALUES ('alice','a@x.com',1,1),('bob','b@x.com',1,1)`,
	)
	defer cleanup()

	statement := NewSQLStatement("user.find_active", func(_ any) (BoundQuery, error) {
		return BoundQuery{
			SQL:  `SELECT "id", "username" FROM "users" WHERE "status" = ?`,
			Args: collectionx.NewList[any](1),
		}, nil
	})

	items, err := SQLQueryList(context.Background(), New(sqlDB, testSQLiteDialect{}), statement, nil, MustStructMapper[UserSummary]())
	if err != nil {
		t.Fatalf("SQLQueryList returned error: %v", err)
	}
	if items.Len() != 2 {
		t.Fatalf("unexpected list size: %d", items.Len())
	}
	last, _ := items.GetLast()
	if last.ID != 2 || last.Username != "bob" {
		t.Fatalf("unexpected last item: %+v", last)
	}
}

func TestSQLGetAndFind(t *testing.T) {
	statement := NewSQLStatement("user.find_one", func(_ any) (BoundQuery, error) {
		return BoundQuery{SQL: `SELECT "id", "username" FROM "users"`}, nil
	})

	runSQLGetExpectRow(t, statement)
	runSQLGetExpectNoRows(t, statement)
	runSQLFindExpectNone(t, statement)
	runSQLFindExpectRow(t, statement)
	runSQLGetExpectTooManyRows(t, statement)
}

func TestSQLScalarAndScalarOption(t *testing.T) {
	statement := NewSQLStatement("user.count", func(_ any) (BoundQuery, error) {
		return BoundQuery{SQL: `SELECT count(*) FROM "users"`}, nil
	})

	t.Run("scalar returns single value", func(t *testing.T) {
		sqlDB, cleanup := OpenTestSQLiteWithSchema(t,
			`INSERT INTO "roles" ("id","name") VALUES (1,'r1')`,
			`INSERT INTO "users" ("username","email_address","status","role_id") VALUES ('a','a@x.com',1,1),('b','b@x.com',1,1)`,
		)
		defer cleanup()

		value, err := SQLScalar[int64](context.Background(), New(sqlDB, testSQLiteDialect{}), statement, nil)
		if err != nil {
			t.Fatalf("SQLScalar returned error: %v", err)
		}
		if value != 2 {
			t.Fatalf("unexpected scalar value: %d", value)
		}
	})

	t.Run("scalar option returns none", func(t *testing.T) {
		sqlDB, cleanup := OpenTestSQLiteWithSchema(t)
		defer cleanup()

		// Use a query that returns 0 rows when table is empty (count(*) always returns 1 row)
		emptyStmt := NewSQLStatement("user.max_id", func(_ any) (BoundQuery, error) {
			return BoundQuery{SQL: `SELECT "id" FROM "users" LIMIT 1`}, nil
		})
		value, err := SQLScalarOption[int64](context.Background(), New(sqlDB, testSQLiteDialect{}), emptyStmt, nil)
		if err != nil {
			t.Fatalf("SQLScalarOption returned error: %v", err)
		}
		if value.IsPresent() {
			t.Fatalf("expected empty scalar option, got %+v", value)
		}
	})

	t.Run("scalar returns too many rows", func(t *testing.T) {
		sqlDB, cleanup := OpenTestSQLiteWithSchema(t,
			`INSERT INTO "roles" ("id","name") VALUES (1,'r1')`,
			`INSERT INTO "users" ("username","email_address","status","role_id") VALUES ('a','a@x.com',1,1),('b','b@x.com',1,1)`,
		)
		defer cleanup()

		multiRowStmt := NewSQLStatement("user.ids", func(_ any) (BoundQuery, error) {
			return BoundQuery{SQL: `SELECT "id" FROM "users"`}, nil
		})
		_, err := SQLScalar[int64](context.Background(), New(sqlDB, testSQLiteDialect{}), multiRowStmt, nil)
		if !errors.Is(err, ErrTooManyRows) {
			t.Fatalf("expected ErrTooManyRows, got %v", err)
		}
	})
}

func TestSQLCursorAndEach(t *testing.T) {
	statement := NewSQLStatement("user.stream", func(_ any) (BoundQuery, error) {
		return BoundQuery{SQL: `SELECT "id", "username" FROM "users"`}, nil
	})

	sqlDB, cleanup := OpenTestSQLiteWithSchema(t,
		`INSERT INTO "roles" ("id","name") VALUES (1,'r1')`,
		`INSERT INTO "users" ("username","email_address","status","role_id") VALUES ('alice','a@x.com',1,1),('bob','b@x.com',1,1)`,
	)
	defer cleanup()

	db := New(sqlDB, testSQLiteDialect{})
	mapper := MustStructMapper[UserSummary]()

	cursor, err := SQLCursor(context.Background(), db, statement, nil, mapper)
	if err != nil {
		t.Fatalf("SQLCursor returned error: %v", err)
	}
	defer closeCursorOrFatal(t, cursor)

	assertUserSummaryRows(t, collectUserSummaryCursor(t, cursor))
	assertUserSummaryRows(t, collectSQLUserSummaryEach(t, db, statement, mapper))
}

func runSQLGetExpectRow(t *testing.T, statement SQLStatementSource) {
	t.Helper()
	t.Run("get returns single row", func(t *testing.T) {
		sqlDB, cleanup := OpenTestSQLiteWithSchema(t,
			`INSERT INTO "roles" ("id","name") VALUES (1,'r1')`,
			`INSERT INTO "users" ("username","email_address","status","role_id") VALUES ('alice','a@x.com',1,1)`,
		)
		defer cleanup()

		item, err := SQLGet(context.Background(), New(sqlDB, testSQLiteDialect{}), statement, nil, MustStructMapper[UserSummary]())
		if err != nil {
			t.Fatalf("SQLGet returned error: %v", err)
		}
		assertSingleUserSummary(t, item)
	})
}

func runSQLGetExpectNoRows(t *testing.T, statement SQLStatementSource) {
	t.Helper()
	t.Run("get returns sql.ErrNoRows", func(t *testing.T) {
		sqlDB, cleanup := OpenTestSQLiteWithSchema(t)
		defer cleanup()

		_, err := SQLGet(context.Background(), New(sqlDB, testSQLiteDialect{}), statement, nil, MustStructMapper[UserSummary]())
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("expected sql.ErrNoRows, got %v", err)
		}
	})
}

func runSQLFindExpectNone(t *testing.T, statement SQLStatementSource) {
	t.Helper()
	t.Run("find returns none", func(t *testing.T) {
		sqlDB, cleanup := OpenTestSQLiteWithSchema(t)
		defer cleanup()

		result, err := SQLFind(context.Background(), New(sqlDB, testSQLiteDialect{}), statement, nil, MustStructMapper[UserSummary]())
		if err != nil {
			t.Fatalf("SQLFind returned error: %v", err)
		}
		if result.IsPresent() {
			t.Fatalf("expected empty option, got %+v", result)
		}
	})
}

func runSQLFindExpectRow(t *testing.T, statement SQLStatementSource) {
	t.Helper()
	t.Run("find returns row", func(t *testing.T) {
		sqlDB, cleanup := OpenTestSQLiteWithSchema(t,
			`INSERT INTO "roles" ("id","name") VALUES (1,'r1')`,
			`INSERT INTO "users" ("username","email_address","status","role_id") VALUES ('alice','a@x.com',1,1)`,
		)
		defer cleanup()

		result, err := SQLFind(context.Background(), New(sqlDB, testSQLiteDialect{}), statement, nil, MustStructMapper[UserSummary]())
		if err != nil {
			t.Fatalf("SQLFind returned error: %v", err)
		}
		item, ok := result.Get()
		if !ok {
			t.Fatal("expected option value")
		}
		assertSingleUserSummary(t, item)
	})
}

func runSQLGetExpectTooManyRows(t *testing.T, statement SQLStatementSource) {
	t.Helper()
	t.Run("get returns too many rows", func(t *testing.T) {
		sqlDB, cleanup := OpenTestSQLiteWithSchema(t,
			`INSERT INTO "roles" ("id","name") VALUES (1,'r1')`,
			`INSERT INTO "users" ("username","email_address","status","role_id") VALUES ('alice','a@x.com',1,1),('bob','b@x.com',1,1)`,
		)
		defer cleanup()

		_, err := SQLGet(context.Background(), New(sqlDB, testSQLiteDialect{}), statement, nil, MustStructMapper[UserSummary]())
		if !errors.Is(err, ErrTooManyRows) {
			t.Fatalf("expected ErrTooManyRows, got %v", err)
		}
	})
}

func collectSQLUserSummaryEach(t *testing.T, db *DB, statement SQLStatementSource, mapper StructMapper[UserSummary]) []UserSummary {
	t.Helper()
	var items []UserSummary
	SQLEach(context.Background(), db, statement, nil, mapper)(func(item UserSummary, err error) bool {
		if err != nil {
			t.Fatalf("SQLEach yielded error: %v", err)
		}
		items = append(items, item)
		return true
	})
	return items
}

func assertSingleUserSummary(t *testing.T, item UserSummary) {
	t.Helper()
	if item.ID != 1 || item.Username != "alice" {
		t.Fatalf("unexpected item: %+v", item)
	}
}
