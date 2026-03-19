package dbx

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/DaiYuANg/arcgo/dbx/internal/testsql"
)

type UserSummary struct {
	ID       int64  `dbx:"id"`
	Username string `dbx:"username"`
}

func TestQueryAllBuildsAndScansWithMapper(t *testing.T) {
	sqlDB, recorder, cleanup, err := testsql.Open(testsql.Plan{
		Queries: []testsql.QueryPlan{
			{
				SQL:  `SELECT "users"."id", "users"."username", "users"."email_address", "users"."status", "users"."role_id" FROM "users" WHERE "users"."status" = ?`,
				Args: []driver.Value{int64(1)},
				Columns: []string{
					"id",
					"username",
					"email_address",
					"status",
					"role_id",
				},
				Rows: [][]driver.Value{
					{int64(1), "alice", "alice@example.com", int64(1), int64(2)},
					{int64(2), "bob", "bob@example.com", int64(1), int64(3)},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("testsql.Open returned error: %v", err)
	}
	defer cleanup()

	users := MustSchema("users", UserSchema{})
	mapper := MustMapper[User](users)
	core := New(sqlDB, testSQLiteDialect{})

	items, err := QueryAll(context.Background(), core, Select(users.AllColumns()...).From(users).Where(users.Status.Eq(1)), mapper)
	if err != nil {
		t.Fatalf("QueryAll returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("unexpected item count: %d", len(items))
	}
	if items[0].Username != "alice" || items[1].RoleID != 3 {
		t.Fatalf("unexpected scanned entities: %+v", items)
	}
	if len(recorder.Queries) != 1 {
		t.Fatalf("unexpected recorded query count: %d", len(recorder.Queries))
	}
}

func TestSelectMappedBuildsProjectionForDTO(t *testing.T) {
	users := MustSchema("users", UserSchema{})
	mapper := MustMapper[UserSummary](users)

	query, err := SelectMapped(users, mapper)
	if err != nil {
		t.Fatalf("SelectMapped returned error: %v", err)
	}

	bound, err := query.Build(testSQLiteDialect{})
	if err != nil {
		t.Fatalf("build returned error: %v", err)
	}
	if bound.SQL != `SELECT "users"."id", "users"."username" FROM "users"` {
		t.Fatalf("unexpected projection sql: %q", bound.SQL)
	}
}

func TestQueryAllScansDTOProjection(t *testing.T) {
	sqlDB, _, cleanup, err := testsql.Open(testsql.Plan{
		Queries: []testsql.QueryPlan{
			{
				SQL:     `SELECT "users"."id", "users"."username" FROM "users"`,
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

	users := MustSchema("users", UserSchema{})
	mapper := MustMapper[UserSummary](users)
	query := MustSelectMapped(users, mapper)

	items, err := QueryAll(context.Background(), New(sqlDB, testSQLiteDialect{}), query, mapper)
	if err != nil {
		t.Fatalf("QueryAll returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("unexpected dto count: %d", len(items))
	}
	if items[0].Username != "alice" || items[1].ID != 2 {
		t.Fatalf("unexpected dto payload: %+v", items)
	}
}

func TestQueryCursorAndEach(t *testing.T) {
	sqlDB, _, cleanup, err := testsql.Open(testsql.Plan{
		Queries: []testsql.QueryPlan{
			{
				SQL:     `SELECT "users"."id", "users"."username" FROM "users"`,
				Columns: []string{"id", "username"},
				Rows: [][]driver.Value{
					{int64(1), "alice"},
					{int64(2), "bob"},
				},
			},
			{
				SQL:     `SELECT "users"."id", "users"."username" FROM "users"`,
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

	users := MustSchema("users", UserSchema{})
	mapper := MustStructMapper[UserSummary]()
	query := Select(users.ID, users.Username).From(users)
	core := New(sqlDB, testSQLiteDialect{})

	cursor, err := QueryCursor(context.Background(), core, query, mapper)
	if err != nil {
		t.Fatalf("QueryCursor returned error: %v", err)
	}
	defer cursor.Close()

	var fromCursor []UserSummary
	for cursor.Next() {
		item, err := cursor.Get()
		if err != nil {
			t.Fatalf("cursor.Get returned error: %v", err)
		}
		fromCursor = append(fromCursor, item)
	}
	if err := cursor.Err(); err != nil {
		t.Fatalf("cursor.Err returned error: %v", err)
	}
	if len(fromCursor) != 2 || fromCursor[0].Username != "alice" || fromCursor[1].ID != 2 {
		t.Fatalf("unexpected cursor items: %+v", fromCursor)
	}

	var fromEach []UserSummary
	QueryEach(context.Background(), core, query, mapper)(func(item UserSummary, err error) bool {
		if err != nil {
			t.Fatalf("QueryEach yielded error: %v", err)
		}
		fromEach = append(fromEach, item)
		return true
	})
	if len(fromEach) != 2 || fromEach[0].Username != "alice" || fromEach[1].ID != 2 {
		t.Fatalf("unexpected each items: %+v", fromEach)
	}
}

func TestMapperBuildsAssignmentsAndPrimaryPredicate(t *testing.T) {
	users := MustSchema("users", UserSchema{})
	mapper := MustMapper[User](users)
	entity := &User{
		ID:       42,
		Username: "alice",
		Email:    "alice@example.com",
		Status:   1,
		RoleID:   9,
	}

	insertAssignments, err := mapper.InsertAssignments(users, entity)
	if err != nil {
		t.Fatalf("InsertAssignments returned error: %v", err)
	}
	if len(insertAssignments) != 4 {
		t.Fatalf("unexpected insert assignment count: %d fields=%+v columns=%+v", len(insertAssignments), mapper.Fields(), users.Columns())
	}
	insertBound, err := InsertInto(users).Values(insertAssignments...).Build(testSQLiteDialect{})
	if err != nil {
		t.Fatalf("insert build returned error: %v", err)
	}
	if insertBound.SQL != `INSERT INTO "users" ("username", "email_address", "status", "role_id") VALUES (?, ?, ?, ?)` {
		t.Fatalf("unexpected insert sql: %q", insertBound.SQL)
	}

	updateAssignments, err := mapper.UpdateAssignments(users, entity)
	if err != nil {
		t.Fatalf("UpdateAssignments returned error: %v", err)
	}
	if len(updateAssignments) != 4 {
		t.Fatalf("unexpected update assignment count: %d", len(updateAssignments))
	}

	predicate, err := mapper.PrimaryPredicate(users, entity)
	if err != nil {
		t.Fatalf("PrimaryPredicate returned error: %v", err)
	}
	updateBound, err := Update(users).Set(updateAssignments...).Where(predicate).Build(testSQLiteDialect{})
	if err != nil {
		t.Fatalf("update build returned error: %v", err)
	}
	if updateBound.SQL != `UPDATE "users" SET "username" = ?, "email_address" = ?, "status" = ?, "role_id" = ? WHERE "users"."id" = ?` {
		t.Fatalf("unexpected update sql: %q", updateBound.SQL)
	}
	if len(updateBound.Args) != 5 || updateBound.Args[4] != int64(42) {
		t.Fatalf("unexpected update args: %#v", updateBound.Args)
	}
}

func TestExecBuildsAndRunsBoundQuery(t *testing.T) {
	sqlDB, recorder, cleanup, err := testsql.Open(testsql.Plan{
		Execs: []testsql.ExecPlan{
			{
				SQL:          `INSERT INTO "users" ("username", "email_address", "status", "role_id") VALUES (?, ?, ?, ?)`,
				Args:         []driver.Value{"alice", "alice@example.com", int64(1), int64(9)},
				RowsAffected: 1,
				LastInsertID: 100,
			},
		},
	})
	if err != nil {
		t.Fatalf("testsql.Open returned error: %v", err)
	}
	defer cleanup()

	users := MustSchema("users", UserSchema{})
	mapper := MustMapper[User](users)
	entity := &User{
		ID:       42,
		Username: "alice",
		Email:    "alice@example.com",
		Status:   1,
		RoleID:   9,
	}

	assignments, err := mapper.InsertAssignments(users, entity)
	if err != nil {
		t.Fatalf("InsertAssignments returned error: %v", err)
	}

	result, err := Exec(context.Background(), New(sqlDB, testSQLiteDialect{}), InsertInto(users).Values(assignments...))
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("RowsAffected returned error: %v", err)
	}
	if rowsAffected != 1 {
		t.Fatalf("unexpected rows affected: %d", rowsAffected)
	}
	if len(recorder.Execs) != 1 {
		t.Fatalf("unexpected recorded exec count: %d", len(recorder.Execs))
	}
}

func TestBeginTxExecsWithinTransaction(t *testing.T) {
	sqlDB, recorder, cleanup, err := testsql.Open(testsql.Plan{
		Execs: []testsql.ExecPlan{
			{
				SQL:          `UPDATE "users" SET "status" = ? WHERE "users"."id" = ?`,
				Args:         []driver.Value{int64(2), int64(42)},
				RowsAffected: 1,
			},
		},
	})
	if err != nil {
		t.Fatalf("testsql.Open returned error: %v", err)
	}
	defer cleanup()

	users := MustSchema("users", UserSchema{})
	core := New(sqlDB, testSQLiteDialect{})
	tx, err := core.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("BeginTx returned error: %v", err)
	}

	result, err := Exec(context.Background(), tx, Update(users).Set(users.Status.Set(2)).Where(users.ID.Eq(42)))
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit returned error: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("RowsAffected returned error: %v", err)
	}
	if rowsAffected != 1 {
		t.Fatalf("unexpected rows affected: %d", rowsAffected)
	}
	if len(recorder.Execs) != 1 {
		t.Fatalf("unexpected recorded exec count: %d", len(recorder.Execs))
	}
}
