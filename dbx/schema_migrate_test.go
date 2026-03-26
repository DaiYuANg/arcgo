package dbx

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/DaiYuANg/arcgo/dbx/dialect"
)

type fakeSchemaDialect struct {
	tables   map[string]TableState
	actions  map[string]func()
	executed []string
}

type fakeSession struct {
	dialect *fakeSchemaDialect
}

type fakeResult struct{}

func newFakeSchemaDialect() *fakeSchemaDialect {
	return &fakeSchemaDialect{
		tables:   make(map[string]TableState),
		actions:  make(map[string]func()),
		executed: make([]string, 0, 8),
	}
}

func (d *fakeSchemaDialect) Name() string         { return "fake" }
func (d *fakeSchemaDialect) BindVar(_ int) string { return "?" }
func (d *fakeSchemaDialect) QuoteIdent(ident string) string {
	return `"` + ident + `"`
}
func (d *fakeSchemaDialect) RenderLimitOffset(limit, offset *int) (string, error) {
	return testSQLiteDialect{}.RenderLimitOffset(limit, offset)
}
func (d *fakeSchemaDialect) NormalizeType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func (d *fakeSchemaDialect) BuildCreateTable(spec TableSpec) (BoundQuery, error) {
	sql := "create table " + spec.Name
	columns := make([]ColumnState, len(spec.Columns))
	for i, column := range spec.Columns {
		columns[i] = ColumnState{
			Name:          column.Name,
			Type:          strings.ToLower(column.SQLType),
			Nullable:      column.Nullable,
			PrimaryKey:    column.PrimaryKey,
			AutoIncrement: column.AutoIncrement,
			DefaultValue:  column.DefaultValue,
		}
		if columns[i].Type == "" {
			columns[i].Type = strings.ToLower(inferTypeName(column))
		}
	}
	indexes := append([]IndexState(nil), toIndexStates(spec.Indexes)...)
	var primaryKey *PrimaryKeyState
	if spec.PrimaryKey != nil {
		copyPrimary := clonePrimaryKeyMeta(*spec.PrimaryKey)
		primaryKey = &PrimaryKeyState{Name: copyPrimary.Name, Columns: copyPrimary.Columns}
	}
	foreignKeys := append([]ForeignKeyState(nil), toForeignKeyStates(spec.ForeignKeys)...)
	checks := append([]CheckState(nil), toCheckStates(spec.Checks)...)
	d.actions[sql] = func() {
		d.tables[spec.Name] = TableState{
			Exists:      true,
			Name:        spec.Name,
			Columns:     columns,
			Indexes:     indexes,
			PrimaryKey:  primaryKey,
			ForeignKeys: foreignKeys,
			Checks:      checks,
		}
	}
	return BoundQuery{SQL: sql}, nil
}

func (d *fakeSchemaDialect) BuildAddColumn(table string, column ColumnMeta) (BoundQuery, error) {
	sql := "alter table " + table + " add column " + column.Name
	state := toColumnState(column)
	d.actions[sql] = func() {
		current := d.tables[table]
		current.Columns = append(current.Columns, state)
		if column.References != nil {
			current.ForeignKeys = append(current.ForeignKeys, ForeignKeyState{
				Name:          "fk_" + table + "_" + column.Name,
				Columns:       []string{column.Name},
				TargetTable:   column.References.TargetTable,
				TargetColumns: []string{column.References.TargetColumn},
				OnDelete:      column.References.OnDelete,
				OnUpdate:      column.References.OnUpdate,
			})
		}
		d.tables[table] = current
	}
	return BoundQuery{SQL: sql}, nil
}

func (d *fakeSchemaDialect) BuildCreateIndex(index IndexMeta) (BoundQuery, error) {
	sql := "create index " + index.Name + " on " + index.Table + "(" + strings.Join(index.Columns, ",") + ")"
	state := IndexState{Name: index.Name, Columns: append([]string(nil), index.Columns...), Unique: index.Unique}
	d.actions[sql] = func() {
		current := d.tables[index.Table]
		current.Indexes = append(current.Indexes, state)
		d.tables[index.Table] = current
	}
	return BoundQuery{SQL: sql}, nil
}

func (d *fakeSchemaDialect) BuildAddForeignKey(table string, foreignKey ForeignKeyMeta) (BoundQuery, error) {
	sql := "alter table " + table + " add constraint " + foreignKey.Name + " foreign key"
	state := ForeignKeyState{
		Name:          foreignKey.Name,
		Columns:       append([]string(nil), foreignKey.Columns...),
		TargetTable:   foreignKey.TargetTable,
		TargetColumns: append([]string(nil), foreignKey.TargetColumns...),
		OnDelete:      foreignKey.OnDelete,
		OnUpdate:      foreignKey.OnUpdate,
	}
	d.actions[sql] = func() {
		current := d.tables[table]
		current.ForeignKeys = append(current.ForeignKeys, state)
		d.tables[table] = current
	}
	return BoundQuery{SQL: sql}, nil
}

func (d *fakeSchemaDialect) BuildAddCheck(table string, check CheckMeta) (BoundQuery, error) {
	sql := "alter table " + table + " add constraint " + check.Name + " check"
	state := CheckState{Name: check.Name, Expression: check.Expression}
	d.actions[sql] = func() {
		current := d.tables[table]
		current.Checks = append(current.Checks, state)
		d.tables[table] = current
	}
	return BoundQuery{SQL: sql}, nil
}

func (d *fakeSchemaDialect) InspectTable(_ context.Context, _ Executor, table string) (TableState, error) {
	if state, ok := d.tables[table]; ok {
		copyState := state
		copyState.Columns = append([]ColumnState(nil), state.Columns...)
		copyState.Indexes = append([]IndexState(nil), state.Indexes...)
		if state.PrimaryKey != nil {
			copyState.PrimaryKey = new(clonePrimaryKeyState(*state.PrimaryKey))
		}
		copyState.ForeignKeys = append([]ForeignKeyState(nil), state.ForeignKeys...)
		copyState.Checks = append([]CheckState(nil), state.Checks...)
		return copyState, nil
	}
	return TableState{Name: table, Exists: false}, nil
}

func (s *fakeSession) Dialect() dialect.Dialect {
	return s.dialect
}

func (s *fakeSession) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, nil
}

func (s *fakeSession) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.ExecBoundContext(ctx, BoundQuery{SQL: query, Args: args})
}

func (s *fakeSession) QueryRowContext(context.Context, string, ...any) *Row {
	return errorRow(sql.ErrNoRows)
}

func (s *fakeSession) QueryBoundContext(context.Context, BoundQuery) (*sql.Rows, error) {
	return nil, nil
}

func (s *fakeSession) ExecBoundContext(_ context.Context, bound BoundQuery) (sql.Result, error) {
	if action, ok := s.dialect.actions[bound.SQL]; ok {
		action()
	}
	s.dialect.executed = append(s.dialect.executed, bound.SQL)
	return fakeResult{}, nil
}

func (s *fakeSession) SQL() *SQLExecutor {
	return &SQLExecutor{session: s}
}

func (fakeResult) LastInsertId() (int64, error) { return 0, nil }
func (fakeResult) RowsAffected() (int64, error) { return 1, nil }

func TestValidateSchemasReportsMissingTable(t *testing.T) {
	users := MustSchema("users", UserSchema{})
	session := &fakeSession{dialect: newFakeSchemaDialect()}

	report, err := ValidateSchemas(context.Background(), session, users)
	if err != nil {
		t.Fatalf("ValidateSchemas returned error: %v", err)
	}
	if report.Valid() {
		t.Fatal("expected invalid report for missing table")
	}
	if len(report.Tables) != 1 || !report.Tables[0].MissingTable {
		t.Fatalf("unexpected report: %+v", report)
	}
	if report.Complete {
		t.Fatal("expected legacy validation report to be partial")
	}
	if report.Backend != ValidationBackendLegacy {
		t.Fatalf("expected legacy backend report, got: %q", report.Backend)
	}
	if !report.HasWarnings() {
		t.Fatal("expected legacy validation warning")
	}
	if report.Tables[0].PrimaryKeyDiff == nil {
		t.Fatal("expected primary key diff for missing table")
	}
}

func TestAutoMigrateCreatesTableAndIndexes(t *testing.T) {
	users := MustSchema("users", UserSchema{})
	dialect := newFakeSchemaDialect()
	session := &fakeSession{dialect: dialect}

	report, err := AutoMigrate(context.Background(), session, users)
	if err != nil {
		t.Fatalf("AutoMigrate returned error: %v", err)
	}
	if !report.Valid() {
		t.Fatalf("expected valid report after automigrate: %+v", report)
	}
	if len(dialect.executed) != 2 {
		t.Fatalf("unexpected executed statement count: %d", len(dialect.executed))
	}
	if _, ok := dialect.tables["users"]; !ok {
		t.Fatal("expected users table to be created")
	}
}

func TestAutoMigrateReturnsDriftForIncompatibleColumn(t *testing.T) {
	users := MustSchema("users", UserSchema{})
	dialect := newFakeSchemaDialect()
	dialect.tables["users"] = TableState{
		Exists: true,
		Name:   "users",
		Columns: []ColumnState{
			{Name: "id", Type: "bigint", PrimaryKey: true},
			{Name: "username", Type: "bigint", Nullable: false},
			{Name: "email_address", Type: "text", Nullable: false},
			{Name: "status", Type: "integer", Nullable: false},
			{Name: "role_id", Type: "bigint", Nullable: false},
		},
		Indexes:    toIndexStates(deriveIndexes(users.schemaRef())),
		PrimaryKey: &PrimaryKeyState{Name: "pk_users", Columns: []string{"id"}},
	}
	session := &fakeSession{dialect: dialect}

	report, err := AutoMigrate(context.Background(), session, users)
	if err == nil {
		t.Fatal("expected schema drift error")
	}
	if _, ok := err.(SchemaDriftError); !ok {
		t.Fatalf("unexpected error type: %T", err)
	}
	if report.Valid() {
		t.Fatalf("expected invalid report: %+v", report)
	}
	if len(dialect.executed) != 0 {
		t.Fatalf("unexpected executed statements for incompatible drift: %#v", dialect.executed)
	}
}

type advancedRole struct {
	ID   int64  `dbx:"id"`
	Name string `dbx:"name"`
}

type advancedUser struct {
	ID       int64  `dbx:"id"`
	TenantID int64  `dbx:"tenant_id"`
	Username string `dbx:"username"`
	Status   int    `dbx:"status"`
	RoleID   int64  `dbx:"role_id"`
}

type advancedRoleSchema struct {
	Schema[advancedRole]
	ID   Column[advancedRole, int64]  `dbx:"id,pk,auto"`
	Name Column[advancedRole, string] `dbx:"name,unique"`
}

type advancedUserSchema struct {
	Schema[advancedUser]
	ID          Column[advancedUser, int64]           `dbx:"id"`
	TenantID    Column[advancedUser, int64]           `dbx:"tenant_id"`
	Username    Column[advancedUser, string]          `dbx:"username"`
	Status      Column[advancedUser, int]             `dbx:"status"`
	RoleID      Column[advancedUser, int64]           `dbx:"role_id"`
	Role        BelongsTo[advancedUser, advancedRole] `rel:"table=roles,local=role_id,target=id"`
	LookupIndex Index[advancedUser]                   `idx:"columns=tenant_id|username"`
	PK          CompositeKey[advancedUser]            `key:"columns=id|tenant_id"`
	StatusCheck Check[advancedUser]                   `check:"expr=status >= 0"`
}

func TestPlanSchemaChangesIncludesDerivedConstraints(t *testing.T) {
	users := MustSchema("users", advancedUserSchema{})
	dialect := newFakeSchemaDialect()
	session := &fakeSession{dialect: dialect}

	plan, err := PlanSchemaChanges(context.Background(), session, users)
	if err != nil {
		t.Fatalf("PlanSchemaChanges returned error: %v", err)
	}

	if len(plan.Actions) != 2 {
		t.Fatalf("unexpected action count: %d", len(plan.Actions))
	}
	if plan.Actions[0].Kind != MigrationActionCreateTable {
		t.Fatalf("unexpected first action: %+v", plan.Actions[0])
	}
	if plan.Actions[1].Kind != MigrationActionCreateIndex {
		t.Fatalf("unexpected second action: %+v", plan.Actions[1])
	}
	if report := plan.Report; len(report.Tables) != 1 || report.Tables[0].PrimaryKeyDiff == nil {
		t.Fatalf("unexpected report: %+v", report)
	}
	preview := plan.SQLPreview()
	if len(preview) != 2 {
		t.Fatalf("unexpected preview count: %d", len(preview))
	}
	if !strings.Contains(preview[0], "create table users") {
		t.Fatalf("unexpected preview sql: %+v", preview)
	}
}

func TestAutoMigrateAddsMissingForeignKeyAndCheck(t *testing.T) {
	users := MustSchema("users", advancedUserSchema{})
	dialect := newFakeSchemaDialect()
	dialect.tables["users"] = TableState{
		Exists: true,
		Name:   "users",
		Columns: []ColumnState{
			{Name: "id", Type: "bigint", Nullable: false},
			{Name: "tenant_id", Type: "bigint", Nullable: false},
			{Name: "username", Type: "text", Nullable: false},
			{Name: "status", Type: "integer", Nullable: false},
			{Name: "role_id", Type: "bigint", Nullable: false},
		},
		Indexes:    toIndexStates(deriveIndexes(users.schemaRef())),
		PrimaryKey: &PrimaryKeyState{Name: "pk_users", Columns: []string{"id", "tenant_id"}},
	}
	session := &fakeSession{dialect: dialect}

	report, err := AutoMigrate(context.Background(), session, users)
	if err != nil {
		t.Fatalf("AutoMigrate returned error: %v", err)
	}
	if !report.Valid() {
		t.Fatalf("expected valid report: %+v", report)
	}
	if len(dialect.tables["users"].ForeignKeys) != 1 {
		t.Fatalf("expected derived foreign key to be created: %+v", dialect.tables["users"].ForeignKeys)
	}
	if len(dialect.tables["users"].Checks) != 1 {
		t.Fatalf("expected check constraint to be created: %+v", dialect.tables["users"].Checks)
	}
}

type failingIndexDialect struct {
}

func (failingIndexDialect) Name() string         { return "failing-sqlite" }
func (failingIndexDialect) BindVar(_ int) string { return "?" }
func (failingIndexDialect) QuoteIdent(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}
func (failingIndexDialect) RenderLimitOffset(limit, offset *int) (string, error) {
	return testSQLiteDialect{}.RenderLimitOffset(limit, offset)
}
func (failingIndexDialect) NormalizeType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
func (d failingIndexDialect) BuildCreateTable(spec TableSpec) (BoundQuery, error) {
	parts := make([]string, 0, len(spec.Columns))
	singlePK := ""
	if spec.PrimaryKey != nil && len(spec.PrimaryKey.Columns) == 1 {
		singlePK = spec.PrimaryKey.Columns[0]
	}
	for _, column := range spec.Columns {
		typeName := column.SQLType
		if typeName == "" {
			typeName = inferTypeName(column)
		}
		part := d.QuoteIdent(column.Name) + " " + strings.ToUpper(typeName)
		if column.Name == singlePK {
			part += " PRIMARY KEY"
		} else if !column.Nullable {
			part += " NOT NULL"
		}
		parts = append(parts, part)
	}
	return BoundQuery{SQL: "CREATE TABLE IF NOT EXISTS " + d.QuoteIdent(spec.Name) + " (" + strings.Join(parts, ", ") + ")"}, nil
}
func (d failingIndexDialect) BuildAddColumn(table string, column ColumnMeta) (BoundQuery, error) {
	return BoundQuery{}, fmt.Errorf("unexpected add column for test table %s column %s", table, column.Name)
}
func (d failingIndexDialect) BuildCreateIndex(index IndexMeta) (BoundQuery, error) {
	return BoundQuery{SQL: "CREATE INDEX broken syntax"}, nil
}
func (d failingIndexDialect) BuildAddForeignKey(table string, foreignKey ForeignKeyMeta) (BoundQuery, error) {
	return BoundQuery{}, fmt.Errorf("unexpected add foreign key for test table %s constraint %s", table, foreignKey.Name)
}
func (d failingIndexDialect) BuildAddCheck(table string, check CheckMeta) (BoundQuery, error) {
	return BoundQuery{}, fmt.Errorf("unexpected add check for test table %s check %s", table, check.Name)
}
func (d failingIndexDialect) InspectTable(ctx context.Context, executor Executor, table string) (TableState, error) {
	rows, err := executor.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?", table)
	if err != nil {
		return TableState{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return TableState{Name: table, Exists: false}, nil
	}
	return TableState{Name: table, Exists: true}, rows.Err()
}

func TestAutoMigrateRollsBackTransactionalDDLOnFailure(t *testing.T) {
	ctx := context.Background()
	raw, cleanup := OpenTestSQLite(t)
	defer cleanup()

	core := MustNewWithOptions(raw, failingIndexDialect{})
	users := MustSchema("users", UserSchema{})

	_, err := core.AutoMigrate(ctx, users)
	if err == nil {
		t.Fatal("expected automigrate to fail on invalid index SQL")
	}

	var exists bool
	if scanErr := raw.QueryRow(`SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ?)`, "users").Scan(&exists); scanErr != nil {
		t.Fatalf("inspect sqlite_master: %v", scanErr)
	}
	if exists {
		t.Fatal("expected users table creation to roll back after transactional automigrate failure")
	}
}

func TestAutoMigrateWarnsWhenTransactionSupportIsUnavailable(t *testing.T) {
	users := MustSchema("users", UserSchema{})
	dialect := newFakeSchemaDialect()
	session := &fakeSession{dialect: dialect}

	report, err := AutoMigrate(context.Background(), session, users)
	if err != nil {
		t.Fatalf("AutoMigrate returned error: %v", err)
	}
	if !report.HasWarnings() {
		t.Fatal("expected auto migrate warning without transaction support")
	}
	if !strings.Contains(strings.Join(report.Warnings, " "), "without transaction") {
		t.Fatalf("expected non-transactional warning, got: %+v", report.Warnings)
	}
}

func toColumnState(column ColumnMeta) ColumnState {
	typeName := column.SQLType
	if typeName == "" {
		typeName = inferTypeName(column)
	}
	return ColumnState{
		Name:          column.Name,
		Type:          strings.ToLower(typeName),
		Nullable:      column.Nullable,
		PrimaryKey:    column.PrimaryKey,
		AutoIncrement: column.AutoIncrement,
		DefaultValue:  column.DefaultValue,
	}
}

func toIndexStates(indexes []IndexMeta) []IndexState {
	items := make([]IndexState, len(indexes))
	for i, index := range indexes {
		items[i] = IndexState{
			Name:    index.Name,
			Columns: append([]string(nil), index.Columns...),
			Unique:  index.Unique,
		}
	}
	return items
}

func toForeignKeyStates(foreignKeys []ForeignKeyMeta) []ForeignKeyState {
	items := make([]ForeignKeyState, len(foreignKeys))
	for i, foreignKey := range foreignKeys {
		items[i] = ForeignKeyState{
			Name:          foreignKey.Name,
			Columns:       append([]string(nil), foreignKey.Columns...),
			TargetTable:   foreignKey.TargetTable,
			TargetColumns: append([]string(nil), foreignKey.TargetColumns...),
			OnDelete:      foreignKey.OnDelete,
			OnUpdate:      foreignKey.OnUpdate,
		}
	}
	return items
}

func toCheckStates(checks []CheckMeta) []CheckState {
	items := make([]CheckState, len(checks))
	for i, check := range checks {
		items[i] = CheckState{Name: check.Name, Expression: check.Expression}
	}
	return items
}
