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

	core, closeDB, err := shared.OpenSQLite(
		"dbx-mutation",
		dbx.WithLogger(shared.NewLogger()),
		dbx.WithDebug(true),
	)
	if err != nil {
		panic(err)
	}
	defer func() { _ = closeDB() }()

	if _, err := core.AutoMigrate(ctx, catalog.Roles, catalog.Users, catalog.UserRoles, archive); err != nil {
		panic(err)
	}
	if err := shared.SeedDemoData(ctx, core, catalog); err != nil {
		panic(err)
	}

	aggregateRows, err := dbx.QueryAll(
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
	fmt.Println("aggregate status counts:")
	for _, row := range aggregateRows {
		fmt.Printf("- status=%d count=%d\n", row.Status, row.UserCount)
	}

	adminRoleIDs := dbx.Select(catalog.Roles.ID).
		From(catalog.Roles).
		Where(catalog.Roles.Name.Eq("admin"))

	adminUsers, err := dbx.QueryAll(
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
	fmt.Println("users resolved by subquery + exists:")
	for _, row := range adminUsers {
		fmt.Printf("- username=%s\n", row.Username)
	}

	archiveMapper := dbx.MustMapper[userArchive](archive)

	insertedFromSelect, err := dbx.QueryAll(
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
	fmt.Println("insert-select returning:")
	for _, row := range insertedFromSelect {
		fmt.Printf("- id=%d username=%s status=%d\n", row.ID, row.Username, row.Status)
	}

	batchInserted, err := dbx.QueryAll(
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
	fmt.Println("batch insert returning:")
	for _, row := range batchInserted {
		fmt.Printf("- id=%d username=%s status=%d\n", row.ID, row.Username, row.Status)
	}

	upserted, err := dbx.QueryAll(
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
	fmt.Println("upsert returning:")
	for _, row := range upserted {
		fmt.Printf("- id=%d username=%s status=%d\n", row.ID, row.Username, row.Status)
	}
}
