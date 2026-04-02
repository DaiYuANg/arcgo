package dbx_test

import (
	"context"
	"errors"
	"sync"
	"testing"
)

type UserSummary struct {
	ID       int64  `dbx:"id"`
	Username string `dbx:"username"`
}

type SnowflakeUser struct {
	ID       int64  `dbx:"id"`
	Username string `dbx:"username"`
}

type SnowflakeUserSchema struct {
	Schema[SnowflakeUser]
	ID       IDColumn[SnowflakeUser, int64, IDSnowflake] `dbx:"id,pk"`
	Username Column[SnowflakeUser, string]               `dbx:"username"`
}

type UUIDUser struct {
	ID       string `dbx:"id"`
	Username string `dbx:"username"`
}

type UUIDUserSchema struct {
	Schema[UUIDUser]
	ID       IDColumn[UUIDUser, string, IDUUIDv7] `dbx:"id,pk"`
	Username Column[UUIDUser, string]             `dbx:"username"`
}

type ULIDUser struct {
	ID       string `dbx:"id"`
	Username string `dbx:"username"`
}

type ULIDUserSchema struct {
	Schema[ULIDUser]
	ID       IDColumn[ULIDUser, string, IDULID] `dbx:"id,pk"`
	Username Column[ULIDUser, string]           `dbx:"username"`
}

type KSUIDUser struct {
	ID       string `dbx:"id"`
	Username string `dbx:"username"`
}

type KSUIDUserSchema struct {
	Schema[KSUIDUser]
	ID       IDColumn[KSUIDUser, string, IDKSUID] `dbx:"id,pk"`
	Username Column[KSUIDUser, string]            `dbx:"username"`
}

type hookRecorder struct {
	mu         sync.Mutex
	queryCount int
	execCount  int
}

func (r *hookRecorder) after(_ context.Context, event *HookEvent) {
	if event == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	switch event.Operation {
	case OperationQuery:
		r.queryCount++
	case OperationExec:
		r.execCount++
	case OperationQueryRow, OperationBeginTx, OperationCommitTx, OperationRollbackTx, OperationAutoMigrate, OperationValidate:
	}
}

func TestQueryAllBuildsAndScansWithMapper(t *testing.T) {
	sqlDB, cleanup := OpenTestSQLiteWithSchema(t,
		`INSERT INTO "roles" ("id","name") VALUES (2,'r2'),(3,'r3')`,
		`INSERT INTO "users" ("username","email_address","status","role_id") VALUES ('alice','alice@example.com',1,2),('bob','bob@example.com',1,3)`,
	)
	defer cleanup()

	users := MustSchema("users", UserSchema{})
	mapper := MustMapper[User](users)
	rec := &hookRecorder{}
	core := MustNewWithOptions(sqlDB, testSQLiteDialect{}, WithHooks(HookFuncs{AfterFunc: rec.after}))

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
	if rec.queryCount != 1 {
		t.Fatalf("unexpected recorded query count: %d", rec.queryCount)
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
	sqlDB, cleanup := OpenTestSQLiteWithSchema(t,
		`INSERT INTO "roles" ("id","name") VALUES (1,'r1')`,
		`INSERT INTO "users" ("username","email_address","status","role_id") VALUES ('alice','a@x.com',1,1),('bob','b@x.com',1,1)`,
	)
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
	sqlDB, cleanup := OpenTestSQLiteWithSchema(t,
		`INSERT INTO "roles" ("id","name") VALUES (1,'r1')`,
		`INSERT INTO "users" ("username","email_address","status","role_id") VALUES ('alice','a@x.com',1,1),('bob','b@x.com',1,1)`,
	)
	defer cleanup()

	users := MustSchema("users", UserSchema{})
	mapper := MustStructMapper[UserSummary]()
	query := Select(users.ID, users.Username).From(users)
	core := New(sqlDB, testSQLiteDialect{})

	cursor, err := QueryCursor(context.Background(), core, query, mapper)
	if err != nil {
		t.Fatalf("QueryCursor returned error: %v", err)
	}
	defer closeCursorOrFatal(t, cursor)

	assertUserSummaryRows(t, collectUserSummaryCursor(t, cursor))
	assertUserSummaryRows(t, collectUserSummaryEach(t, core, query, mapper))
}

func TestBuildRejectsNilQuery(t *testing.T) {
	_, err := Build(New(nil, testSQLiteDialect{}), nil)
	if !errors.Is(err, ErrNilQuery) {
		t.Fatalf("expected ErrNilQuery, got: %v", err)
	}
}

func TestExecRejectsNilQuery(t *testing.T) {
	_, err := Exec(context.Background(), New(nil, testSQLiteDialect{}), nil)
	if !errors.Is(err, ErrNilQuery) {
		t.Fatalf("expected ErrNilQuery, got: %v", err)
	}
}

func TestQueryAllRejectsNilQuery(t *testing.T) {
	_, err := QueryAll(context.Background(), New(nil, testSQLiteDialect{}), nil, MustStructMapper[UserSummary]())
	if !errors.Is(err, ErrNilQuery) {
		t.Fatalf("expected ErrNilQuery, got: %v", err)
	}
}

func TestQueryCursorRejectsNilQuery(t *testing.T) {
	_, err := QueryCursor(context.Background(), New(nil, testSQLiteDialect{}), nil, MustStructMapper[UserSummary]())
	if !errors.Is(err, ErrNilQuery) {
		t.Fatalf("expected ErrNilQuery, got: %v", err)
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

	insertAssignments := mustInsertAssignments(t, mapper, users, entity)
	if len(insertAssignments) != 4 {
		t.Fatalf("unexpected insert assignment count: %d fields=%+v columns=%+v", len(insertAssignments), mapper.Fields().Values(), users.Columns())
	}
	insertBound, err := InsertInto(users).Values(insertAssignments...).Build(testSQLiteDialect{})
	if err != nil {
		t.Fatalf("insert build returned error: %v", err)
	}
	if insertBound.SQL != `INSERT INTO "users" ("username", "email_address", "status", "role_id") VALUES (?, ?, ?, ?)` {
		t.Fatalf("unexpected insert sql: %q", insertBound.SQL)
	}

	updateAssignments := mustUpdateAssignments(t, mapper, users, entity)
	if len(updateAssignments) != 4 {
		t.Fatalf("unexpected update assignment count: %d", len(updateAssignments))
	}

	predicate := mustPrimaryPredicate(t, mapper, users, entity)
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
