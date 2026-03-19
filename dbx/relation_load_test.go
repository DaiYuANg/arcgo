package dbx

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/DaiYuANg/arcgo/dbx/internal/testsql"
	"github.com/samber/mo"
)

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
	sqlDB, _, cleanup, err := testsql.Open(testsql.Plan{
		Queries: []testsql.QueryPlan{
			{
				SQL:     `SELECT "roles"."id", "roles"."name" FROM "roles" WHERE "roles"."id" IN (?, ?)`,
				Args:    []driver.Value{int64(2), int64(4)},
				Columns: []string{"id", "name"},
				Rows: [][]driver.Value{
					{int64(2), "admin"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("testsql.Open returned error: %v", err)
	}
	defer cleanup()

	users := MustSchema("users", relationUserSchema{})
	roles := MustSchema("roles", relationRoleSchema{})
	items := []relationUser{
		{ID: 1, Name: "alice", RoleID: 2},
		{ID: 2, Name: "bob", RoleID: 4},
	}
	loaded := make([]mo.Option[relationRole], len(items))

	err = LoadBelongsTo(
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
	sqlDB, _, cleanup, err := testsql.Open(testsql.Plan{
		Queries: []testsql.QueryPlan{
			{
				SQL:     `SELECT "profiles"."id", "profiles"."user_id", "profiles"."bio" FROM "profiles" WHERE "profiles"."user_id" IN (?, ?)`,
				Args:    []driver.Value{int64(1), int64(2)},
				Columns: []string{"id", "user_id", "bio"},
				Rows: [][]driver.Value{
					{int64(10), int64(1), "hello"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("testsql.Open returned error: %v", err)
	}
	defer cleanup()

	users := MustSchema("users", relationUserSchema{})
	profiles := MustSchema("profiles", relationProfileSchema{})
	items := []relationUser{{ID: 1, Name: "alice"}, {ID: 2, Name: "bob"}}
	loaded := make([]mo.Option[relationProfile], len(items))

	err = LoadHasOne(
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
	sqlDB, _, cleanup, err := testsql.Open(testsql.Plan{
		Queries: []testsql.QueryPlan{
			{
				SQL:     `SELECT "posts"."id", "posts"."user_id", "posts"."title" FROM "posts" WHERE "posts"."user_id" IN (?, ?)`,
				Args:    []driver.Value{int64(1), int64(2)},
				Columns: []string{"id", "user_id", "title"},
				Rows: [][]driver.Value{
					{int64(100), int64(1), "first"},
					{int64(101), int64(1), "second"},
					{int64(200), int64(2), "third"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("testsql.Open returned error: %v", err)
	}
	defer cleanup()

	users := MustSchema("users", relationUserSchema{})
	posts := MustSchema("posts", relationPostSchema{})
	items := []relationUser{{ID: 1, Name: "alice"}, {ID: 2, Name: "bob"}}
	loaded := make([][]relationPost, len(items))

	err = LoadHasMany(
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
	sqlDB, _, cleanup, err := testsql.Open(testsql.Plan{
		Queries: []testsql.QueryPlan{
			{
				SQL:     `SELECT "user_tags"."user_id", "user_tags"."tag_id" FROM "user_tags" WHERE "user_tags"."user_id" IN (?, ?)`,
				Args:    []driver.Value{int64(1), int64(2)},
				Columns: []string{"user_id", "tag_id"},
				Rows: [][]driver.Value{
					{int64(1), int64(10)},
					{int64(1), int64(11)},
					{int64(2), int64(11)},
				},
			},
			{
				SQL:     `SELECT "tags"."id", "tags"."name" FROM "tags" WHERE "tags"."id" IN (?, ?)`,
				Args:    []driver.Value{int64(10), int64(11)},
				Columns: []string{"id", "name"},
				Rows: [][]driver.Value{
					{int64(10), "red"},
					{int64(11), "blue"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("testsql.Open returned error: %v", err)
	}
	defer cleanup()

	users := MustSchema("users", relationUserSchema{})
	tags := MustSchema("tags", relationTagSchema{})
	items := []relationUser{{ID: 1, Name: "alice"}, {ID: 2, Name: "bob"}}
	loaded := make([][]relationTag, len(items))

	err = LoadManyToMany(
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
