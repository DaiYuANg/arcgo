package dbx

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/DaiYuANg/arcgo/dbx/internal/testsql"
)

func TestSQLListScansStructMapperAndPropagatesStatementName(t *testing.T) {
	sqlDB, _, cleanup, err := testsql.Open(testsql.Plan{
		Queries: []testsql.QueryPlan{
			{
				SQL:     `SELECT "id", "username" FROM "users" WHERE "status" = ?`,
				Args:    []driver.Value{int64(1)},
				Columns: []string{"id", "username"},
				Rows: [][]driver.Value{
					{int64(1), "alice"},
					{int64(2), "bob"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("testsql.Open returned error: %v", err)
	}
	defer cleanup()

	var event HookEvent
	core := NewWithOptions(sqlDB, testSQLiteDialect{}, WithHooks(HookFuncs{
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
		value := actual.(params)
		return BoundQuery{
			SQL:  `SELECT "id", "username" FROM "users" WHERE "status" = ?`,
			Args: []any{value.Status},
		}, nil
	})

	items, err := SQLList(context.Background(), core.SQL(), statement, params{Status: 1}, MustStructMapper[UserSummary]())
	if err != nil {
		t.Fatalf("SQLList returned error: %v", err)
	}
	if len(items) != 2 || items[0].Username != "alice" || items[1].ID != 2 {
		t.Fatalf("unexpected items: %+v", items)
	}
	if event.Statement != "user.find_active" {
		t.Fatalf("unexpected statement name in hook event: %+v", event)
	}
	if event.SQL != `SELECT "id", "username" FROM "users" WHERE "status" = ?` {
		t.Fatalf("unexpected sql in hook event: %+v", event)
	}
}

func TestSQLGetAndFind(t *testing.T) {
	statement := NewSQLStatement("user.find_one", func(_ any) (BoundQuery, error) {
		return BoundQuery{SQL: `SELECT "id", "username" FROM "users"`}, nil
	})

	t.Run("get returns sql.ErrNoRows", func(t *testing.T) {
		sqlDB, _, cleanup, err := testsql.Open(testsql.Plan{
			Queries: []testsql.QueryPlan{
				{
					SQL:     `SELECT "id", "username" FROM "users"`,
					Columns: []string{"id", "username"},
					Rows:    [][]driver.Value{},
				},
			},
		})
		if err != nil {
			t.Fatalf("testsql.Open returned error: %v", err)
		}
		defer cleanup()

		_, err = SQLGet(context.Background(), New(sqlDB, testSQLiteDialect{}).SQL(), statement, nil, MustStructMapper[UserSummary]())
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("expected sql.ErrNoRows, got %v", err)
		}
	})

	t.Run("find returns none", func(t *testing.T) {
		sqlDB, _, cleanup, err := testsql.Open(testsql.Plan{
			Queries: []testsql.QueryPlan{
				{
					SQL:     `SELECT "id", "username" FROM "users"`,
					Columns: []string{"id", "username"},
					Rows:    [][]driver.Value{},
				},
			},
		})
		if err != nil {
			t.Fatalf("testsql.Open returned error: %v", err)
		}
		defer cleanup()

		result, err := SQLFind(context.Background(), New(sqlDB, testSQLiteDialect{}).SQL(), statement, nil, MustStructMapper[UserSummary]())
		if err != nil {
			t.Fatalf("SQLFind returned error: %v", err)
		}
		if result.IsPresent() {
			t.Fatalf("expected empty option, got %+v", result)
		}
	})

	t.Run("get returns too many rows", func(t *testing.T) {
		sqlDB, _, cleanup, err := testsql.Open(testsql.Plan{
			Queries: []testsql.QueryPlan{
				{
					SQL:     `SELECT "id", "username" FROM "users"`,
					Columns: []string{"id", "username"},
					Rows: [][]driver.Value{
						{int64(1), "alice"},
						{int64(2), "bob"},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("testsql.Open returned error: %v", err)
		}
		defer cleanup()

		_, err = SQLGet(context.Background(), New(sqlDB, testSQLiteDialect{}).SQL(), statement, nil, MustStructMapper[UserSummary]())
		if !errors.Is(err, ErrTooManyRows) {
			t.Fatalf("expected ErrTooManyRows, got %v", err)
		}
	})
}

func TestSQLScalarAndScalarOption(t *testing.T) {
	statement := NewSQLStatement("user.count", func(_ any) (BoundQuery, error) {
		return BoundQuery{SQL: `SELECT count(*) FROM "users"`}, nil
	})

	t.Run("scalar returns single value", func(t *testing.T) {
		sqlDB, _, cleanup, err := testsql.Open(testsql.Plan{
			Queries: []testsql.QueryPlan{
				{
					SQL:     `SELECT count(*) FROM "users"`,
					Columns: []string{"count"},
					Rows: [][]driver.Value{
						{int64(2)},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("testsql.Open returned error: %v", err)
		}
		defer cleanup()

		value, err := SQLScalar[int64](context.Background(), New(sqlDB, testSQLiteDialect{}).SQL(), statement, nil)
		if err != nil {
			t.Fatalf("SQLScalar returned error: %v", err)
		}
		if value != 2 {
			t.Fatalf("unexpected scalar value: %d", value)
		}
	})

	t.Run("scalar option returns none", func(t *testing.T) {
		sqlDB, _, cleanup, err := testsql.Open(testsql.Plan{
			Queries: []testsql.QueryPlan{
				{
					SQL:     `SELECT count(*) FROM "users"`,
					Columns: []string{"count"},
					Rows:    [][]driver.Value{},
				},
			},
		})
		if err != nil {
			t.Fatalf("testsql.Open returned error: %v", err)
		}
		defer cleanup()

		value, err := SQLScalarOption[int64](context.Background(), New(sqlDB, testSQLiteDialect{}).SQL(), statement, nil)
		if err != nil {
			t.Fatalf("SQLScalarOption returned error: %v", err)
		}
		if value.IsPresent() {
			t.Fatalf("expected empty scalar option, got %+v", value)
		}
	})

	t.Run("scalar returns too many rows", func(t *testing.T) {
		sqlDB, _, cleanup, err := testsql.Open(testsql.Plan{
			Queries: []testsql.QueryPlan{
				{
					SQL:     `SELECT count(*) FROM "users"`,
					Columns: []string{"count"},
					Rows: [][]driver.Value{
						{int64(2)},
						{int64(3)},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("testsql.Open returned error: %v", err)
		}
		defer cleanup()

		_, err = SQLScalar[int64](context.Background(), New(sqlDB, testSQLiteDialect{}).SQL(), statement, nil)
		if !errors.Is(err, ErrTooManyRows) {
			t.Fatalf("expected ErrTooManyRows, got %v", err)
		}
	})
}
