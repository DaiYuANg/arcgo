package dbx

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dbx/dialect"
	"github.com/samber/lo"
)

type SchemaResource interface {
	TableSource
	schemaRef() schemaDefinition
}

type SchemaDialect interface {
	dialect.Dialect
	BuildCreateTable(spec TableSpec) (BoundQuery, error)
	BuildAddColumn(table string, column ColumnMeta) (BoundQuery, error)
	BuildCreateIndex(index IndexMeta) (BoundQuery, error)
	BuildAddForeignKey(table string, foreignKey ForeignKeyMeta) (BoundQuery, error)
	BuildAddCheck(table string, check CheckMeta) (BoundQuery, error)
	InspectTable(ctx context.Context, executor Executor, table string) (TableState, error)
	NormalizeType(value string) string
}

type TableSpec struct {
	Name        string
	Columns     []ColumnMeta
	Indexes     []IndexMeta
	PrimaryKey  *PrimaryKeyMeta
	ForeignKeys []ForeignKeyMeta
	Checks      []CheckMeta
}

type IndexMeta struct {
	Name    string
	Table   string
	Columns []string
	Unique  bool
}

type TableState struct {
	Exists      bool
	Name        string
	Columns     []ColumnState
	Indexes     []IndexState
	PrimaryKey  *PrimaryKeyState
	ForeignKeys []ForeignKeyState
	Checks      []CheckState
}

type ColumnState struct {
	Name          string
	Type          string
	Nullable      bool
	PrimaryKey    bool
	AutoIncrement bool
	DefaultValue  string
}

type IndexState struct {
	Name    string
	Columns []string
	Unique  bool
}

type PrimaryKeyState struct {
	Name    string
	Columns []string
}

type ForeignKeyState struct {
	Name          string
	Columns       []string
	TargetTable   string
	TargetColumns []string
	OnDelete      ReferentialAction
	OnUpdate      ReferentialAction
}

type CheckState struct {
	Name       string
	Expression string
}

type ValidationReport struct {
	Tables []TableDiff
}

type TableDiff struct {
	Table              string
	MissingTable       bool
	MissingColumns     []ColumnMeta
	MissingIndexes     []IndexMeta
	MissingForeignKeys []ForeignKeyMeta
	MissingChecks      []CheckMeta
	PrimaryKeyDiff     *PrimaryKeyDiff
	ColumnDiffs        []ColumnDiff
}

type PrimaryKeyDiff struct {
	Expected *PrimaryKeyMeta
	Actual   *PrimaryKeyState
	Issues   []string
}

type ColumnDiff struct {
	Column ColumnMeta
	Issues []string
}

type MigrationActionKind string

const (
	MigrationActionCreateTable   MigrationActionKind = "create_table"
	MigrationActionAddColumn     MigrationActionKind = "add_column"
	MigrationActionCreateIndex   MigrationActionKind = "create_index"
	MigrationActionAddForeignKey MigrationActionKind = "add_foreign_key"
	MigrationActionAddCheck      MigrationActionKind = "add_check"
	MigrationActionManual        MigrationActionKind = "manual"
)

type MigrationAction struct {
	Kind       MigrationActionKind
	Table      string
	Summary    string
	Statement  BoundQuery
	Executable bool
}

type MigrationPlan struct {
	Actions []MigrationAction
	Report  ValidationReport
}

type SchemaDriftError struct {
	Report ValidationReport
}

func (e SchemaDriftError) Error() string {
	tables := lo.FilterMap(e.Report.Tables, func(table TableDiff, _ int) (string, bool) {
		return table.Table, !table.Empty()
	})
	if len(tables) == 0 {
		return "dbx: schema drift detected"
	}
	return "dbx: schema drift detected for tables: " + strings.Join(tables, ", ")
}

func (r ValidationReport) Valid() bool {
	return !lo.SomeBy(r.Tables, func(table TableDiff) bool {
		return !table.Empty()
	})
}

func (t TableDiff) Empty() bool {
	return !t.MissingTable &&
		len(t.MissingColumns) == 0 &&
		len(t.MissingIndexes) == 0 &&
		len(t.MissingForeignKeys) == 0 &&
		len(t.MissingChecks) == 0 &&
		t.PrimaryKeyDiff == nil &&
		len(t.ColumnDiffs) == 0
}

func (p MigrationPlan) ExecutableActions() []MigrationAction {
	return lo.Filter(p.Actions, func(action MigrationAction, _ int) bool {
		return action.Executable
	})
}

func (p MigrationPlan) HasManualActions() bool {
	return lo.SomeBy(p.Actions, func(action MigrationAction) bool {
		return !action.Executable
	})
}

func PlanSchemaChanges(ctx context.Context, session Session, schemas ...SchemaResource) (MigrationPlan, error) {
	if plan, ok, err := planSchemaChangesWithAtlas(ctx, session, schemas...); ok || err != nil {
		return plan, err
	}
	return planSchemaChangesLegacy(ctx, session, schemas...)
}

func planSchemaChangesLegacy(ctx context.Context, session Session, schemas ...SchemaResource) (MigrationPlan, error) {
	schemaDialect, err := requireSchemaDialect(session)
	if err != nil {
		return MigrationPlan{}, err
	}

	reportTables := collectionx.NewListWithCapacity[TableDiff](len(schemas))
	actions := collectionx.NewListWithCapacity[MigrationAction](len(schemas))
	for _, schema := range schemas {
		diff, diffErr := diffSchema(ctx, schemaDialect, session, schema)
		if diffErr != nil {
			return MigrationPlan{}, diffErr
		}
		reportTables.Add(diff)

		spec := buildTableSpec(schema.schemaRef())
		if diff.MissingTable {
			actions.Add(buildCreateTableAction(schemaDialect, spec))
			for _, index := range spec.Indexes {
				actions.Add(buildCreateIndexAction(schemaDialect, index))
			}
			continue
		}

		for _, column := range diff.MissingColumns {
			actions.Add(buildAddColumnAction(schemaDialect, diff.Table, column))
		}
		for _, index := range diff.MissingIndexes {
			actions.Add(buildCreateIndexAction(schemaDialect, index))
		}
		for _, foreignKey := range diff.MissingForeignKeys {
			actions.Add(buildAddForeignKeyAction(schemaDialect, diff.Table, foreignKey))
		}
		for _, check := range diff.MissingChecks {
			actions.Add(buildAddCheckAction(schemaDialect, diff.Table, check))
		}
		if diff.PrimaryKeyDiff != nil {
			actions.Add(MigrationAction{
				Kind:    MigrationActionManual,
				Table:   diff.Table,
				Summary: "manual primary key migration required",
			})
		}
		for _, columnDiff := range diff.ColumnDiffs {
			actions.Add(MigrationAction{
				Kind:    MigrationActionManual,
				Table:   diff.Table,
				Summary: "manual column migration required for " + columnDiff.Column.Name + ": " + strings.Join(columnDiff.Issues, "; "),
			})
		}
	}

	return MigrationPlan{
		Actions: actions.Values(),
		Report: ValidationReport{
			Tables: reportTables.Values(),
		},
	}, nil
}

func ValidateSchemas(ctx context.Context, session Session, schemas ...SchemaResource) (ValidationReport, error) {
	plan, err := PlanSchemaChanges(ctx, session, schemas...)
	if err != nil {
		return ValidationReport{}, err
	}
	return plan.Report, nil
}

func AutoMigrate(ctx context.Context, session Session, schemas ...SchemaResource) (ValidationReport, error) {
	plan, err := PlanSchemaChanges(ctx, session, schemas...)
	if err != nil {
		return ValidationReport{}, err
	}
	if plan.HasManualActions() {
		return plan.Report, SchemaDriftError{Report: plan.Report}
	}

	for _, action := range plan.Actions {
		if !action.Executable {
			continue
		}
		if _, execErr := session.ExecBoundContext(ctx, action.Statement); execErr != nil {
			return ValidationReport{}, execErr
		}
	}

	report, err := ValidateSchemas(ctx, session, schemas...)
	if err != nil {
		return ValidationReport{}, err
	}
	if !report.Valid() {
		return report, SchemaDriftError{Report: report}
	}
	return report, nil
}

func (db *DB) PlanSchemaChanges(ctx context.Context, schemas ...SchemaResource) (MigrationPlan, error) {
	return PlanSchemaChanges(ctx, db, schemas...)
}

func (db *DB) ValidateSchemas(ctx context.Context, schemas ...SchemaResource) (ValidationReport, error) {
	ctx, event, err := db.observe.before(ctx, HookEvent{
		Operation: OperationValidate,
		Table:     schemaNames(schemas),
	})
	if err != nil {
		return ValidationReport{}, err
	}
	report, validateErr := ValidateSchemas(ctx, db, schemas...)
	event.Err = validateErr
	db.observe.after(ctx, event)
	return report, validateErr
}

func (db *DB) AutoMigrate(ctx context.Context, schemas ...SchemaResource) (ValidationReport, error) {
	ctx, event, err := db.observe.before(ctx, HookEvent{
		Operation: OperationAutoMigrate,
		Table:     schemaNames(schemas),
	})
	if err != nil {
		return ValidationReport{}, err
	}
	report, migrateErr := AutoMigrate(ctx, db, schemas...)
	event.Err = migrateErr
	db.observe.after(ctx, event)
	return report, migrateErr
}

func (tx *Tx) PlanSchemaChanges(ctx context.Context, schemas ...SchemaResource) (MigrationPlan, error) {
	return PlanSchemaChanges(ctx, tx, schemas...)
}

func (tx *Tx) ValidateSchemas(ctx context.Context, schemas ...SchemaResource) (ValidationReport, error) {
	ctx, event, err := tx.observe.before(ctx, HookEvent{
		Operation: OperationValidate,
		Table:     schemaNames(schemas),
	})
	if err != nil {
		return ValidationReport{}, err
	}
	report, validateErr := ValidateSchemas(ctx, tx, schemas...)
	event.Err = validateErr
	tx.observe.after(ctx, event)
	return report, validateErr
}

func (tx *Tx) AutoMigrate(ctx context.Context, schemas ...SchemaResource) (ValidationReport, error) {
	ctx, event, err := tx.observe.before(ctx, HookEvent{
		Operation: OperationAutoMigrate,
		Table:     schemaNames(schemas),
	})
	if err != nil {
		return ValidationReport{}, err
	}
	report, migrateErr := AutoMigrate(ctx, tx, schemas...)
	event.Err = migrateErr
	tx.observe.after(ctx, event)
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

func diffSchema(ctx context.Context, schemaDialect SchemaDialect, session Session, schema SchemaResource) (TableDiff, error) {
	spec := buildTableSpec(schema.schemaRef())
	actual, err := schemaDialect.InspectTable(ctx, session, spec.Name)
	if err != nil {
		return TableDiff{}, err
	}

	diff := TableDiff{
		Table: spec.Name,
	}
	missingColumns := collectionx.NewListWithCapacity[ColumnMeta](len(spec.Columns))
	missingIndexes := collectionx.NewListWithCapacity[IndexMeta](len(spec.Indexes))
	missingForeignKeys := collectionx.NewListWithCapacity[ForeignKeyMeta](len(spec.ForeignKeys))
	missingChecks := collectionx.NewListWithCapacity[CheckMeta](len(spec.Checks))
	columnDiffs := collectionx.NewListWithCapacity[ColumnDiff](len(spec.Columns))
	if !actual.Exists {
		diff.MissingTable = true
		missingColumns.MergeSlice(spec.Columns)
		missingIndexes.MergeSlice(spec.Indexes)
		missingForeignKeys.MergeSlice(spec.ForeignKeys)
		missingChecks.MergeSlice(spec.Checks)
		diff.MissingColumns = missingColumns.Values()
		diff.MissingIndexes = missingIndexes.Values()
		diff.MissingForeignKeys = missingForeignKeys.Values()
		diff.MissingChecks = missingChecks.Values()
		if spec.PrimaryKey != nil {
			diff.PrimaryKeyDiff = &PrimaryKeyDiff{Expected: new(clonePrimaryKeyMeta(*spec.PrimaryKey)), Issues: []string{"table does not exist"}}
		}
		return diff, nil
	}

	actualColumns := collectionx.NewMapWithCapacity[string, ColumnState](len(actual.Columns))
	for _, column := range actual.Columns {
		actualColumns.Set(column.Name, column)
	}

	for _, expected := range spec.Columns {
		column, ok := actualColumns.Get(expected.Name)
		if !ok {
			missingColumns.Add(expected)
			continue
		}

		issues := collectionx.NewList[string]()
		expectedType := normalizeExpectedType(schemaDialect, expected)
		actualType := schemaDialect.NormalizeType(column.Type)
		if expectedType != "" && actualType != "" && expectedType != actualType {
			issues.Add("type mismatch: expected " + expectedType + " got " + actualType)
		}
		if !column.PrimaryKey && expected.Nullable != column.Nullable {
			issues.Add("nullable mismatch")
		}
		if expected.AutoIncrement != column.AutoIncrement {
			issues.Add("auto increment mismatch")
		}
		if expected.DefaultValue != "" && normalizeDefault(expected.DefaultValue) != normalizeDefault(column.DefaultValue) {
			issues.Add("default mismatch")
		}
		if issues.Len() > 0 {
			columnDiffs.Add(ColumnDiff{
				Column: expected,
				Issues: issues.Values(),
			})
		}
	}

	if issues := primaryKeyIssues(spec.PrimaryKey, actual.PrimaryKey); len(issues) > 0 {
		var expected *PrimaryKeyMeta
		if spec.PrimaryKey != nil {
			expected = new(clonePrimaryKeyMeta(*spec.PrimaryKey))
		}
		var current *PrimaryKeyState
		if actual.PrimaryKey != nil {
			current = new(clonePrimaryKeyState(*actual.PrimaryKey))
		}
		diff.PrimaryKeyDiff = &PrimaryKeyDiff{
			Expected: expected,
			Actual:   current,
			Issues:   issues,
		}
	}

	actualIndexes := collectionx.NewMapWithCapacity[string, IndexState](len(actual.Indexes))
	for _, index := range actual.Indexes {
		actualIndexes.Set(indexKey(index.Unique, index.Columns), index)
	}
	for _, expected := range spec.Indexes {
		if _, ok := actualIndexes.Get(indexKey(expected.Unique, expected.Columns)); !ok {
			missingIndexes.Add(expected)
		}
	}

	actualForeignKeys := collectionx.NewMapWithCapacity[string, ForeignKeyState](len(actual.ForeignKeys))
	for _, foreignKey := range actual.ForeignKeys {
		actualForeignKeys.Set(foreignKeyKeyFromState(foreignKey), foreignKey)
	}
	for _, expected := range spec.ForeignKeys {
		if _, ok := actualForeignKeys.Get(foreignKeyKey(expected)); !ok {
			missingForeignKeys.Add(expected)
		}
	}

	actualChecks := collectionx.NewMapWithCapacity[string, CheckState](len(actual.Checks))
	for _, check := range actual.Checks {
		actualChecks.Set(checkKey(check.Expression), check)
	}
	for _, expected := range spec.Checks {
		if _, ok := actualChecks.Get(checkKey(expected.Expression)); !ok {
			missingChecks.Add(expected)
		}
	}

	diff.MissingColumns = missingColumns.Values()
	diff.MissingIndexes = missingIndexes.Values()
	diff.MissingForeignKeys = missingForeignKeys.Values()
	diff.MissingChecks = missingChecks.Values()
	diff.ColumnDiffs = columnDiffs.Values()
	return diff, nil
}

func buildTableSpec(def schemaDefinition) TableSpec {
	return TableSpec{
		Name:        def.table.name,
		Columns:     append([]ColumnMeta(nil), def.columns...),
		Indexes:     deriveIndexes(def),
		PrimaryKey:  derivePrimaryKey(def),
		ForeignKeys: deriveForeignKeys(def),
		Checks:      deriveChecks(def),
	}
}

func deriveIndexes(def schemaDefinition) []IndexMeta {
	indexes := collectionx.NewOrderedMap[string, IndexMeta]()
	for _, index := range def.indexes {
		indexes.Set(indexKey(index.Unique, index.Columns), cloneIndexMeta(index))
	}
	for _, column := range def.columns {
		if column.PrimaryKey {
			continue
		}
		if !column.Unique && !column.Indexed {
			continue
		}
		prefix := "idx"
		if column.Unique {
			prefix = "ux"
		}
		meta := IndexMeta{
			Name:    prefix + "_" + def.table.name + "_" + column.Name,
			Table:   def.table.name,
			Columns: []string{column.Name},
			Unique:  column.Unique,
		}
		indexes.Set(indexKey(meta.Unique, meta.Columns), meta)
	}
	items := collectionx.NewListWithCapacity[IndexMeta](indexes.Len())
	indexes.Range(func(_ string, value IndexMeta) bool {
		items.Add(cloneIndexMeta(value))
		return true
	})
	return items.Values()
}

func derivePrimaryKey(def schemaDefinition) *PrimaryKeyMeta {
	if def.primaryKey != nil {
		copyPrimary := clonePrimaryKeyMeta(*def.primaryKey)
		if copyPrimary.Name == "" {
			copyPrimary.Name = "pk_" + def.table.name
		}
		if copyPrimary.Table == "" {
			copyPrimary.Table = def.table.name
		}
		return &copyPrimary
	}

	columns := collectionx.NewList[string]()
	for _, column := range def.columns {
		if column.PrimaryKey {
			columns.Add(column.Name)
		}
	}
	if columns.Len() == 0 {
		return nil
	}
	return &PrimaryKeyMeta{
		Name:    "pk_" + def.table.name,
		Table:   def.table.name,
		Columns: columns.Values(),
	}
}

func deriveForeignKeys(def schemaDefinition) []ForeignKeyMeta {
	foreignKeys := collectionx.NewOrderedMap[string, ForeignKeyMeta]()
	explicitColumns := collectionx.NewSet[string]()
	for _, column := range def.columns {
		if column.References == nil {
			continue
		}
		explicitColumns.Add(column.Name)
		meta := ForeignKeyMeta{
			Name:          "fk_" + def.table.name + "_" + column.Name,
			Table:         def.table.name,
			Columns:       []string{column.Name},
			TargetTable:   column.References.TargetTable,
			TargetColumns: []string{column.References.TargetColumn},
			OnDelete:      column.References.OnDelete,
			OnUpdate:      column.References.OnUpdate,
		}
		foreignKeys.Set(foreignKeyKey(meta), meta)
	}
	for _, relation := range def.relations {
		if relation.Kind != RelationBelongsTo || relation.LocalColumn == "" || relation.TargetColumn == "" || relation.TargetTable == "" {
			continue
		}
		if explicitColumns.Contains(relation.LocalColumn) {
			continue
		}
		if !hasColumn(def.columns, relation.LocalColumn) {
			continue
		}
		meta := ForeignKeyMeta{
			Name:          "fk_" + def.table.name + "_" + relation.LocalColumn,
			Table:         def.table.name,
			Columns:       []string{relation.LocalColumn},
			TargetTable:   relation.TargetTable,
			TargetColumns: []string{relation.TargetColumn},
		}
		key := foreignKeyKey(meta)
		if _, exists := foreignKeys.Get(key); !exists {
			foreignKeys.Set(key, meta)
		}
	}
	items := collectionx.NewListWithCapacity[ForeignKeyMeta](foreignKeys.Len())
	foreignKeys.Range(func(_ string, value ForeignKeyMeta) bool {
		items.Add(cloneForeignKeyMeta(value))
		return true
	})
	return items.Values()
}

func deriveChecks(def schemaDefinition) []CheckMeta {
	return lo.Map(def.checks, func(check CheckMeta, _ int) CheckMeta {
		return cloneCheckMeta(check)
	})
}

func buildCreateTableAction(schemaDialect SchemaDialect, spec TableSpec) MigrationAction {
	statement, err := schemaDialect.BuildCreateTable(spec)
	if err != nil {
		return MigrationAction{
			Kind:    MigrationActionManual,
			Table:   spec.Name,
			Summary: "manual create table migration required: " + err.Error(),
		}
	}
	return MigrationAction{
		Kind:       MigrationActionCreateTable,
		Table:      spec.Name,
		Summary:    "create table " + spec.Name,
		Statement:  statement,
		Executable: true,
	}
}

func buildAddColumnAction(schemaDialect SchemaDialect, table string, column ColumnMeta) MigrationAction {
	statement, err := schemaDialect.BuildAddColumn(table, column)
	if err != nil {
		return MigrationAction{
			Kind:    MigrationActionManual,
			Table:   table,
			Summary: "manual add column migration required for " + column.Name + ": " + err.Error(),
		}
	}
	return MigrationAction{
		Kind:       MigrationActionAddColumn,
		Table:      table,
		Summary:    "add column " + column.Name,
		Statement:  statement,
		Executable: true,
	}
}

func buildCreateIndexAction(schemaDialect SchemaDialect, index IndexMeta) MigrationAction {
	statement, err := schemaDialect.BuildCreateIndex(index)
	if err != nil {
		return MigrationAction{
			Kind:    MigrationActionManual,
			Table:   index.Table,
			Summary: "manual create index migration required for " + index.Name + ": " + err.Error(),
		}
	}
	return MigrationAction{
		Kind:       MigrationActionCreateIndex,
		Table:      index.Table,
		Summary:    "create index " + index.Name,
		Statement:  statement,
		Executable: true,
	}
}

func buildAddForeignKeyAction(schemaDialect SchemaDialect, table string, foreignKey ForeignKeyMeta) MigrationAction {
	statement, err := schemaDialect.BuildAddForeignKey(table, foreignKey)
	if err != nil {
		return MigrationAction{
			Kind:    MigrationActionManual,
			Table:   table,
			Summary: "manual add foreign key migration required for " + foreignKey.Name + ": " + err.Error(),
		}
	}
	return MigrationAction{
		Kind:       MigrationActionAddForeignKey,
		Table:      table,
		Summary:    "add foreign key " + foreignKey.Name,
		Statement:  statement,
		Executable: true,
	}
}

func buildAddCheckAction(schemaDialect SchemaDialect, table string, check CheckMeta) MigrationAction {
	statement, err := schemaDialect.BuildAddCheck(table, check)
	if err != nil {
		return MigrationAction{
			Kind:    MigrationActionManual,
			Table:   table,
			Summary: "manual add check migration required for " + check.Name + ": " + err.Error(),
		}
	}
	return MigrationAction{
		Kind:       MigrationActionAddCheck,
		Table:      table,
		Summary:    "add check " + check.Name,
		Statement:  statement,
		Executable: true,
	}
}

func primaryKeyIssues(expected *PrimaryKeyMeta, actual *PrimaryKeyState) []string {
	if expected == nil && actual == nil {
		return nil
	}
	if expected == nil && actual != nil {
		return []string{"unexpected primary key present"}
	}
	if expected != nil && actual == nil {
		return []string{"missing primary key"}
	}
	if columnsKey(expected.Columns) != columnsKey(actual.Columns) {
		return []string{"primary key columns mismatch"}
	}
	return nil
}

func hasColumn(columns []ColumnMeta, name string) bool {
	return lo.SomeBy(columns, func(column ColumnMeta) bool {
		return column.Name == name
	})
}

func indexKey(unique bool, columns []string) string {
	prefix := "idx:"
	if unique {
		prefix = "ux:"
	}
	return prefix + columnsKey(columns)
}

func foreignKeyKey(meta ForeignKeyMeta) string {
	return columnsKey(meta.Columns) + "->" + meta.TargetTable + ":" + columnsKey(meta.TargetColumns) + ":" + string(normalizeReferentialAction(meta.OnDelete)) + ":" + string(normalizeReferentialAction(meta.OnUpdate))
}

func foreignKeyKeyFromState(state ForeignKeyState) string {
	return columnsKey(state.Columns) + "->" + state.TargetTable + ":" + columnsKey(state.TargetColumns) + ":" + string(normalizeReferentialAction(state.OnDelete)) + ":" + string(normalizeReferentialAction(state.OnUpdate))
}

func checkKey(expression string) string {
	return normalizeCheckExpression(expression)
}

func columnsKey(columns []string) string {
	return strings.Join(columns, ",")
}

func normalizeReferentialAction(action ReferentialAction) ReferentialAction {
	if strings.TrimSpace(string(action)) == "" {
		return ReferentialNoAction
	}
	return action
}

func normalizeExpectedType(schemaDialect SchemaDialect, column ColumnMeta) string {
	if column.SQLType != "" {
		return schemaDialect.NormalizeType(column.SQLType)
	}
	return schemaDialect.NormalizeType(inferTypeName(column))
}

func inferTypeName(column ColumnMeta) string {
	if column.SQLType != "" {
		return column.SQLType
	}
	if column.GoType == nil {
		return ""
	}
	typ := column.GoType
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.PkgPath() == "time" && typ.Name() == "Time" {
		return "timestamp"
	}
	switch typ.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int32:
		return "integer"
	case reflect.Int64:
		return "bigint"
	case reflect.Uint, reflect.Uint32:
		return "integer"
	case reflect.Uint64:
		return "bigint"
	case reflect.Float32:
		return "real"
	case reflect.Float64:
		return "double"
	case reflect.String:
		return "text"
	case reflect.Slice:
		if typ.Elem().Kind() == reflect.Uint8 {
			return "blob"
		}
	}
	return strings.ToLower(typ.Name())
}

func normalizeDefault(value string) string {
	return strings.TrimSpace(strings.Trim(value, "()"))
}

func normalizeCheckExpression(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}

func schemaNames(schemas []SchemaResource) string {
	if len(schemas) == 0 {
		return ""
	}
	return strings.Join(lo.Map(schemas, func(schema SchemaResource, _ int) string {
		return schema.tableRef().TableName()
	}), ",")
}

func clonePrimaryKeyState(state PrimaryKeyState) PrimaryKeyState {
	state.Columns = append([]string(nil), state.Columns...)
	return state
}
