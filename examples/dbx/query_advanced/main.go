// Package main demonstrates advanced dbx query composition examples.
package main

import (
	"context"
	"fmt"

	"github.com/DaiYuANg/arcgo/collectionx"
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
	defer closeOrPanic(closeDB)

	_, err = core.AutoMigrate(ctx, catalog.Roles, catalog.Users, catalog.UserRoles)
	if err != nil {
		panic(err)
	}
	err = shared.SeedDemoData(ctx, core, catalog)
	if err != nil {
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

	activeRows, err := dbx.QueryAll[activeUserRow](ctx, core, activeQuery, dbx.MustStructMapper[activeUserRow]())
	if err != nil {
		panic(err)
	}
	printActiveRows(activeRows)

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

	labeledRows, err := dbx.QueryAll[labeledUserRow](ctx, core, labeledQuery, dbx.MustStructMapper[labeledUserRow]())
	if err != nil {
		panic(err)
	}
	printLabeledRows(labeledRows)

	label := dbx.ResultColumn[string]("label")
	unionQuery := dbx.Select(catalog.Users.Username.As("label")).
		From(catalog.Users).
		Where(catalog.Users.Status.Eq(1)).
		UnionAll(
			dbx.Select(catalog.Roles.Name.As("label")).
				From(catalog.Roles),
		).
		OrderBy(label.Asc())

	unionRows, err := dbx.QueryAll[unionLabelRow](ctx, core, unionQuery, dbx.MustStructMapper[unionLabelRow]())
	if err != nil {
		panic(err)
	}
	printUnionRows(unionRows)
}

func printActiveRows(rows collectionx.List[activeUserRow]) {
	printLine("cte query:")
	rows.Range(func(_ int, row activeUserRow) bool {
		printFormat("- id=%d username=%s\n", row.ID, row.Username)
		return true
	})
}

func printLabeledRows(rows collectionx.List[labeledUserRow]) {
	printLine("case query:")
	rows.Range(func(_ int, row labeledUserRow) bool {
		printFormat("- id=%d username=%s status=%s\n", row.ID, row.Username, row.StatusLabel)
		return true
	})
}

func printUnionRows(rows collectionx.List[unionLabelRow]) {
	printLine("union query:")
	rows.Range(func(_ int, row unionLabelRow) bool {
		printFormat("- label=%s\n", row.Label)
		return true
	})
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
