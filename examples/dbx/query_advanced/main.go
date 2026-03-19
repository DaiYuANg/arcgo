package main

import (
	"context"
	"fmt"

	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/DaiYuANg/arcgo/examples/dbx/internal/shared"
)

type activeUserRow struct {
	ID       int64  `dbx:"id"`
	Username string `dbx:"username"`
}

type labeledUserRow struct {
	ID          int64  `dbx:"id"`
	Username    string `dbx:"username"`
	StatusLabel string `dbx:"status_label"`
}

type unionLabelRow struct {
	Label string `dbx:"label"`
}

func main() {
	ctx := context.Background()
	catalog := shared.NewCatalog()

	core, closeDB, err := shared.OpenSQLite("dbx-query-advanced", dbx.WithLogger(shared.NewLogger()), dbx.WithDebug(true))
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

	activeUsers := dbx.NamedTable("active_users")
	activeID := dbx.NamedColumn[int64](activeUsers, "id")
	activeUsername := dbx.NamedColumn[string](activeUsers, "username")
	activeQuery := dbx.Select(activeID, activeUsername).
		With("active_users",
			dbx.Select(catalog.Users.ID, catalog.Users.Username).
				From(catalog.Users).
				Where(catalog.Users.Status.Eq(1)),
		).
		From(activeUsers).
		OrderBy(activeID.Asc())

	activeRows, err := dbx.QueryAll(ctx, core, activeQuery, dbx.MustStructMapper[activeUserRow]())
	if err != nil {
		panic(err)
	}
	fmt.Println("cte query:")
	for _, row := range activeRows {
		fmt.Printf("- id=%d username=%s\n", row.ID, row.Username)
	}

	statusLabel := dbx.CaseWhen[string](catalog.Users.Status.Eq(1), "active").
		When(catalog.Users.Status.Eq(0), "inactive").
		Else("unknown")
	labeledQuery := dbx.Select(
		catalog.Users.ID,
		catalog.Users.Username,
		statusLabel.As("status_label"),
	).
		From(catalog.Users).
		OrderBy(catalog.Users.ID.Asc())

	labeledRows, err := dbx.QueryAll(ctx, core, labeledQuery, dbx.MustStructMapper[labeledUserRow]())
	if err != nil {
		panic(err)
	}
	fmt.Println("case query:")
	for _, row := range labeledRows {
		fmt.Printf("- id=%d username=%s status=%s\n", row.ID, row.Username, row.StatusLabel)
	}

	label := dbx.ResultColumn[string]("label")
	unionQuery := dbx.Select(catalog.Users.Username.As("label")).
		From(catalog.Users).
		Where(catalog.Users.Status.Eq(1)).
		UnionAll(
			dbx.Select(catalog.Roles.Name.As("label")).
				From(catalog.Roles),
		).
		OrderBy(label.Asc())

	unionRows, err := dbx.QueryAll(ctx, core, unionQuery, dbx.MustStructMapper[unionLabelRow]())
	if err != nil {
		panic(err)
	}
	fmt.Println("union query:")
	for _, row := range unionRows {
		fmt.Printf("- label=%s\n", row.Label)
	}
}
