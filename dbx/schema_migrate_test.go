package dbx_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/DaiYuANg/arcgo/collectionx"
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
	stmt := "create table " + spec.Name
	columns := collectionx.NewListWithCapacity[ColumnState](spec.Columns.Len())
	spec.Columns.Range(func(_ int, column ColumnMeta) bool {
		state := ColumnState{
			Name:          column.Name,
			Type:          strings.ToLower(column.SQLType),
			Nullable:      column.Nullable,
			PrimaryKey:    column.PrimaryKey,
			AutoIncrement: column.AutoIncrement,
			DefaultValue:  column.DefaultValue,
		}
		if state.Type == "" {
			state.Type = strings.ToLower(InferTypeNameForTest(column))
		}
		columns.Add(state)
		return true
	})
	indexes := toIndexStates(spec.Indexes)
	var primaryKey *PrimaryKeyState
	if spec.PrimaryKey != nil {
		copyPrimary := ClonePrimaryKeyMetaForTest(*spec.PrimaryKey)
		primaryKey = &PrimaryKeyState{Name: copyPrimary.Name, Columns: copyPrimary.Columns}
	}
	foreignKeys := toForeignKeyStates(spec.ForeignKeys)
	checks := toCheckStates(spec.Checks)
	d.actions[stmt] = func() {
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
	return BoundQuery{SQL: stmt}, nil
}

func (d *fakeSchemaDialect) BuildAddColumn(table string, column ColumnMeta) (BoundQuery, error) {
	stmt := "alter table " + table + " add column " + column.Name
	state := toColumnState(column)
	d.actions[stmt] = func() {
		current := d.tables[table]
		if current.Columns == nil {
			current.Columns = collectionx.NewList[ColumnState]()
		}
		current.Columns.Add(state)
		if column.References != nil {
			if current.ForeignKeys == nil {
				current.ForeignKeys = collectionx.NewList[ForeignKeyState]()
			}
			current.ForeignKeys.Add(ForeignKeyState{
				Name:          "fk_" + table + "_" + column.Name,
				Columns:       collectionx.NewList(column.Name),
				TargetTable:   column.References.TargetTable,
				TargetColumns: collectionx.NewList(column.References.TargetColumn),
				OnDelete:      column.References.OnDelete,
				OnUpdate:      column.References.OnUpdate,
			})
		}
		d.tables[table] = current
	}
	return BoundQuery{SQL: stmt}, nil
}

func (d *fakeSchemaDialect) BuildCreateIndex(index IndexMeta) (BoundQuery, error) {
	stmt := "create index " + index.Name + " on " + index.Table + "(" + strings.Join(index.Columns.Values(), ",") + ")"
	state := IndexState{Name: index.Name, Columns: index.Columns.Clone(), Unique: index.Unique}
	d.actions[stmt] = func() {
		current := d.tables[index.Table]
		if current.Indexes == nil {
			current.Indexes = collectionx.NewList[IndexState]()
		}
		current.Indexes.Add(state)
		d.tables[index.Table] = current
	}
	return BoundQuery{SQL: stmt}, nil
}

func (d *fakeSchemaDialect) BuildAddForeignKey(table string, foreignKey ForeignKeyMeta) (BoundQuery, error) {
	stmt := "alter table " + table + " add constraint " + foreignKey.Name + " foreign key"
	state := ForeignKeyState{
		Name:          foreignKey.Name,
		Columns:       foreignKey.Columns.Clone(),
		TargetTable:   foreignKey.TargetTable,
		TargetColumns: foreignKey.TargetColumns.Clone(),
		OnDelete:      foreignKey.OnDelete,
		OnUpdate:      foreignKey.OnUpdate,
	}
	d.actions[stmt] = func() {
		current := d.tables[table]
		if current.ForeignKeys == nil {
			current.ForeignKeys = collectionx.NewList[ForeignKeyState]()
		}
		current.ForeignKeys.Add(state)
		d.tables[table] = current
	}
	return BoundQuery{SQL: stmt}, nil
}

func (d *fakeSchemaDialect) BuildAddCheck(table string, check CheckMeta) (BoundQuery, error) {
	stmt := "alter table " + table + " add constraint " + check.Name + " check"
	state := CheckState{Name: check.Name, Expression: check.Expression}
	d.actions[stmt] = func() {
		current := d.tables[table]
		if current.Checks == nil {
			current.Checks = collectionx.NewList[CheckState]()
		}
		current.Checks.Add(state)
		d.tables[table] = current
	}
	return BoundQuery{SQL: stmt}, nil
}

func (d *fakeSchemaDialect) InspectTable(_ context.Context, _ Executor, table string) (TableState, error) {
	if state, ok := d.tables[table]; ok {
		copyState := state
		copyState.Columns = state.Columns.Clone()
		copyState.Indexes = state.Indexes.Clone()
		if state.PrimaryKey != nil {
			copyState.PrimaryKey = new(PrimaryKeyState)
			*copyState.PrimaryKey = ClonePrimaryKeyStateForTest(*state.PrimaryKey)
		}
		copyState.ForeignKeys = state.ForeignKeys.Clone()
		copyState.Checks = state.Checks.Clone()
		return copyState, nil
	}
	return TableState{Name: table, Exists: false}, nil
}

func (s *fakeSession) Dialect() dialect.Dialect {
	return s.dialect
}

func (s *fakeSession) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	var rows *sql.Rows
	return rows, nil
}

func (s *fakeSession) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.ExecBoundContext(ctx, BoundQuery{SQL: query, Args: collectionx.NewList(args...)})
}

func (s *fakeSession) QueryRowContext(context.Context, string, ...any) *Row {
	return ErrorRowForTest(sql.ErrNoRows)
}

func (s *fakeSession) QueryBoundContext(context.Context, BoundQuery) (*sql.Rows, error) {
	var rows *sql.Rows
	return rows, nil
}

func (s *fakeSession) ExecBoundContext(_ context.Context, bound BoundQuery) (sql.Result, error) {
	if action, ok := s.dialect.actions[bound.SQL]; ok {
		action()
	}
	s.dialect.executed = append(s.dialect.executed, bound.SQL)
	return fakeResult{}, nil
}

func (s *fakeSession) SQL() *SQLExecutor {
	return NewSQLExecutorForTest(s)
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
	firstTable, ok := report.Tables.Get(0)
	if report.Tables.Len() != 1 || !ok || !firstTable.MissingTable {
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
	if firstTable.PrimaryKeyDiff == nil {
		t.Fatal("expected primary key diff for missing table")
	}
}

func TestAutoMigrateCreatesTableAndIndexes(t *testing.T) {
	users := MustSchema("users", UserSchema{})
	schemaDialect := newFakeSchemaDialect()
	session := &fakeSession{dialect: schemaDialect}

	report, err := AutoMigrate(context.Background(), session, users)
	if err != nil {
		t.Fatalf("AutoMigrate returned error: %v", err)
	}
	if !report.Valid() {
		t.Fatalf("expected valid report after automigrate: %+v", report)
	}
	if len(schemaDialect.executed) != 2 {
		t.Fatalf("unexpected executed statement count: %d", len(schemaDialect.executed))
	}
	if _, ok := schemaDialect.tables["users"]; !ok {
		t.Fatal("expected users table to be created")
	}
}

func TestAutoMigrateReturnsDriftForIncompatibleColumn(t *testing.T) {
	users := MustSchema("users", UserSchema{})
	schemaDialect := newFakeSchemaDialect()
	schemaDialect.tables["users"] = TableState{
		Exists: true,
		Name:   "users",
		Columns: collectionx.NewList(
			ColumnState{Name: "id", Type: "bigint", PrimaryKey: true},
			ColumnState{Name: "username", Type: "bigint", Nullable: false},
			ColumnState{Name: "email_address", Type: "text", Nullable: false},
			ColumnState{Name: "status", Type: "integer", Nullable: false},
			ColumnState{Name: "role_id", Type: "bigint", Nullable: false},
		),
		Indexes:    toIndexStates(IndexesForTest(users)),
		PrimaryKey: &PrimaryKeyState{Name: "pk_users", Columns: collectionx.NewList("id")},
	}
	session := &fakeSession{dialect: schemaDialect}

	report, err := AutoMigrate(context.Background(), session, users)
	if err == nil {
		t.Fatal("expected schema drift error")
	}
	var driftErr SchemaDriftError
	if !errors.As(err, &driftErr) {
		t.Fatalf("unexpected error type: %T", err)
	}
	if report.Valid() {
		t.Fatalf("expected invalid report: %+v", report)
	}
	if len(schemaDialect.executed) != 0 {
		t.Fatalf("unexpected executed statements for incompatible drift: %#v", schemaDialect.executed)
	}
}
