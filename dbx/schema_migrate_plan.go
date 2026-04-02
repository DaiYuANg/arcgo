package dbx

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/lo"
)

func PlanSchemaChanges(ctx context.Context, session Session, schemas ...SchemaResource) (MigrationPlan, error) {
	logRuntimeNode(session, "schema.plan.start", "schemas", len(schemas))
	if plan, ok, err := planSchemaChangesWithAtlas(ctx, session, schemas...); ok || err != nil {
		logPlanSchemaChangesResult(session, "atlas", plan, err)
		return plan, err
	}

	plan, err := planSchemaChangesLegacy(ctx, session, schemas...)
	logPlanSchemaChangesResult(session, "legacy", plan, err)
	return plan, err
}

func logPlanSchemaChangesResult(session Session, backend string, plan MigrationPlan, err error) {
	if err != nil {
		logRuntimeNode(session, "schema.plan.error", "backend", backend, "error", err)
		return
	}
	logRuntimeNode(session, "schema.plan.done", "backend", backend, "actions", plan.Actions.Len(), "manual_actions", plan.HasManualActions())
}

func planSchemaChangesLegacy(ctx context.Context, session Session, schemas ...SchemaResource) (MigrationPlan, error) {
	schemaDialect, err := requireSchemaDialect(session)
	if err != nil {
		return MigrationPlan{}, err
	}

	reportTables := collectionx.NewListWithCapacity[TableDiff](len(schemas))
	actions := collectionx.NewListWithCapacity[MigrationAction](len(schemas))
	for _, schema := range schemas {
		diff, err := planLegacySchemaDiff(ctx, schemaDialect, session, schema)
		if err != nil {
			return MigrationPlan{}, err
		}
		reportTables.Add(diff)
		actions.Add(buildLegacyMigrationActions(schemaDialect, schema, diff)...)
	}

	return MigrationPlan{
		Actions: actions,
		Report: ValidationReport{
			Tables:   reportTables,
			Backend:  ValidationBackendLegacy,
			Complete: false,
			Warnings: collectionx.NewList("dbx: schema validation is running in legacy mode; extra drift may not be reported"),
		},
	}, nil
}

func planLegacySchemaDiff(ctx context.Context, schemaDialect SchemaDialect, session Session, schema SchemaResource) (TableDiff, error) {
	table := schema.tableRef().TableName()
	logRuntimeNode(session, "schema.plan.legacy.diff", "table", table)
	diff, err := diffSchema(ctx, schemaDialect, session, schema)
	if err != nil {
		logRuntimeNode(session, "schema.plan.legacy.error", "table", table, "error", err)
		return TableDiff{}, err
	}
	logLegacyDiffSummary(session, diff)
	return diff, nil
}

func logLegacyDiffSummary(session Session, diff TableDiff) {
	logRuntimeNode(session,
		"schema.plan.legacy.diff_done",
		"table", diff.Table,
		"missing_table", diff.MissingTable,
		"missing_columns", diff.MissingColumns.Len(),
		"missing_indexes", diff.MissingIndexes.Len(),
		"missing_foreign_keys", diff.MissingForeignKeys.Len(),
		"missing_checks", diff.MissingChecks.Len(),
		"column_diffs", diff.ColumnDiffs.Len(),
	)
}

func buildLegacyMigrationActions(schemaDialect SchemaDialect, schema SchemaResource, diff TableDiff) []MigrationAction {
	spec := buildTableSpec(schema.schemaRef())
	if diff.MissingTable {
		return buildMissingTableActions(schemaDialect, spec)
	}
	return buildExistingTableActions(schemaDialect, diff)
}

func buildMissingTableActions(schemaDialect SchemaDialect, spec TableSpec) []MigrationAction {
	return slices.Concat(
		[]MigrationAction{buildCreateTableAction(schemaDialect, spec)},
		mappedMigrationActions(spec.Indexes, func(index IndexMeta) MigrationAction {
			return buildCreateIndexAction(schemaDialect, index)
		}),
	)
}

func buildExistingTableActions(schemaDialect SchemaDialect, diff TableDiff) []MigrationAction {
	return slices.Concat(
		mappedMigrationActionList(diff.MissingColumns, func(column ColumnMeta) MigrationAction {
			return buildAddColumnAction(schemaDialect, diff.Table, column)
		}),
		mappedMigrationActionList(diff.MissingIndexes, func(index IndexMeta) MigrationAction {
			return buildCreateIndexAction(schemaDialect, index)
		}),
		mappedMigrationActionList(diff.MissingForeignKeys, func(foreignKey ForeignKeyMeta) MigrationAction {
			return buildAddForeignKeyAction(schemaDialect, diff.Table, foreignKey)
		}),
		mappedMigrationActionList(diff.MissingChecks, func(check CheckMeta) MigrationAction {
			return buildAddCheckAction(schemaDialect, diff.Table, check)
		}),
		primaryKeyManualActions(diff),
		columnDiffManualActions(diff),
	)
}

func ValidateSchemas(ctx context.Context, session Session, schemas ...SchemaResource) (ValidationReport, error) {
	logRuntimeNode(session, "schema.validate.start", "schemas", len(schemas))
	plan, err := PlanSchemaChanges(ctx, session, schemas...)
	if err != nil {
		logRuntimeNode(session, "schema.validate.error", "error", err)
		return ValidationReport{}, err
	}
	logRuntimeNode(session, "schema.validate.done", "valid", plan.Report.Valid(), "tables", plan.Report.Tables.Len())
	return plan.Report, nil
}

func (r ValidationReport) withWarning(message string) ValidationReport {
	if strings.TrimSpace(message) == "" {
		return r
	}
	next := r
	next.Warnings = r.Warnings.Clone()
	next.Warnings.Add(message)
	return next
}

func (db *DB) PlanSchemaChanges(ctx context.Context, schemas ...SchemaResource) (MigrationPlan, error) {
	return PlanSchemaChanges(ctx, db, schemas...)
}

func (db *DB) ValidateSchemas(ctx context.Context, schemas ...SchemaResource) (ValidationReport, error) {
	return observeSchemaValidation(ctx, db.observe, db, schemas...)
}

func (db *DB) AutoMigrate(ctx context.Context, schemas ...SchemaResource) (ValidationReport, error) {
	return observeSchemaAutoMigrate(ctx, db.observe, db, schemas...)
}

func (tx *Tx) PlanSchemaChanges(ctx context.Context, schemas ...SchemaResource) (MigrationPlan, error) {
	return PlanSchemaChanges(ctx, tx, schemas...)
}

func (tx *Tx) ValidateSchemas(ctx context.Context, schemas ...SchemaResource) (ValidationReport, error) {
	return observeSchemaValidation(ctx, tx.observe, tx, schemas...)
}

func (tx *Tx) AutoMigrate(ctx context.Context, schemas ...SchemaResource) (ValidationReport, error) {
	return observeSchemaAutoMigrate(ctx, tx.observe, tx, schemas...)
}

func observeSchemaValidation(ctx context.Context, observe runtimeObserver, session Session, schemas ...SchemaResource) (ValidationReport, error) {
	ctx, event, err := observe.before(ctx, HookEvent{
		Operation: OperationValidate,
		Table:     schemaNames(schemas),
	})
	if err != nil {
		observe.after(ctx, event)
		return ValidationReport{}, err
	}
	report, validateErr := ValidateSchemas(ctx, session, schemas...)
	event.Err = validateErr
	observe.after(ctx, event)
	return report, validateErr
}

func observeSchemaAutoMigrate(ctx context.Context, observe runtimeObserver, session Session, schemas ...SchemaResource) (ValidationReport, error) {
	ctx, event, err := observe.before(ctx, HookEvent{
		Operation: OperationAutoMigrate,
		Table:     schemaNames(schemas),
	})
	if err != nil {
		observe.after(ctx, event)
		return ValidationReport{}, err
	}
	report, migrateErr := AutoMigrate(ctx, session, schemas...)
	event.Err = migrateErr
	observe.after(ctx, event)
	return report, migrateErr
}

func requireSchemaDialect(session Session) (SchemaDialect, error) {
	if session == nil {
		return nil, ErrNilDB
	}
	if session.Dialect() == nil {
		return nil, ErrNilDialect
	}
	schemaDialect, ok := session.Dialect().(SchemaDialect)
	if !ok {
		return nil, fmt.Errorf("dbx: dialect %T does not implement schema migration support", session.Dialect())
	}
	return schemaDialect, nil
}

func schemaNames(schemas []SchemaResource) string {
	if len(schemas) == 0 {
		return ""
	}
	return strings.Join(lo.Map(schemas, func(schema SchemaResource, _ int) string {
		return schema.tableRef().TableName()
	}), ",")
}

func primaryKeyManualActions(diff TableDiff) []MigrationAction {
	if diff.PrimaryKeyDiff == nil {
		return nil
	}
	return []MigrationAction{{
		Kind:    MigrationActionManual,
		Table:   diff.Table,
		Summary: "manual primary key migration required",
	}}
}
