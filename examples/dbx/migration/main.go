// Package main demonstrates dbx schema migration and runner usage.
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

	core, closeDB := openMigrationDB()
	defer closeOrPanic(closeDB)

	printMigrationPlan(planSchemaChanges(ctx, core, catalog))
	printMigrationReport(autoMigrateSchemas(ctx, core, catalog))
	printSchemaValidation(validateSchemas(ctx, core, catalog))
	printForeignKeys(catalog)

	runner := migrate.NewRunner(core.SQLDB(), core.Dialect(), migrate.RunnerOptions{ValidateHash: true})
	printGoMigrationReport(runGoMigrations(ctx, runner))
	printSQLMigrationReport(runSQLMigrations(ctx, runner))
	printAppliedHistory(appliedHistory(ctx, runner))
	printRunnerEventCount(queryRunnerEventCount(ctx, core))
}

func openMigrationDB() (*dbx.DB, func() error) {
	core, closeDB, err := shared.OpenSQLite("dbx-migration", dbx.WithLogger(shared.NewLogger()), dbx.WithDebug(true))
	if err != nil {
		panic(err)
	}

	return core, closeDB
}

func planSchemaChanges(ctx context.Context, core *dbx.DB, catalog shared.Catalog) dbx.MigrationPlan {
	plan, err := core.PlanSchemaChanges(ctx, catalog.Roles, catalog.Users, catalog.UserRoles)
	if err != nil {
		panic(err)
	}

	return plan
}

func printMigrationPlan(plan dbx.MigrationPlan) {
	printLine("planned migration actions:")
	for index := range plan.Actions {
		action := &plan.Actions[index]
		printFormat("- kind=%s executable=%t summary=%s\n", action.Kind, action.Executable, action.Summary)
	}

	printLine("planned sql preview:")
	preview := plan.SQLPreview()
	for index := range preview {
		printFormat("- sql=%s\n", preview[index])
	}
}

func autoMigrateSchemas(ctx context.Context, core *dbx.DB, catalog shared.Catalog) dbx.ValidationReport {
	report, err := core.AutoMigrate(ctx, catalog.Roles, catalog.Users, catalog.UserRoles)
	if err != nil {
		panic(err)
	}

	return report
}

func printMigrationReport(report dbx.ValidationReport) {
	printFormat("auto migrate valid=%t tables=%d\n", report.Valid(), len(report.Tables))
}

func validateSchemas(ctx context.Context, core *dbx.DB, catalog shared.Catalog) dbx.ValidationReport {
	report, err := core.ValidateSchemas(ctx, catalog.Roles, catalog.Users, catalog.UserRoles)
	if err != nil {
		panic(err)
	}

	return report
}

func printSchemaValidation(report dbx.ValidationReport) {
	printFormat("validate valid=%t\n", report.Valid())
}

func printForeignKeys(catalog shared.Catalog) {
	printLine("users foreign keys:")
	foreignKeys := catalog.Users.ForeignKeys()
	for index := range foreignKeys {
		fk := foreignKeys[index]
		printFormat("- name=%s columns=%v target=%s(%v)\n", fk.Name, fk.Columns, fk.TargetTable, fk.TargetColumns)
	}
}

func runGoMigrations(ctx context.Context, runner *migrate.Runner) migrate.RunReport {
	report, err := runner.UpGo(ctx, migrate.NewGoMigration(
		"1",
		"create runner events",
		func(ctx context.Context, tx *sql.Tx) error {
			_, execErr := tx.ExecContext(ctx, `CREATE TABLE runner_events (id INTEGER PRIMARY KEY, message TEXT NOT NULL)`)
			if execErr != nil {
				return fmt.Errorf("create runner_events table: %w", execErr)
			}

			return nil
		},
		nil,
	))
	if err != nil {
		panic(err)
	}

	return report
}

func printGoMigrationReport(report migrate.RunReport) {
	printFormat("go migrations applied=%d\n", len(report.Applied))
}

func runSQLMigrations(ctx context.Context, runner *migrate.Runner) migrate.RunReport {
	source := migrate.FileSource{FS: migrationFS, Dir: "migrations"}
	report, err := runner.UpSQL(ctx, source)
	if err != nil {
		panic(err)
	}

	return report
}

func printSQLMigrationReport(report migrate.RunReport) {
	printFormat("sql migrations applied=%d\n", len(report.Applied))
}

func appliedHistory(ctx context.Context, runner *migrate.Runner) []migrate.AppliedRecord {
	applied, err := runner.Applied(ctx)
	if err != nil {
		panic(err)
	}

	return applied
}

func printAppliedHistory(applied []migrate.AppliedRecord) {
	printLine("applied history:")
	for index := range applied {
		record := &applied[index]
		checksum := truncateChecksum(record.Checksum)
		printFormat("- version=%s kind=%s description=%s checksum=%s\n", record.Version, record.Kind, record.Description, checksum)
	}
}

func truncateChecksum(checksum string) string {
	if len(checksum) > 12 {
		return checksum[:12]
	}

	return checksum
}

func queryRunnerEventCount(ctx context.Context, core *dbx.DB) int {
	row := core.QueryRowContext(ctx, `SELECT COUNT(*) FROM runner_events`)
	var total int
	if err := row.Scan(&total); err != nil {
		panic(err)
	}

	return total
}

func printRunnerEventCount(total int) {
	printFormat("runner_events rows=%d\n", total)
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
