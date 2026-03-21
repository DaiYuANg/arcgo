package main

import (
	"context"
	"embed"
	"fmt"

	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/DaiYuANg/arcgo/dbx/sqltmplx"
	"github.com/DaiYuANg/arcgo/examples/dbx/internal/shared"
)

//go:embed sql/**/*.sql
var sqlFS embed.FS

func main() {
	ctx := context.Background()
	catalog := shared.NewCatalog()

	core, closeDB, err := shared.OpenSQLite(
		"dbx-pure-sql",
		dbx.WithLogger(shared.NewLogger()),
		dbx.WithDebug(true),
	)
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

	registry := sqltmplx.NewRegistry(sqlFS, core.Dialect())

	activeUsers, err := dbx.SQLList[shared.UserSummary](
		ctx,
		core,
		registry.MustStatement("sql/user/find_active.sql"),
		struct {
			Status int `dbx:"status"`
		}{Status: 1},
		dbx.MustStructMapper[shared.UserSummary](),
	)
	if err != nil {
		panic(err)
	}

	fmt.Println("active users from pure sql:")
	for _, user := range activeUsers {
		fmt.Printf("- id=%d username=%s email=%s\n", user.ID, user.Username, user.Email)
	}

	total, err := dbx.SQLScalar[int64](
		ctx,
		core,
		registry.MustStatement("sql/user/count_by_status.sql"),
		struct {
			Status int `dbx:"status"`
		}{Status: 1},
	)
	if err != nil {
		panic(err)
	}
	fmt.Printf("active user count: %d\n", total)

	tx, err := core.BeginTx(ctx, nil)
	if err != nil {
		panic(err)
	}
	if _, err := tx.SQL().Exec(
		ctx,
		registry.MustStatement("sql/user/update_status.sql"),
		struct {
			Status   int    `dbx:"status"`
			Username string `dbx:"username"`
		}{
			Status:   2,
			Username: "bob",
		},
	); err != nil {
		_ = tx.Rollback()
		panic(err)
	}
	if err := tx.Commit(); err != nil {
		panic(err)
	}

	updated, err := dbx.SQLGet[shared.User](
		ctx,
		core,
		registry.MustStatement("sql/user/find_by_username.sql"),
		struct {
			Username string `dbx:"username"`
		}{Username: "bob"},
		dbx.MustStructMapper[shared.User](),
	)
	if err != nil {
		panic(err)
	}
	fmt.Printf("bob status after pure sql update: %d\n", updated.Status)
}
