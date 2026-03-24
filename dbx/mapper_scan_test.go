package dbx

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"
	"testing"
)

type accountLabel string

func (l *accountLabel) Scan(src any) error {
	switch value := src.(type) {
	case string:
		*l = accountLabel(strings.ToUpper(value))
		return nil
	case []byte:
		*l = accountLabel(strings.ToUpper(string(value)))
		return nil
	default:
		return fmt.Errorf("unsupported scan type %T", src)
	}
}

func (l accountLabel) Value() (driver.Value, error) {
	return strings.ToLower(string(l)), nil
}

type AccountProfile struct {
	Nickname *string        `dbx:"nickname"`
	Bio      sql.NullString `dbx:"bio"`
}

type accountRecord struct {
	ID int64 `dbx:"id"`
	*AccountProfile
	Label accountLabel `dbx:"label"`
}

type auditFields struct {
	CreatedBy string `dbx:"created_by"`
	UpdatedBy string `dbx:"updated_by"`
}

type auditedUser struct {
	ID    int64       `dbx:"id"`
	Audit auditFields `dbx:",inline"`
}

type accountSchema struct {
	Schema[accountRecord]
	ID       Column[accountRecord, int64]          `dbx:"id,pk,auto"`
	Nickname Column[accountRecord, *string]        `dbx:"nickname,nullable"`
	Bio      Column[accountRecord, sql.NullString] `dbx:"bio,nullable"`
	Label    Column[accountRecord, accountLabel]   `dbx:"label"`
}

const mapperScanAccountsDDL = `
CREATE TABLE IF NOT EXISTS "accounts" (
	"id" INTEGER PRIMARY KEY AUTOINCREMENT,
	"nickname" TEXT,
	"bio" TEXT,
	"label" TEXT NOT NULL
);
`

func TestStructMapperScansEmbeddedPointerNullableAndScanner(t *testing.T) {
	sqlDB, cleanup := OpenTestSQLite(t, mapperScanAccountsDDL,
		`INSERT INTO "accounts" ("id","nickname","bio","label") VALUES (1,'ally','hello','admin'),(2,NULL,NULL,'reader')`,
	)
	defer cleanup()

	accounts := MustSchema("accounts", accountSchema{})
	mapper := MustStructMapper[accountRecord]()

	items, err := QueryAll(context.Background(), New(sqlDB, testSQLiteDialect{}), Select(accounts.AllColumns()...).From(accounts), mapper)
	if err != nil {
		t.Fatalf("QueryAll returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("unexpected item count: %d", len(items))
	}
	if items[0].AccountProfile == nil {
		t.Fatal("expected embedded profile to be allocated")
	}
	if items[0].Nickname == nil || *items[0].Nickname != "ally" {
		t.Fatalf("unexpected nickname: %+v", items[0].Nickname)
	}
	if !items[0].Bio.Valid || items[0].Bio.String != "hello" {
		t.Fatalf("unexpected bio: %+v", items[0].Bio)
	}
	if items[0].Label != "ADMIN" {
		t.Fatalf("unexpected custom scanner label: %q", items[0].Label)
	}
	if items[1].Nickname != nil {
		t.Fatalf("expected nil nickname, got: %+v", items[1].Nickname)
	}
	if items[1].Bio.Valid {
		t.Fatalf("expected invalid bio, got: %+v", items[1].Bio)
	}
	if items[1].Label != "READER" {
		t.Fatalf("unexpected second label: %q", items[1].Label)
	}
}

func TestMapperInsertAssignmentsWithNilEmbeddedPointerAndValuer(t *testing.T) {
	sqlDB, cleanup := OpenTestSQLite(t, mapperScanAccountsDDL)
	defer cleanup()

	accounts := MustSchema("accounts", accountSchema{})
	mapper := MustMapper[accountRecord](accounts)
	entity := &accountRecord{
		Label: "ADMIN",
	}

	assignments, err := mapper.InsertAssignments(New(nil, testSQLiteDialect{}), accounts, entity)
	if err != nil {
		t.Fatalf("InsertAssignments returned error: %v", err)
	}
	if len(assignments) != 3 {
		t.Fatalf("unexpected assignment count: %d", len(assignments))
	}

	rec := &hookRecorder{}
	if _, err := Exec(context.Background(), MustNewWithOptions(sqlDB, testSQLiteDialect{}, WithHooks(HookFuncs{AfterFunc: rec.after})), InsertInto(accounts).Values(assignments...)); err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if rec.execCount != 1 {
		t.Fatalf("unexpected exec count: %d", rec.execCount)
	}
}

func TestStructMapperSupportsNamedInlineFields(t *testing.T) {
	mapper := MustStructMapper[auditedUser]()

	fields := mapper.Fields()
	if len(fields) != 3 {
		t.Fatalf("unexpected mapped field count: %d", len(fields))
	}
	createdBy, ok := mapper.FieldByColumn("created_by")
	if !ok {
		t.Fatal("expected created_by mapping")
	}
	if len(createdBy.Path) != 2 {
		t.Fatalf("expected inline field path depth=2, got: %+v", createdBy.Path)
	}
	if createdBy.Path[0] != 1 {
		t.Fatalf("unexpected inline field root path: %+v", createdBy.Path)
	}
}

func TestStructMapperScanPlanMatchesQualifiedAndCaseInsensitiveColumns(t *testing.T) {
	sqlDB, cleanup := OpenTestSQLiteWithSchema(t,
		`INSERT INTO "roles" ("id","name") VALUES (1,'r1')`,
		`INSERT INTO "users" ("username","email_address","status","role_id") VALUES ('a','a@x.com',1,1),('b','b@x.com',1,1)`,
	)
	defer cleanup()

	type aggregateRow struct {
		ID        int64 `dbx:"id"`
		UserCount int64 `dbx:"user_count"`
	}

	users := MustSchema("users", UserSchema{})
	items, err := QueryAll(
		context.Background(),
		New(sqlDB, testSQLiteDialect{}),
		Select(users.ID, CountAll().As("user_count")).From(users),
		MustStructMapper[aggregateRow](),
	)
	if err != nil {
		t.Fatalf("QueryAll returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected item count: %d", len(items))
	}
	if items[0].ID != 1 || items[0].UserCount != 2 {
		t.Fatalf("unexpected aggregate row: %+v", items[0])
	}
}

func TestQueryCursorScansEmbeddedPointerNullableAndScanner(t *testing.T) {
	sqlDB, cleanup := OpenTestSQLite(t, mapperScanAccountsDDL,
		`INSERT INTO "accounts" ("id","nickname","bio","label") VALUES (1,'ally','hello','admin'),(2,NULL,NULL,'reader')`,
	)
	defer cleanup()

	accounts := MustSchema("accounts", accountSchema{})
	mapper := MustStructMapper[accountRecord]()
	core := New(sqlDB, testSQLiteDialect{})
	query := Select(accounts.AllColumns()...).From(accounts)

	cursor, err := QueryCursor(context.Background(), core, query, mapper)
	if err != nil {
		t.Fatalf("QueryCursor returned error: %v", err)
	}
	defer cursor.Close()

	var items []accountRecord
	for cursor.Next() {
		item, err := cursor.Get()
		if err != nil {
			t.Fatalf("cursor.Get returned error: %v", err)
		}
		items = append(items, item)
	}
	if err := cursor.Err(); err != nil {
		t.Fatalf("cursor.Err returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("unexpected item count: %d", len(items))
	}
	if items[0].AccountProfile == nil || items[0].Nickname == nil || *items[0].Nickname != "ally" {
		t.Fatalf("unexpected first item: %+v", items[0])
	}
	if items[1].Label != "READER" {
		t.Fatalf("unexpected second label: %q", items[1].Label)
	}

	var fromEach []accountRecord
	QueryEach(context.Background(), core, query, mapper)(func(item accountRecord, err error) bool {
		if err != nil {
			t.Fatalf("QueryEach yielded error: %v", err)
		}
		fromEach = append(fromEach, item)
		return true
	})
	if len(fromEach) != 2 || fromEach[0].Label != "ADMIN" || fromEach[1].Label != "READER" {
		t.Fatalf("unexpected each items: %+v", fromEach)
	}
}
