package dbx

import (
	"context"
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
	ID       Column[SnowflakeUser, int64]  `dbx:"id,pk"`
	Username Column[SnowflakeUser, string] `dbx:"username"`
}

type UUIDUser struct {
	ID       string `dbx:"id"`
	Username string `dbx:"username"`
}

type UUIDUserSchema struct {
	Schema[UUIDUser]
	ID       Column[UUIDUser, string] `dbx:"id,pk"`
	Username Column[UUIDUser, string] `dbx:"username"`
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
	core := NewWithOptions(sqlDB, testSQLiteDialect{}, WithHooks(HookFuncs{AfterFunc: rec.after}))

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
	sqlDB, cleanup := OpenTestSQLiteWithSchema(t, `INSERT INTO "roles" ("id","name") VALUES (9,'admin')`)
	defer cleanup()

	users := MustSchema("users", UserSchema{})
	mapper := MustMapper[User](users)
	entity := &User{
		Username: "alice",
		Email:    "alice@example.com",
		Status:   1,
		RoleID:   9,
	}

	assignments, err := mapper.InsertAssignments(users, entity)
	if err != nil {
		t.Fatalf("InsertAssignments returned error: %v", err)
	}

	rec := &hookRecorder{}
	result, err := Exec(context.Background(), NewWithOptions(sqlDB, testSQLiteDialect{}, WithHooks(HookFuncs{AfterFunc: rec.after})), InsertInto(users).Values(assignments...))
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
	if rec.execCount != 1 {
		t.Fatalf("unexpected recorded exec count: %d", rec.execCount)
	}
}

func TestBeginTxExecsWithinTransaction(t *testing.T) {
	sqlDB, cleanup := OpenTestSQLiteWithSchema(t,
		`INSERT INTO "roles" ("id","name") VALUES (1,'r1')`,
		`INSERT INTO "users" ("username","email_address","status","role_id") VALUES ('u','e@x.com',1,1)`,
	)
	defer cleanup()

	users := MustSchema("users", UserSchema{})
	rec := &hookRecorder{}
	core := NewWithOptions(sqlDB, testSQLiteDialect{}, WithHooks(HookFuncs{AfterFunc: rec.after}))
	tx, err := core.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("BeginTx returned error: %v", err)
	}

	result, err := Exec(context.Background(), tx, Update(users).Set(users.Status.Set(2)).Where(users.ID.Eq(1)))
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
	if rec.execCount != 1 {
		t.Fatalf("unexpected recorded exec count: %d", rec.execCount)
	}
}

func TestInsertAssignmentsGenerateSnowflakeID(t *testing.T) {
	users := MustSchema("users", SnowflakeUserSchema{
		ID: NewIDColumn[SnowflakeUser, int64, IDSnowflake](),
	})
	mapper := MustMapper[SnowflakeUser](users)
	entity := &SnowflakeUser{Username: "alice"}

	assignments, err := mapper.InsertAssignments(users, entity)
	if err != nil {
		t.Fatalf("InsertAssignments returned error: %v", err)
	}
	if entity.ID == 0 {
		t.Fatal("expected generated snowflake id")
	}
	if len(assignments) != 2 {
		t.Fatalf("expected id + username assignments, got %d", len(assignments))
	}
}

func TestInsertAssignmentsGenerateUUIDv7ID(t *testing.T) {
	users := MustSchema("users", UUIDUserSchema{
		ID: NewIDColumn[UUIDUser, string, IDUUIDv7](),
	})
	mapper := MustMapper[UUIDUser](users)
	entity := &UUIDUser{Username: "alice"}

	assignments, err := mapper.InsertAssignments(users, entity)
	if err != nil {
		t.Fatalf("InsertAssignments returned error: %v", err)
	}
	if entity.ID == "" {
		t.Fatal("expected generated uuid id")
	}
	if len(assignments) != 2 {
		t.Fatalf("expected id + username assignments, got %d", len(assignments))
	}
}
