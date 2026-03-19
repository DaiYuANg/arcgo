package dbx

import (
	"reflect"
	"testing"
)

func TestSelectBuildSQLite(t *testing.T) {
	users := MustSchema("users", UserSchema{})
	roles := MustSchema("roles", RoleSchema{})

	query := Select(users.ID, users.Username, roles.Name).
		From(users).
		Join(roles).On(users.RoleID.EqColumn(roles.ID)).
		Where(And(users.Status.Eq(1), Like(users.Username, "a%"))).
		OrderBy(users.ID.Desc()).
		Limit(20).
		Offset(10)

	bound, err := query.Build(testSQLiteDialect{})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	expectedSQL := `SELECT "users"."id", "users"."username", "roles"."name" FROM "users" INNER JOIN "roles" ON "users"."role_id" = "roles"."id" WHERE ("users"."status" = ? AND "users"."username" LIKE ?) ORDER BY "users"."id" DESC LIMIT 20 OFFSET 10`
	if bound.SQL != expectedSQL {
		t.Fatalf("unexpected sqlite select SQL:\nwant: %s\n got: %s", expectedSQL, bound.SQL)
	}
	if !reflect.DeepEqual(bound.Args, []any{1, "a%"}) {
		t.Fatalf("unexpected sqlite select args: %#v", bound.Args)
	}
}

func TestSelectBuildPostgresWithAliasAndIn(t *testing.T) {
	users := Alias(MustSchema("users", UserSchema{}), "u")

	query := Select(users.ID, users.Username).
		From(users).
		Where(users.ID.In(int64(1), int64(2), int64(3))).
		Offset(5)

	bound, err := query.Build(testPostgresDialect{})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	expectedSQL := `SELECT "u"."id", "u"."username" FROM "users" AS "u" WHERE "u"."id" IN ($1, $2, $3) OFFSET 5`
	if bound.SQL != expectedSQL {
		t.Fatalf("unexpected postgres select SQL:\nwant: %s\n got: %s", expectedSQL, bound.SQL)
	}
	if !reflect.DeepEqual(bound.Args, []any{int64(1), int64(2), int64(3)}) {
		t.Fatalf("unexpected postgres select args: %#v", bound.Args)
	}
}

func TestInsertUpdateDeleteBuildMySQL(t *testing.T) {
	users := MustSchema("users", UserSchema{})

	insertBound, err := InsertInto(users).
		Values(users.Username.Set("alice"), users.Status.Set(1)).
		Build(testMySQLDialect{})
	if err != nil {
		t.Fatalf("insert Build returned error: %v", err)
	}
	insertSQL := "INSERT INTO `users` (`username`, `status`) VALUES (?, ?)"
	if insertBound.SQL != insertSQL {
		t.Fatalf("unexpected mysql insert SQL:\nwant: %s\n got: %s", insertSQL, insertBound.SQL)
	}
	if !reflect.DeepEqual(insertBound.Args, []any{"alice", 1}) {
		t.Fatalf("unexpected mysql insert args: %#v", insertBound.Args)
	}

	updateBound, err := Update(users).
		Set(users.Status.Set(2)).
		Where(users.ID.Eq(int64(10))).
		Build(testMySQLDialect{})
	if err != nil {
		t.Fatalf("update Build returned error: %v", err)
	}
	updateSQL := "UPDATE `users` SET `status` = ? WHERE `users`.`id` = ?"
	if updateBound.SQL != updateSQL {
		t.Fatalf("unexpected mysql update SQL:\nwant: %s\n got: %s", updateSQL, updateBound.SQL)
	}
	if !reflect.DeepEqual(updateBound.Args, []any{2, int64(10)}) {
		t.Fatalf("unexpected mysql update args: %#v", updateBound.Args)
	}

	deleteBound, err := DeleteFrom(users).
		Where(users.ID.Eq(int64(10))).
		Build(testMySQLDialect{})
	if err != nil {
		t.Fatalf("delete Build returned error: %v", err)
	}
	deleteSQL := "DELETE FROM `users` WHERE `users`.`id` = ?"
	if deleteBound.SQL != deleteSQL {
		t.Fatalf("unexpected mysql delete SQL:\nwant: %s\n got: %s", deleteSQL, deleteBound.SQL)
	}
	if !reflect.DeepEqual(deleteBound.Args, []any{int64(10)}) {
		t.Fatalf("unexpected mysql delete args: %#v", deleteBound.Args)
	}
}

func TestJoinRelationBuildSQLite(t *testing.T) {
	users := Alias(MustSchema("users", UserSchema{}), "u")
	roles := Alias(MustSchema("roles", RoleSchema{}), "r")

	query := Select(users.ID, roles.Name).From(users)
	if _, err := query.JoinRelation(users, users.Role, roles); err != nil {
		t.Fatalf("JoinRelation returned error: %v", err)
	}

	bound, err := query.Build(testSQLiteDialect{})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	expectedSQL := `SELECT "u"."id", "r"."name" FROM "users" AS "u" INNER JOIN "roles" AS "r" ON "u"."role_id" = "r"."id"`
	if bound.SQL != expectedSQL {
		t.Fatalf("unexpected sqlite relation join SQL:\nwant: %s\n got: %s", expectedSQL, bound.SQL)
	}
}

func TestJoinRelationManyToManyBuildSQLite(t *testing.T) {
	users := Alias(MustSchema("users", UserSchema{}), "u")
	roles := Alias(MustSchema("roles", RoleSchema{}), "r")

	query := Select(users.ID, roles.Name).From(users)
	if _, err := query.JoinRelation(users, users.Roles, roles); err != nil {
		t.Fatalf("JoinRelation returned error: %v", err)
	}

	bound, err := query.Build(testSQLiteDialect{})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	expectedSQL := `SELECT "u"."id", "r"."name" FROM "users" AS "u" INNER JOIN "user_roles" ON "u"."id" = "user_roles"."user_id" INNER JOIN "roles" AS "r" ON "user_roles"."role_id" = "r"."id"`
	if bound.SQL != expectedSQL {
		t.Fatalf("unexpected sqlite many-to-many relation join SQL:\nwant: %s\n got: %s", expectedSQL, bound.SQL)
	}
}

func TestSelectBuildWithGroupByHavingAndAggregates(t *testing.T) {
	users := MustSchema("users", UserSchema{})

	query := Select(
		users.Status,
		CountAll().As("user_count"),
	).
		From(users).
		WithDistinct().
		GroupBy(users.Status).
		Having(CountAll().Gt(int64(1))).
		OrderBy(users.Status.Asc())

	bound, err := query.Build(testSQLiteDialect{})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	expectedSQL := `SELECT DISTINCT "users"."status", COUNT(*) AS "user_count" FROM "users" GROUP BY "users"."status" HAVING COUNT(*) > ? ORDER BY "users"."status" ASC`
	if bound.SQL != expectedSQL {
		t.Fatalf("unexpected aggregate sql:\nwant: %s\n got: %s", expectedSQL, bound.SQL)
	}
	if !reflect.DeepEqual(bound.Args, []any{int64(1)}) {
		t.Fatalf("unexpected aggregate args: %#v", bound.Args)
	}
}

func TestSelectBuildWithSubqueryAndExists(t *testing.T) {
	users := MustSchema("users", UserSchema{})
	roles := MustSchema("roles", RoleSchema{})

	subquery := Select(roles.ID).
		From(roles).
		Where(roles.Name.Eq("admin"))

	existsQuery := Select(roles.ID).
		From(roles).
		Where(And(
			roles.ID.EqColumn(users.RoleID),
			roles.Name.Eq("admin"),
		))

	query := Select(users.ID, users.Username).
		From(users).
		Where(And(
			users.RoleID.InQuery(subquery),
			Exists(existsQuery),
		))

	bound, err := query.Build(testPostgresDialect{})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	expectedSQL := `SELECT "users"."id", "users"."username" FROM "users" WHERE ("users"."role_id" IN (SELECT "roles"."id" FROM "roles" WHERE "roles"."name" = $1) AND EXISTS (SELECT "roles"."id" FROM "roles" WHERE ("roles"."id" = "users"."role_id" AND "roles"."name" = $2)))`
	if bound.SQL != expectedSQL {
		t.Fatalf("unexpected subquery sql:\nwant: %s\n got: %s", expectedSQL, bound.SQL)
	}
	if !reflect.DeepEqual(bound.Args, []any{"admin", "admin"}) {
		t.Fatalf("unexpected subquery args: %#v", bound.Args)
	}
}

func TestInsertBuildWithMultipleRows(t *testing.T) {
	users := MustSchema("users", UserSchema{})

	bound, err := InsertInto(users).
		Values(users.Username.Set("alice"), users.Status.Set(1)).
		Values(users.Username.Set("bob"), users.Status.Set(2)).
		Build(testMySQLDialect{})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	expectedSQL := "INSERT INTO `users` (`username`, `status`) VALUES (?, ?), (?, ?)"
	if bound.SQL != expectedSQL {
		t.Fatalf("unexpected batch insert SQL:\nwant: %s\n got: %s", expectedSQL, bound.SQL)
	}
	if !reflect.DeepEqual(bound.Args, []any{"alice", 1, "bob", 2}) {
		t.Fatalf("unexpected batch insert args: %#v", bound.Args)
	}
}

func TestInsertBuildFromSelect(t *testing.T) {
	users := MustSchema("users", UserSchema{})

	source := Select(users.Username, users.Status).
		From(users).
		Where(users.Status.Eq(1))

	bound, err := InsertInto(users).
		Columns(users.Username, users.Status).
		FromSelect(source).
		Build(testPostgresDialect{})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	expectedSQL := `INSERT INTO "users" ("username", "status") SELECT "users"."username", "users"."status" FROM "users" WHERE "users"."status" = $1`
	if bound.SQL != expectedSQL {
		t.Fatalf("unexpected insert-select SQL:\nwant: %s\n got: %s", expectedSQL, bound.SQL)
	}
	if !reflect.DeepEqual(bound.Args, []any{1}) {
		t.Fatalf("unexpected insert-select args: %#v", bound.Args)
	}
}

func TestInsertBuildWithPostgresUpsert(t *testing.T) {
	users := MustSchema("users", UserSchema{})

	bound, err := InsertInto(users).
		Values(users.Username.Set("alice"), users.Status.Set(1)).
		OnConflict(users.Username).
		DoUpdateSet(users.Status.SetExcluded()).
		Build(testPostgresDialect{})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	expectedSQL := `INSERT INTO "users" ("username", "status") VALUES ($1, $2) ON CONFLICT ("username") DO UPDATE SET "status" = EXCLUDED."status"`
	if bound.SQL != expectedSQL {
		t.Fatalf("unexpected postgres upsert SQL:\nwant: %s\n got: %s", expectedSQL, bound.SQL)
	}
	if !reflect.DeepEqual(bound.Args, []any{"alice", 1}) {
		t.Fatalf("unexpected postgres upsert args: %#v", bound.Args)
	}
}

func TestInsertBuildWithMySQLUpsert(t *testing.T) {
	users := MustSchema("users", UserSchema{})

	bound, err := InsertInto(users).
		Values(users.Username.Set("alice"), users.Status.Set(1)).
		OnConflict(users.Username).
		DoUpdateSet(users.Status.SetExcluded()).
		Build(testMySQLDialect{})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	expectedSQL := "INSERT INTO `users` (`username`, `status`) VALUES (?, ?) ON DUPLICATE KEY UPDATE `status` = VALUES(`status`)"
	if bound.SQL != expectedSQL {
		t.Fatalf("unexpected mysql upsert SQL:\nwant: %s\n got: %s", expectedSQL, bound.SQL)
	}
	if !reflect.DeepEqual(bound.Args, []any{"alice", 1}) {
		t.Fatalf("unexpected mysql upsert args: %#v", bound.Args)
	}
}

func TestInsertBuildWithMySQLDoNothing(t *testing.T) {
	users := MustSchema("users", UserSchema{})

	bound, err := InsertInto(users).
		Values(users.Username.Set("alice"), users.Status.Set(1)).
		OnConflict(users.Username).
		DoNothing().
		Build(testMySQLDialect{})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	expectedSQL := "INSERT IGNORE INTO `users` (`username`, `status`) VALUES (?, ?)"
	if bound.SQL != expectedSQL {
		t.Fatalf("unexpected mysql do-nothing SQL:\nwant: %s\n got: %s", expectedSQL, bound.SQL)
	}
}

func TestMutationBuildWithReturning(t *testing.T) {
	users := MustSchema("users", UserSchema{})

	insertBound, err := InsertInto(users).
		Values(users.Username.Set("alice"), users.Status.Set(1)).
		Returning(users.ID, users.Username).
		Build(testPostgresDialect{})
	if err != nil {
		t.Fatalf("insert Build returned error: %v", err)
	}
	expectedInsertSQL := `INSERT INTO "users" ("username", "status") VALUES ($1, $2) RETURNING "users"."id", "users"."username"`
	if insertBound.SQL != expectedInsertSQL {
		t.Fatalf("unexpected insert returning SQL:\nwant: %s\n got: %s", expectedInsertSQL, insertBound.SQL)
	}

	updateBound, err := Update(users).
		Set(users.Status.Set(2)).
		Where(users.ID.Eq(int64(10))).
		Returning(users.ID, users.Status).
		Build(testSQLiteDialect{})
	if err != nil {
		t.Fatalf("update Build returned error: %v", err)
	}
	expectedUpdateSQL := `UPDATE "users" SET "status" = ? WHERE "users"."id" = ? RETURNING "users"."id", "users"."status"`
	if updateBound.SQL != expectedUpdateSQL {
		t.Fatalf("unexpected update returning SQL:\nwant: %s\n got: %s", expectedUpdateSQL, updateBound.SQL)
	}

	deleteBound, err := DeleteFrom(users).
		Where(users.ID.Eq(int64(10))).
		Returning(users.ID).
		Build(testPostgresDialect{})
	if err != nil {
		t.Fatalf("delete Build returned error: %v", err)
	}
	expectedDeleteSQL := `DELETE FROM "users" WHERE "users"."id" = $1 RETURNING "users"."id"`
	if deleteBound.SQL != expectedDeleteSQL {
		t.Fatalf("unexpected delete returning SQL:\nwant: %s\n got: %s", expectedDeleteSQL, deleteBound.SQL)
	}
}

func TestMutationBuildWithUnsupportedReturning(t *testing.T) {
	users := MustSchema("users", UserSchema{})

	_, err := InsertInto(users).
		Values(users.Username.Set("alice"), users.Status.Set(1)).
		Returning(users.ID).
		Build(testMySQLDialect{})
	if err == nil {
		t.Fatal("expected error for mysql returning, got nil")
	}
}

func TestSelectBuildWithCTE(t *testing.T) {
	users := MustSchema("users", UserSchema{})
	activeUsers := NamedTable("active_users")
	activeID := NamedColumn[int64](activeUsers, "id")
	activeUsername := NamedColumn[string](activeUsers, "username")

	query := Select(activeID, activeUsername).
		With("active_users", Select(users.ID, users.Username).From(users).Where(users.Status.Eq(1))).
		From(activeUsers).
		OrderBy(activeID.Asc())

	bound, err := query.Build(testPostgresDialect{})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	expectedSQL := `WITH "active_users" AS (SELECT "users"."id", "users"."username" FROM "users" WHERE "users"."status" = $1) SELECT "active_users"."id", "active_users"."username" FROM "active_users" ORDER BY "active_users"."id" ASC`
	if bound.SQL != expectedSQL {
		t.Fatalf("unexpected cte sql:\nwant: %s\n got: %s", expectedSQL, bound.SQL)
	}
	if !reflect.DeepEqual(bound.Args, []any{1}) {
		t.Fatalf("unexpected cte args: %#v", bound.Args)
	}
}

func TestSelectBuildWithUnionAllAndOuterOrder(t *testing.T) {
	users := MustSchema("users", UserSchema{})
	roles := MustSchema("roles", RoleSchema{})
	label := ResultColumn[string]("label")

	query := Select(users.Username.As("label")).
		From(users).
		Where(users.Status.Eq(1)).
		UnionAll(
			Select(roles.Name.As("label")).
				From(roles),
		).
		OrderBy(label.Asc()).
		Limit(5)

	bound, err := query.Build(testSQLiteDialect{})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	expectedSQL := `SELECT "users"."username" AS "label" FROM "users" WHERE "users"."status" = ? UNION ALL SELECT "roles"."name" AS "label" FROM "roles" ORDER BY "label" ASC LIMIT 5`
	if bound.SQL != expectedSQL {
		t.Fatalf("unexpected union sql:\nwant: %s\n got: %s", expectedSQL, bound.SQL)
	}
	if !reflect.DeepEqual(bound.Args, []any{1}) {
		t.Fatalf("unexpected union args: %#v", bound.Args)
	}
}

func TestSelectBuildWithCaseWhen(t *testing.T) {
	users := MustSchema("users", UserSchema{})

	statusLabel := CaseWhen[string](users.Status.Eq(1), "active").
		When(users.Status.Eq(2), "blocked").
		Else("unknown")

	query := Select(
		users.ID,
		statusLabel.As("status_label"),
	).
		From(users).
		Where(statusLabel.Ne("unknown")).
		OrderBy(statusLabel.Asc())

	bound, err := query.Build(testPostgresDialect{})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	expectedSQL := `SELECT "users"."id", CASE WHEN "users"."status" = $1 THEN $2 WHEN "users"."status" = $3 THEN $4 ELSE $5 END AS "status_label" FROM "users" WHERE CASE WHEN "users"."status" = $6 THEN $7 WHEN "users"."status" = $8 THEN $9 ELSE $10 END <> $11 ORDER BY CASE WHEN "users"."status" = $12 THEN $13 WHEN "users"."status" = $14 THEN $15 ELSE $16 END ASC`
	if bound.SQL != expectedSQL {
		t.Fatalf("unexpected case sql:\nwant: %s\n got: %s", expectedSQL, bound.SQL)
	}
	expectedArgs := []any{1, "active", 2, "blocked", "unknown", 1, "active", 2, "blocked", "unknown", "unknown", 1, "active", 2, "blocked", "unknown"}
	if !reflect.DeepEqual(bound.Args, expectedArgs) {
		t.Fatalf("unexpected case args: %#v", bound.Args)
	}
}
