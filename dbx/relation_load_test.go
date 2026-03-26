package dbx

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/samber/mo"
)

const relationTestSchemaDDL = `
CREATE TABLE IF NOT EXISTS "roles" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "name" TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS "users" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "name" TEXT NOT NULL, "role_id" INTEGER NOT NULL REFERENCES "roles"("id"));
CREATE TABLE IF NOT EXISTS "profiles" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "user_id" INTEGER NOT NULL REFERENCES "users"("id"), "bio" TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS "posts" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "user_id" INTEGER NOT NULL REFERENCES "users"("id"), "title" TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS "tags" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "name" TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS "user_tags" ("user_id" INTEGER NOT NULL REFERENCES "users"("id"), "tag_id" INTEGER NOT NULL REFERENCES "tags"("id"), PRIMARY KEY ("user_id", "tag_id"));
`

type relationRole struct {
	ID   int64  `dbx:"id"`
	Name string `dbx:"name"`
}

type relationUser struct {
	ID     int64  `dbx:"id"`
	Name   string `dbx:"name"`
	RoleID int64  `dbx:"role_id"`
}

type relationProfile struct {
	ID     int64  `dbx:"id"`
	UserID int64  `dbx:"user_id"`
	Bio    string `dbx:"bio"`
}

type relationPost struct {
	ID     int64  `dbx:"id"`
	UserID int64  `dbx:"user_id"`
	Title  string `dbx:"title"`
}

type relationTag struct {
	ID   int64  `dbx:"id"`
	Name string `dbx:"name"`
}

type relationRoleSchema struct {
	Schema[relationRole]
	ID   Column[relationRole, int64]  `dbx:"id,pk,auto"`
	Name Column[relationRole, string] `dbx:"name"`
}

type relationProfileSchema struct {
	Schema[relationProfile]
	ID     Column[relationProfile, int64]  `dbx:"id,pk,auto"`
	UserID Column[relationProfile, int64]  `dbx:"user_id"`
	Bio    Column[relationProfile, string] `dbx:"bio"`
}

type relationPostSchema struct {
	Schema[relationPost]
	ID     Column[relationPost, int64]  `dbx:"id,pk,auto"`
	UserID Column[relationPost, int64]  `dbx:"user_id"`
	Title  Column[relationPost, string] `dbx:"title"`
}

type relationTagSchema struct {
	Schema[relationTag]
	ID   Column[relationTag, int64]  `dbx:"id,pk,auto"`
	Name Column[relationTag, string] `dbx:"name"`
}

type relationUserSchema struct {
	Schema[relationUser]
	ID      Column[relationUser, int64]           `dbx:"id,pk,auto"`
	Name    Column[relationUser, string]          `dbx:"name"`
	RoleID  Column[relationUser, int64]           `dbx:"role_id"`
	Role    BelongsTo[relationUser, relationRole] `rel:"table=roles,local=role_id,target=id"`
	Profile HasOne[relationUser, relationProfile] `rel:"table=profiles,local=id,target=user_id"`
	Posts   HasMany[relationUser, relationPost]   `rel:"table=posts,local=id,target=user_id"`
	Tags    ManyToMany[relationUser, relationTag] `rel:"table=tags,target=id,join=user_tags,join_local=user_id,join_target=tag_id"`
}

func TestLoadBelongsToBatchesAndAttaches(t *testing.T) {
	sqlDB, cleanup := OpenTestSQLite(t, relationTestSchemaDDL,
		`INSERT INTO "roles" ("id","name") VALUES (2,'admin')`,
	)
	defer cleanup()

	users := MustSchema("users", relationUserSchema{})
	roles := MustSchema("roles", relationRoleSchema{})
	items := []relationUser{
		{ID: 1, Name: "alice", RoleID: 2},
		{ID: 2, Name: "bob", RoleID: 4},
	}
	loaded := make([]mo.Option[relationRole], len(items))

	err := LoadBelongsTo(
		context.Background(),
		New(sqlDB, testSQLiteDialect{}),
		items,
		users,
		MustMapper[relationUser](users),
		users.Role,
		roles,
		MustMapper[relationRole](roles),
		func(index int, _ *relationUser, value mo.Option[relationRole]) {
			loaded[index] = value
		},
	)
	if err != nil {
		t.Fatalf("LoadBelongsTo returned error: %v", err)
	}
	if loaded[0].IsAbsent() {
		t.Fatal("expected first user role to be loaded")
	}
	role, _ := loaded[0].Get()
	if role.Name != "admin" {
		t.Fatalf("unexpected belongs-to payload: %+v", role)
	}
	if loaded[1].IsPresent() {
		t.Fatalf("expected second user role to be absent: %+v", loaded[1])
	}
}

func TestLoadHasOneBatchesAndAttaches(t *testing.T) {
	sqlDB, cleanup := OpenTestSQLite(t, relationTestSchemaDDL,
		`INSERT INTO "roles" ("id","name") VALUES (1,'r')`,
		`INSERT INTO "users" ("id","name","role_id") VALUES (1,'alice',1),(2,'bob',1)`,
		`INSERT INTO "profiles" ("id","user_id","bio") VALUES (10,1,'hello')`,
	)
	defer cleanup()

	users := MustSchema("users", relationUserSchema{})
	profiles := MustSchema("profiles", relationProfileSchema{})
	items := []relationUser{{ID: 1, Name: "alice"}, {ID: 2, Name: "bob"}}
	loaded := make([]mo.Option[relationProfile], len(items))

	err := LoadHasOne(
		context.Background(),
		New(sqlDB, testSQLiteDialect{}),
		items,
		users,
		MustMapper[relationUser](users),
		users.Profile,
		profiles,
		MustMapper[relationProfile](profiles),
		func(index int, _ *relationUser, value mo.Option[relationProfile]) {
			loaded[index] = value
		},
	)
	if err != nil {
		t.Fatalf("LoadHasOne returned error: %v", err)
	}
	if loaded[0].IsAbsent() {
		t.Fatal("expected first user profile to be loaded")
	}
	if loaded[1].IsPresent() {
		t.Fatalf("expected second user profile to be absent: %+v", loaded[1])
	}
}

func TestLoadHasManyBatchesAndAttaches(t *testing.T) {
	sqlDB, cleanup := OpenTestSQLite(t, relationTestSchemaDDL,
		`INSERT INTO "roles" ("id","name") VALUES (1,'r')`,
		`INSERT INTO "users" ("id","name","role_id") VALUES (1,'alice',1),(2,'bob',1)`,
		`INSERT INTO "posts" ("id","user_id","title") VALUES (100,1,'first'),(101,1,'second'),(200,2,'third')`,
	)
	defer cleanup()

	users := MustSchema("users", relationUserSchema{})
	posts := MustSchema("posts", relationPostSchema{})
	items := []relationUser{{ID: 1, Name: "alice"}, {ID: 2, Name: "bob"}}
	loaded := make([][]relationPost, len(items))

	err := LoadHasMany(
		context.Background(),
		New(sqlDB, testSQLiteDialect{}),
		items,
		users,
		MustMapper[relationUser](users),
		users.Posts,
		posts,
		MustMapper[relationPost](posts),
		func(index int, _ *relationUser, value []relationPost) {
			loaded[index] = value
		},
	)
	if err != nil {
		t.Fatalf("LoadHasMany returned error: %v", err)
	}
	if len(loaded[0]) != 2 || len(loaded[1]) != 1 {
		t.Fatalf("unexpected has-many payload: %+v", loaded)
	}
	if loaded[0][1].Title != "second" || loaded[1][0].Title != "third" {
		t.Fatalf("unexpected has-many rows: %+v", loaded)
	}
}

func TestLoadManyToManyBatchesAndAttaches(t *testing.T) {
	sqlDB, cleanup := OpenTestSQLite(t, relationTestSchemaDDL,
		`INSERT INTO "roles" ("id","name") VALUES (1,'r')`,
		`INSERT INTO "users" ("id","name","role_id") VALUES (1,'alice',1),(2,'bob',1)`,
		`INSERT INTO "tags" ("id","name") VALUES (10,'red'),(11,'blue')`,
		`INSERT INTO "user_tags" ("user_id","tag_id") VALUES (1,10),(1,11),(2,11)`,
	)
	defer cleanup()

	users := MustSchema("users", relationUserSchema{})
	tags := MustSchema("tags", relationTagSchema{})
	items := []relationUser{{ID: 1, Name: "alice"}, {ID: 2, Name: "bob"}}
	loaded := make([][]relationTag, len(items))

	err := LoadManyToMany(
		context.Background(),
		New(sqlDB, testSQLiteDialect{}),
		items,
		users,
		MustMapper[relationUser](users),
		users.Tags,
		tags,
		MustMapper[relationTag](tags),
		func(index int, _ *relationUser, value []relationTag) {
			loaded[index] = value
		},
	)
	if err != nil {
		t.Fatalf("LoadManyToMany returned error: %v", err)
	}
	if len(loaded[0]) != 2 || len(loaded[1]) != 1 {
		t.Fatalf("unexpected many-to-many payload: %+v", loaded)
	}
	if loaded[0][0].Name != "red" || loaded[0][1].Name != "blue" || loaded[1][0].Name != "blue" {
		t.Fatalf("unexpected many-to-many rows: %+v", loaded)
	}
}

func TestLoadBelongsToChunksLargeINQueries(t *testing.T) {
	sqlDB, cleanup := OpenTestSQLite(t, relationTestSchemaDDL)
	defer cleanup()

	for i := 1; i <= 950; i++ {
		if _, err := sqlDB.Exec(`INSERT INTO "roles" ("id","name") VALUES (?, ?)`, i, fmt.Sprintf("role-%d", i)); err != nil {
			t.Fatalf("insert role %d: %v", i, err)
		}
	}

	users := MustSchema("users", relationUserSchema{})
	roles := MustSchema("roles", relationRoleSchema{})
	items := make([]relationUser, 0, 950)
	for i := 1; i <= 950; i++ {
		items = append(items, relationUser{ID: int64(i), Name: fmt.Sprintf("user-%d", i), RoleID: int64(i)})
	}
	loaded := make([]mo.Option[relationRole], len(items))

	err := LoadBelongsTo(
		context.Background(),
		New(sqlDB, testSQLiteDialect{}),
		items,
		users,
		MustMapper[relationUser](users),
		users.Role,
		roles,
		MustMapper[relationRole](roles),
		func(index int, _ *relationUser, value mo.Option[relationRole]) {
			loaded[index] = value
		},
	)
	if err != nil {
		t.Fatalf("LoadBelongsTo returned error: %v", err)
	}
	last, ok := loaded[len(loaded)-1].Get()
	if !ok || last.Name != "role-950" {
		t.Fatalf("expected final role to load after chunking, got ok=%v value=%+v", ok, last)
	}
}

func TestLoadManyToManyChunksLargePairQueries(t *testing.T) {
	sqlDB, cleanup := OpenTestSQLite(t, relationTestSchemaDDL)
	defer cleanup()

	if _, err := sqlDB.Exec(`INSERT INTO "roles" ("id","name") VALUES (1,'r')`); err != nil {
		t.Fatalf("insert role: %v", err)
	}
	if _, err := sqlDB.Exec(`INSERT INTO "tags" ("id","name") VALUES (1,'shared')`); err != nil {
		t.Fatalf("insert tag: %v", err)
	}
	for i := 1; i <= 950; i++ {
		if _, err := sqlDB.Exec(`INSERT INTO "users" ("id","name","role_id") VALUES (?,?,1)`, i, fmt.Sprintf("user-%d", i)); err != nil {
			t.Fatalf("insert user %d: %v", i, err)
		}
		if _, err := sqlDB.Exec(`INSERT INTO "user_tags" ("user_id","tag_id") VALUES (?,1)`, i); err != nil {
			t.Fatalf("insert pair %d: %v", i, err)
		}
	}

	users := MustSchema("users", relationUserSchema{})
	tags := MustSchema("tags", relationTagSchema{})
	items := make([]relationUser, 0, 950)
	for i := 1; i <= 950; i++ {
		items = append(items, relationUser{ID: int64(i), Name: fmt.Sprintf("user-%d", i)})
	}
	loaded := make([][]relationTag, len(items))

	err := LoadManyToMany(
		context.Background(),
		New(sqlDB, testSQLiteDialect{}),
		items,
		users,
		MustMapper[relationUser](users),
		users.Tags,
		tags,
		MustMapper[relationTag](tags),
		func(index int, _ *relationUser, value []relationTag) {
			loaded[index] = value
		},
	)
	if err != nil {
		t.Fatalf("LoadManyToMany returned error: %v", err)
	}
	if len(loaded[0]) != 1 || loaded[0][0].Name != "shared" {
		t.Fatalf("unexpected first many-to-many payload: %+v", loaded[0])
	}
	if len(loaded[len(loaded)-1]) != 1 || loaded[len(loaded)-1][0].Name != "shared" {
		t.Fatalf("unexpected final many-to-many payload: %+v", loaded[len(loaded)-1])
	}
}

func TestLoadHasManyReturnsDeterministicOrder(t *testing.T) {
	sqlDB, cleanup := OpenTestSQLite(t, relationTestSchemaDDL,
		`INSERT INTO "roles" ("id","name") VALUES (1,'r')`,
		`INSERT INTO "users" ("id","name","role_id") VALUES (1,'alice',1)`,
		`INSERT INTO "posts" ("id","user_id","title") VALUES (200,1,'second'),(100,1,'first')`,
	)
	defer cleanup()

	users := MustSchema("users", relationUserSchema{})
	posts := MustSchema("posts", relationPostSchema{})
	items := []relationUser{{ID: 1, Name: "alice"}}
	loaded := make([][]relationPost, len(items))

	err := LoadHasMany(
		context.Background(),
		New(sqlDB, testSQLiteDialect{}),
		items,
		users,
		MustMapper[relationUser](users),
		users.Posts,
		posts,
		MustMapper[relationPost](posts),
		func(index int, _ *relationUser, value []relationPost) {
			loaded[index] = value
		},
	)
	if err != nil {
		t.Fatalf("LoadHasMany returned error: %v", err)
	}
	if len(loaded[0]) != 2 || loaded[0][0].ID != 100 || loaded[0][1].ID != 200 {
		t.Fatalf("expected deterministic has-many order by primary key, got %+v", loaded[0])
	}
}

func TestLoadHasOneRejectsDuplicateTargets(t *testing.T) {
	sqlDB, cleanup := OpenTestSQLite(t, relationTestSchemaDDL,
		`INSERT INTO "roles" ("id","name") VALUES (1,'r')`,
		`INSERT INTO "users" ("id","name","role_id") VALUES (1,'alice',1)`,
		`INSERT INTO "profiles" ("id","user_id","bio") VALUES (10,1,'one'),(11,1,'two')`,
	)
	defer cleanup()

	users := MustSchema("users", relationUserSchema{})
	profiles := MustSchema("profiles", relationProfileSchema{})
	items := []relationUser{{ID: 1, Name: "alice"}}

	err := LoadHasOne(
		context.Background(),
		New(sqlDB, testSQLiteDialect{}),
		items,
		users,
		MustMapper[relationUser](users),
		users.Profile,
		profiles,
		MustMapper[relationProfile](profiles),
		func(int, *relationUser, mo.Option[relationProfile]) {},
	)
	if !errors.Is(err, ErrRelationCardinality) {
		t.Fatalf("expected relation cardinality error, got: %v", err)
	}
}
