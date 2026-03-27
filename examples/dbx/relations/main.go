// Package main demonstrates dbx relation joins and relation loaders.
package main

import (
	"context"
	"fmt"
	"strings"

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

	core, closeDB := openRelationsDB()
	defer closeOrPanic(closeDB)

	prepareRelationsData(ctx, core, catalog)

	belongsToSQL, belongsToRows := runBelongsToExample(ctx, core, catalog)
	printFormat("belongs-to SQL: %s\n", belongsToSQL)
	printUserRoleRows("users with role=admin:", belongsToRows)

	manyToManySQL, manyToManyRows := runManyToManyExample(ctx, core, catalog)
	printFormat("many-to-many SQL: %s\n", manyToManySQL)
	printUserRolePairs("alice roles:", manyToManyRows)

	printLoadedRelations("relation loaders:", loadRelations(ctx, core, catalog))
}

type loadedUserRelations struct {
	Username      string
	BelongsToRole string
	ManyToMany    []string
}

func openRelationsDB() (*dbx.DB, func() error) {
	core, closeDB, err := shared.OpenSQLite("dbx-relations", dbx.WithLogger(shared.NewLogger()), dbx.WithDebug(true))
	if err != nil {
		panic(err)
	}

	return core, closeDB
}

func prepareRelationsData(ctx context.Context, core *dbx.DB, catalog shared.Catalog) {
	_, err := core.AutoMigrate(ctx, catalog.Roles, catalog.Users, catalog.UserRoles)
	if err != nil {
		panic(err)
	}
	err = shared.SeedDemoData(ctx, core, catalog)
	if err != nil {
		panic(err)
	}
}

func runBelongsToExample(ctx context.Context, core *dbx.DB, catalog shared.Catalog) (string, []userRoleRow) {
	users := dbx.Alias(catalog.Users, "u")
	roles := dbx.Alias(catalog.Roles, "r")

	query := dbx.Select(users.ID, users.Username, roles.Name).From(users)
	_, err := query.JoinRelation(users, users.Role, roles)
	if err != nil {
		panic(err)
	}
	query = query.Where(roles.Name.Eq("admin")).OrderBy(users.ID.Asc())

	bound, err := dbx.Build(core, query)
	if err != nil {
		panic(err)
	}

	return bound.SQL, scanUserRoleRows(ctx, core, bound)
}

func runManyToManyExample(ctx context.Context, core *dbx.DB, catalog shared.Catalog) (string, []userRolePair) {
	users := dbx.Alias(catalog.Users, "u")
	roles := dbx.Alias(catalog.Roles, "r")

	query := dbx.Select(users.Username, roles.Name).From(users)
	_, err := query.JoinRelation(users, users.Roles, roles)
	if err != nil {
		panic(err)
	}
	query = query.Where(users.Username.Eq("alice")).OrderBy(roles.Name.Asc())

	bound, err := dbx.Build(core, query)
	if err != nil {
		panic(err)
	}

	return bound.SQL, scanUserRolePairs(ctx, core, bound)
}

func scanUserRoleRows(ctx context.Context, core *dbx.DB, bound dbx.BoundQuery) []userRoleRow {
	rows, err := core.QueryBoundContext(ctx, bound)
	if err != nil {
		panic(err)
	}
	defer closeRowsOrPanic(rows.Close)

	var out []userRoleRow
	for rows.Next() {
		var row userRoleRow
		err = rows.Scan(&row.ID, &row.Username, &row.RoleName)
		if err != nil {
			panic(err)
		}
		out = append(out, row)
	}
	err = rows.Err()
	if err != nil {
		panic(err)
	}

	return out
}

func scanUserRolePairs(ctx context.Context, core *dbx.DB, bound dbx.BoundQuery) []userRolePair {
	rows, err := core.QueryBoundContext(ctx, bound)
	if err != nil {
		panic(err)
	}
	defer closeRowsOrPanic(rows.Close)

	var out []userRolePair
	for rows.Next() {
		var row userRolePair
		err = rows.Scan(&row.Username, &row.RoleName)
		if err != nil {
			panic(err)
		}
		out = append(out, row)
	}
	err = rows.Err()
	if err != nil {
		panic(err)
	}

	return out
}

func loadRelations(ctx context.Context, core *dbx.DB, catalog shared.Catalog) []loadedUserRelations {
	userMapper := dbx.MustMapper[shared.User](catalog.Users)
	roleMapper := dbx.MustMapper[shared.Role](catalog.Roles)
	usersToLoad, err := dbx.QueryAll[shared.User](
		ctx,
		core,
		dbx.Select(catalog.Users.AllColumns()...).From(catalog.Users).OrderBy(catalog.Users.ID.Asc()),
		userMapper,
	)
	if err != nil {
		panic(err)
	}

	loadedRole := make([]mo.Option[shared.Role], len(usersToLoad))
	err = dbx.LoadBelongsTo(
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
	)
	if err != nil {
		panic(err)
	}

	loadedRoles := make([][]shared.Role, len(usersToLoad))
	err = dbx.LoadManyToMany(
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
	)
	if err != nil {
		panic(err)
	}

	results := make([]loadedUserRelations, 0, len(usersToLoad))
	for index := range usersToLoad {
		user := &usersToLoad[index]
		results = append(results, loadedUserRelations{
			Username:      user.Username,
			BelongsToRole: optionRoleName(loadedRole[index]),
			ManyToMany:    roleNames(loadedRoles[index]),
		})
	}

	return results
}

func optionRoleName(value mo.Option[shared.Role]) string {
	if value.IsPresent() {
		role, _ := value.Get()
		return role.Name
	}

	return "<none>"
}

func roleNames(roles []shared.Role) []string {
	names := make([]string, 0, len(roles))
	for index := range roles {
		names = append(names, roles[index].Name)
	}

	return names
}

func printUserRoleRows(title string, rows []userRoleRow) {
	printLine(title)
	for index := range rows {
		row := &rows[index]
		printFormat("- id=%d username=%s role=%s\n", row.ID, row.Username, row.RoleName)
	}
}

func printUserRolePairs(title string, rows []userRolePair) {
	printLine(title)
	for index := range rows {
		row := &rows[index]
		printFormat("- username=%s role=%s\n", row.Username, row.RoleName)
	}
}

func printLoadedRelations(title string, rows []loadedUserRelations) {
	printLine(title)
	for index := range rows {
		row := &rows[index]
		printFormat(
			"- user=%s belongs-to role=%s many-to-many roles=%s\n",
			row.Username,
			row.BelongsToRole,
			strings.Join(row.ManyToMany, ","),
		)
	}
}

func closeRowsOrPanic(closeFn func() error) {
	if err := closeFn(); err != nil {
		panic(err)
	}
}

func closeOrPanic(closeFn func() error) {
	if err := closeFn(); err != nil {
		panic(err)
	}
}

func printLine(text string) {
	if _, err := fmt.Println(text); err != nil {
		panic(err)
	}
}

func printFormat(format string, args ...any) {
	if _, err := fmt.Printf(format, args...); err != nil {
		panic(err)
	}
}
