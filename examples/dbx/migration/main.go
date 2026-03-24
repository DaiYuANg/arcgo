package main

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/DaiYuANg/arcgo/dbx/migrate"
	"github.com/DaiYuANg/arcgo/examples/dbx/internal/shared"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

func main() {
	ctx := context.Background()
	catalog := shared.NewCatalog()

	core, closeDB, err := shared.OpenSQLite("dbx-migration", dbx.WithLogger(shared.NewLogger()), dbx.WithDebug(true))
	if err != nil {
		panic(err)
	}
	defer func() { _ = closeDB() }()

	plan, err := core.PlanSchemaChanges(ctx, catalog.Roles, catalog.Users, catalog.UserRoles)
	if err != nil {
		panic(err)
	}

	fmt.Println("planned migration actions:")
	for _, action := range plan.Actions {
		fmt.Printf("- kind=%s executable=%t summary=%s\n", action.Kind, action.Executable, action.Summary)
	}

	fmt.Println("planned sql preview:")
	for _, sqlText := range plan.SQLPreview() {
		fmt.Printf("- sql=%s\n", sqlText)
	}

	report, err := core.AutoMigrate(ctx, catalog.Roles, catalog.Users, catalog.UserRoles)
	if err != nil {
		panic(err)
	}
	fmt.Printf("auto migrate valid=%t tables=%d\n", report.Valid(), len(report.Tables))

	validated, err := core.ValidateSchemas(ctx, catalog.Roles, catalog.Users, catalog.UserRoles)
	if err != nil {
		panic(err)
	}
	fmt.Printf("validate valid=%t\n", validated.Valid())

	fmt.Println("users foreign keys:")
	for _, fk := range catalog.Users.ForeignKeys() {
		fmt.Printf("- name=%s columns=%v target=%s(%v)\n", fk.Name, fk.Columns, fk.TargetTable, fk.TargetColumns)
	}

	runner := migrate.NewRunner(core.SQLDB(), core.Dialect(), migrate.RunnerOptions{ValidateHash: true})
	goReport, err := runner.UpGo(ctx, migrate.NewGoMigration(
		"1",
		"create runner events",
		func(ctx context.Context, tx *sql.Tx) error {
			_, execErr := tx.ExecContext(ctx, `CREATE TABLE runner_events (id INTEGER PRIMARY KEY, message TEXT NOT NULL)`)
			return execErr
		},
		nil,
	))
	if err != nil {
		panic(err)
	}
	fmt.Printf("go migrations applied=%d\n", len(goReport.Applied))

	source := migrate.FileSource{FS: migrationFS, Dir: "migrations"}
	sqlReport, err := runner.UpSQL(ctx, source)
	if err != nil {
		panic(err)
	}
	fmt.Printf("sql migrations applied=%d\n", len(sqlReport.Applied))

	applied, err := runner.Applied(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println("applied history:")
	for _, record := range applied {
		checksum := record.Checksum
		if len(checksum) > 12 {
			checksum = checksum[:12]
		}
		fmt.Printf("- version=%s kind=%s description=%s checksum=%s\n", record.Version, record.Kind, record.Description, checksum)
	}

	row := core.QueryRowContext(ctx, `SELECT COUNT(*) FROM runner_events`)
	var total int
	if err := row.Scan(&total); err != nil {
		panic(err)
	}
	fmt.Printf("runner_events rows=%d\n", total)
}
