// Package main demonstrates dbx mutation and returning-query patterns.
package main

import (
	"context"
	"fmt"

	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/DaiYuANg/arcgo/examples/dbx/internal/shared"
)

type statusSummary struct {
	Status    int   `dbx:"status"`
	UserCount int64 `dbx:"user_count"`
}

type userNameRow struct {
	Username string `dbx:"username"`
}

type userArchive struct {
	ID       int64  `dbx:"id"`
	Username string `dbx:"username"`
	Status   int    `dbx:"status"`
}

type userArchiveSchema struct {
	dbx.Schema[userArchive]
	ID       dbx.Column[userArchive, int64]  `dbx:"id,pk,auto"`
	Username dbx.Column[userArchive, string] `dbx:"username,unique"`
	Status   dbx.Column[userArchive, int]    `dbx:"status"`
}

func main() {
	ctx := context.Background()
	catalog := shared.NewCatalog()
	archive := dbx.MustSchema("user_archive", userArchiveSchema{})

	core, closeDB := openMutationDB()
	defer closeOrPanic(closeDB)

	prepareMutationData(ctx, core, catalog, archive)

	printStatusSummaries(queryStatusSummaries(ctx, core, catalog))
	printUserNameRows("users resolved by subquery + exists:", queryAdminUsers(ctx, core, catalog))

	archiveMapper := dbx.MustMapper[userArchive](archive)
	printArchiveRows("insert-select returning:", insertArchiveFromSelect(ctx, core, catalog, archive, archiveMapper))
	printArchiveRows("batch insert returning:", batchInsertArchive(ctx, core, archive, archiveMapper))
	printArchiveRows("upsert returning:", upsertArchive(ctx, core, archive, archiveMapper))
}

func openMutationDB() (*dbx.DB, func() error) {
	core, closeDB, err := shared.OpenSQLite(
		"dbx-mutation",
		dbx.WithLogger(shared.NewLogger()),
		dbx.WithDebug(true),
	)
	if err != nil {
		panic(err)
	}

	return core, closeDB
}

func prepareMutationData(ctx context.Context, core *dbx.DB, catalog shared.Catalog, archive userArchiveSchema) {
	_, err := core.AutoMigrate(ctx, catalog.Roles, catalog.Users, catalog.UserRoles, archive)
	if err != nil {
		panic(err)
	}
	err = shared.SeedDemoData(ctx, core, catalog)
	if err != nil {
		panic(err)
	}
}

func queryStatusSummaries(ctx context.Context, core *dbx.DB, catalog shared.Catalog) []statusSummary {
	rows, err := dbx.QueryAll[statusSummary](
		ctx,
		core,
		dbx.Select(
			catalog.Users.Status,
			dbx.CountAll().As("user_count"),
		).
			From(catalog.Users).
			GroupBy(catalog.Users.Status).
			Having(dbx.CountAll().Gt(int64(0))).
			OrderBy(catalog.Users.Status.Asc()),
		dbx.MustStructMapper[statusSummary](),
	)
	if err != nil {
		panic(err)
	}

	return rows
}

func queryAdminUsers(ctx context.Context, core *dbx.DB, catalog shared.Catalog) []userNameRow {
	adminRoleIDs := dbx.Select(catalog.Roles.ID).
		From(catalog.Roles).
		Where(catalog.Roles.Name.Eq("admin"))

	rows, err := dbx.QueryAll[userNameRow](
		ctx,
		core,
		dbx.Select(catalog.Users.Username).
			From(catalog.Users).
			Where(dbx.And(
				catalog.Users.RoleID.InQuery(adminRoleIDs),
				dbx.Exists(
					dbx.Select(catalog.UserRoles.UserID).
						From(catalog.UserRoles).
						Where(catalog.UserRoles.UserID.EqColumn(catalog.Users.ID)).
						Limit(1),
				),
			)),
		dbx.MustStructMapper[userNameRow](),
	)
	if err != nil {
		panic(err)
	}

	return rows
}

func insertArchiveFromSelect(
	ctx context.Context,
	core *dbx.DB,
	catalog shared.Catalog,
	archive userArchiveSchema,
	archiveMapper dbx.Mapper[userArchive],
) []userArchive {
	rows, err := dbx.QueryAll[userArchive](
		ctx,
		core,
		dbx.InsertInto(archive).
			Columns(archive.Username, archive.Status).
			FromSelect(
				dbx.Select(catalog.Users.Username, catalog.Users.Status).
					From(catalog.Users).
					Where(catalog.Users.Status.Eq(1)).
					OrderBy(catalog.Users.ID.Asc()),
			).
			Returning(archive.ID, archive.Username, archive.Status),
		archiveMapper,
	)
	if err != nil {
		panic(err)
	}

	return rows
}

func batchInsertArchive(
	ctx context.Context,
	core *dbx.DB,
	archive userArchiveSchema,
	archiveMapper dbx.Mapper[userArchive],
) []userArchive {
	rows, err := dbx.QueryAll[userArchive](
		ctx,
		core,
		dbx.InsertInto(archive).
			Values(
				archive.Username.Set("eve"),
				archive.Status.Set(1),
			).
			Values(
				archive.Username.Set("mallory"),
				archive.Status.Set(0),
			).
			Returning(archive.ID, archive.Username, archive.Status),
		archiveMapper,
	)
	if err != nil {
		panic(err)
	}

	return rows
}

func upsertArchive(
	ctx context.Context,
	core *dbx.DB,
	archive userArchiveSchema,
	archiveMapper dbx.Mapper[userArchive],
) []userArchive {
	rows, err := dbx.QueryAll[userArchive](
		ctx,
		core,
		dbx.InsertInto(archive).
			Values(
				archive.Username.Set("alice"),
				archive.Status.Set(9),
			).
			OnConflict(archive.Username).
			DoUpdateSet(archive.Status.SetExcluded()).
			Returning(archive.ID, archive.Username, archive.Status),
		archiveMapper,
	)
	if err != nil {
		panic(err)
	}

	return rows
}

func printStatusSummaries(rows []statusSummary) {
	printLine("aggregate status counts:")
	for index := range rows {
		row := &rows[index]
		printFormat("- status=%d count=%d\n", row.Status, row.UserCount)
	}
}

func printUserNameRows(title string, rows []userNameRow) {
	printLine(title)
	for index := range rows {
		row := &rows[index]
		printFormat("- username=%s\n", row.Username)
	}
}

func printArchiveRows(title string, rows []userArchive) {
	printLine(title)
	for index := range rows {
		row := &rows[index]
		printFormat("- id=%d username=%s status=%d\n", row.ID, row.Username, row.Status)
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
