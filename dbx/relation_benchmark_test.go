package dbx

import (
	"context"
	"testing"

	"github.com/samber/mo"
)

func BenchmarkLoadBelongsTo(b *testing.B) {
	users := MustSchema("users", relationUserSchema{})
	roles := MustSchema("roles", relationRoleSchema{})
	items := []relationUser{{ID: 1, Name: "alice", RoleID: 2}, {ID: 2, Name: "bob", RoleID: 4}}

	sqlDB, cleanup := OpenTestSQLite(b, relationTestSchemaDDL,
		`INSERT INTO "roles" ("id","name") VALUES (2,'admin')`,
	)
	defer cleanup()

	core := New(sqlDB, testSQLiteDialect{})
	sourceMapper := MustMapper[relationUser](users)
	targetMapper := MustMapper[relationRole](roles)
	loaded := make([]mo.Option[relationRole], len(items))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := LoadBelongsTo(context.Background(), core, items, users, sourceMapper, users.Role, roles, targetMapper, func(index int, _ *relationUser, value mo.Option[relationRole]) {
			loaded[index] = value
		}); err != nil {
			b.Fatalf("LoadBelongsTo returned error: %v", err)
		}
	}
}

func BenchmarkLoadHasMany(b *testing.B) {
	users := MustSchema("users", relationUserSchema{})
	posts := MustSchema("posts", relationPostSchema{})
	items := []relationUser{{ID: 1, Name: "alice"}, {ID: 2, Name: "bob"}}

	sqlDB, cleanup := OpenTestSQLite(b, relationTestSchemaDDL,
		`INSERT INTO "roles" ("id","name") VALUES (1,'r')`,
		`INSERT INTO "users" ("id","name","role_id") VALUES (1,'alice',1),(2,'bob',1)`,
		`INSERT INTO "posts" ("id","user_id","title") VALUES (100,1,'first'),(101,1,'second'),(200,2,'third')`,
	)
	defer cleanup()

	core := New(sqlDB, testSQLiteDialect{})
	sourceMapper := MustMapper[relationUser](users)
	targetMapper := MustMapper[relationPost](posts)
	loaded := make([][]relationPost, len(items))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := LoadHasMany(context.Background(), core, items, users, sourceMapper, users.Posts, posts, targetMapper, func(index int, _ *relationUser, value []relationPost) {
			loaded[index] = value
		}); err != nil {
			b.Fatalf("LoadHasMany returned error: %v", err)
		}
	}
}

func BenchmarkLoadManyToMany(b *testing.B) {
	users := MustSchema("users", relationUserSchema{})
	tags := MustSchema("tags", relationTagSchema{})
	items := []relationUser{{ID: 1, Name: "alice"}, {ID: 2, Name: "bob"}}

	sqlDB, cleanup := OpenTestSQLite(b, relationTestSchemaDDL,
		`INSERT INTO "roles" ("id","name") VALUES (1,'r')`,
		`INSERT INTO "users" ("id","name","role_id") VALUES (1,'alice',1),(2,'bob',1)`,
		`INSERT INTO "tags" ("id","name") VALUES (10,'red'),(11,'blue')`,
		`INSERT INTO "user_tags" ("user_id","tag_id") VALUES (1,10),(1,11),(2,11)`,
	)
	defer cleanup()

	core := New(sqlDB, testSQLiteDialect{})
	sourceMapper := MustMapper[relationUser](users)
	targetMapper := MustMapper[relationTag](tags)
	loaded := make([][]relationTag, len(items))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := LoadManyToMany(context.Background(), core, items, users, sourceMapper, users.Tags, tags, targetMapper, func(index int, _ *relationUser, value []relationTag) {
			loaded[index] = value
		}); err != nil {
			b.Fatalf("LoadManyToMany returned error: %v", err)
		}
	}
}
