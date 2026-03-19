package main

import (
	"context"
	"fmt"

	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/DaiYuANg/arcgo/examples/dbx/internal/shared"
	"github.com/samber/mo"
)

type userRoleRow struct {
	ID       int64
	Username string
	RoleName string
}

type userRolePair struct {
	Username string
	RoleName string
}

func main() {
	ctx := context.Background()
	catalog := shared.NewCatalog()

	core, closeDB, err := shared.OpenSQLite("dbx-relations", dbx.WithLogger(shared.NewLogger()), dbx.WithDebug(true))
	if err != nil {
		panic(err)
	}
	defer func() { _ = closeDB() }()

	if _, err := core.AutoMigrate(ctx, catalog.Roles, catalog.Users, catalog.UserRoles); err != nil {
		panic(err)
	}
	if err := shared.SeedDemoData(ctx, core, catalog); err != nil {
		panic(err)
	}

	users := dbx.Alias(catalog.Users, "u")
	roles := dbx.Alias(catalog.Roles, "r")

	belongsToQuery := dbx.Select(users.ID, users.Username, roles.Name).From(users)
	if _, err := belongsToQuery.JoinRelation(users, users.Role, roles); err != nil {
		panic(err)
	}
	belongsToQuery = belongsToQuery.Where(roles.Name.Eq("admin")).OrderBy(users.ID.Asc())

	belongsToBound, err := dbx.Build(core, belongsToQuery)
	if err != nil {
		panic(err)
	}
	fmt.Printf("belongs-to SQL: %s\n", belongsToBound.SQL)

	belongsToRows, err := core.QueryBoundContext(ctx, belongsToBound)
	if err != nil {
		panic(err)
	}
	defer belongsToRows.Close()

	fmt.Println("users with role=admin:")
	for belongsToRows.Next() {
		var row userRoleRow
		if err := belongsToRows.Scan(&row.ID, &row.Username, &row.RoleName); err != nil {
			panic(err)
		}
		fmt.Printf("- id=%d username=%s role=%s\n", row.ID, row.Username, row.RoleName)
	}
	if err := belongsToRows.Err(); err != nil {
		panic(err)
	}

	users = dbx.Alias(catalog.Users, "u")
	roles = dbx.Alias(catalog.Roles, "r")
	manyToManyQuery := dbx.Select(users.Username, roles.Name).From(users)
	if _, err := manyToManyQuery.JoinRelation(users, users.Roles, roles); err != nil {
		panic(err)
	}
	manyToManyQuery = manyToManyQuery.Where(users.Username.Eq("alice")).OrderBy(roles.Name.Asc())

	manyToManyBound, err := dbx.Build(core, manyToManyQuery)
	if err != nil {
		panic(err)
	}
	fmt.Printf("many-to-many SQL: %s\n", manyToManyBound.SQL)

	manyToManyRows, err := core.QueryBoundContext(ctx, manyToManyBound)
	if err != nil {
		panic(err)
	}
	defer manyToManyRows.Close()

	fmt.Println("alice roles:")
	for manyToManyRows.Next() {
		var row userRolePair
		if err := manyToManyRows.Scan(&row.Username, &row.RoleName); err != nil {
			panic(err)
		}
		fmt.Printf("- username=%s role=%s\n", row.Username, row.RoleName)
	}
	if err := manyToManyRows.Err(); err != nil {
		panic(err)
	}

	fmt.Println("relation loaders:")
	userMapper := dbx.MustMapper[shared.User](catalog.Users)
	roleMapper := dbx.MustMapper[shared.Role](catalog.Roles)
	usersToLoad, err := dbx.QueryAll(
		ctx,
		core,
		dbx.Select(catalog.Users.AllColumns()...).From(catalog.Users).OrderBy(catalog.Users.ID.Asc()),
		userMapper,
	)
	if err != nil {
		panic(err)
	}

	loadedRole := make([]mo.Option[shared.Role], len(usersToLoad))
	if err := dbx.LoadBelongsTo(
		ctx,
		core,
		usersToLoad,
		catalog.Users,
		userMapper,
		catalog.Users.Role,
		catalog.Roles,
		roleMapper,
		func(index int, _ *shared.User, value mo.Option[shared.Role]) {
			loadedRole[index] = value
		},
	); err != nil {
		panic(err)
	}

	loadedRoles := make([][]shared.Role, len(usersToLoad))
	if err := dbx.LoadManyToMany(
		ctx,
		core,
		usersToLoad,
		catalog.Users,
		userMapper,
		catalog.Users.Roles,
		catalog.Roles,
		roleMapper,
		func(index int, _ *shared.User, value []shared.Role) {
			loadedRoles[index] = value
		},
	); err != nil {
		panic(err)
	}

	for index, user := range usersToLoad {
		roleName := "<none>"
		if loadedRole[index].IsPresent() {
			role, _ := loadedRole[index].Get()
			roleName = role.Name
		}
		fmt.Printf("- user=%s belongs-to role=%s many-to-many roles=", user.Username, roleName)
		for roleIndex, role := range loadedRoles[index] {
			if roleIndex > 0 {
				fmt.Print(",")
			}
			fmt.Print(role.Name)
		}
		fmt.Println()
	}
}
