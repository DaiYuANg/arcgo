package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/DaiYuANg/arcgo/dbx/dialect"
	"github.com/samber/lo"
)

const (
	postgresTableExistsQuery = "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = $1)"
	postgresPrimaryKeyQuery  = "SELECT tc.constraint_name, kcu.column_name FROM information_schema.table_constraints tc JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema AND tc.table_name = kcu.table_name WHERE tc.table_schema = current_schema() AND tc.table_name = $1 AND tc.constraint_type = 'PRIMARY KEY' ORDER BY kcu.ordinal_position"
	postgresColumnsQuery     = "SELECT c.column_name, c.udt_name, c.is_nullable, c.column_default, (c.is_identity = 'YES') AS is_identity FROM information_schema.columns c WHERE c.table_schema = current_schema() AND c.table_name = $1 ORDER BY c.ordinal_position"
	postgresIndexesQuery     = "SELECT indexname, indexdef FROM pg_indexes WHERE schemaname = current_schema() AND tablename = $1"
	postgresForeignKeysQuery = "SELECT tc.constraint_name, kcu.column_name, ccu.table_name, ccu.column_name, rc.update_rule, rc.delete_rule FROM information_schema.table_constraints tc JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema AND tc.table_name = kcu.table_name JOIN information_schema.constraint_column_usage ccu ON tc.constraint_name = ccu.constraint_name AND tc.table_schema = ccu.table_schema JOIN information_schema.referential_constraints rc ON tc.constraint_name = rc.constraint_name AND tc.table_schema = rc.constraint_schema WHERE tc.table_schema = current_schema() AND tc.table_name = $1 AND tc.constraint_type = 'FOREIGN KEY' ORDER BY tc.constraint_name, kcu.ordinal_position"
	postgresChecksQuery      = "SELECT tc.constraint_name, cc.check_clause FROM information_schema.table_constraints tc JOIN information_schema.check_constraints cc ON tc.constraint_name = cc.constraint_name AND tc.table_schema = cc.constraint_schema WHERE tc.table_schema = current_schema() AND tc.table_name = $1 AND tc.constraint_type = 'CHECK' ORDER BY tc.constraint_name"
)

var postgresNormalizedTypes = map[string]string{
	"int2":                        "smallint",
	"smallint":                    "smallint",
	"int4":                        "integer",
	"integer":                     "integer",
	"serial":                      "integer",
	"serial4":                     "integer",
	"int8":                        "bigint",
	"bigint":                      "bigint",
	"bigserial":                   "bigint",
	"serial8":                     "bigint",
	"float4":                      "real",
	"real":                        "real",
	"float8":                      "double",
	"double precision":            "double",
	"numeric":                     "double",
	"decimal":                     "double",
	"bool":                        "boolean",
	"boolean":                     "boolean",
	"varchar":                     "text",
	"bpchar":                      "text",
	"text":                        "text",
	"citext":                      "text",
	"bytea":                       "blob",
	"timestamp":                   "timestamp",
	"timestamptz":                 "timestamp",
	"timestamp with time zone":    "timestamp",
	"timestamp without time zone": "timestamp",
}

var (
	postgresIntKinds         = []reflect.Kind{reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32}
	postgresUnsignedIntKinds = []reflect.Kind{reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32}
)

// Dialect implements PostgreSQL rendering and schema inspection.
type Dialect struct{}

// New returns a PostgreSQL dialect implementation.
func New() Dialect { return Dialect{} }

// Name returns the dialect name.
func (Dialect) Name() string { return "postgres" }

// BindVar returns the bind placeholder for a parameter index.
func (Dialect) BindVar(n int) string { return "$" + strconv.Itoa(n) }

// QuoteIdent quotes an identifier for PostgreSQL.
func (Dialect) QuoteIdent(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}

// RenderLimitOffset renders a LIMIT/OFFSET clause for PostgreSQL.
func (Dialect) RenderLimitOffset(limit, offset *int) (string, error) {
	if limit == nil && offset == nil {
		return "", nil
	}
	if limit != nil && offset != nil {
		return fmt.Sprintf("LIMIT %d OFFSET %d", *limit, *offset), nil
	}
	if limit != nil {
		return fmt.Sprintf("LIMIT %d", *limit), nil
	}
	return fmt.Sprintf("OFFSET %d", *offset), nil
}

// QueryFeatures returns the supported query feature set.
func (Dialect) QueryFeatures() dialect.QueryFeatures {
	return dialect.DefaultQueryFeatures("postgres")
}

// BuildCreateTable builds a CREATE TABLE statement.
func (d Dialect) BuildCreateTable(spec dbx.TableSpec) (dbx.BoundQuery, error) {
	parts := collectionx.NewListWithCapacity[string](len(spec.Columns) + len(spec.ForeignKeys) + len(spec.Checks) + 1)
	inlinePrimaryKey := singlePrimaryKeyColumn(spec.PrimaryKey)
	parts.Add(lo.Map(spec.Columns, func(column dbx.ColumnMeta, _ int) string {
		return d.columnDDL(column, inlinePrimaryKey == column.Name, false)
	})...)
	if spec.PrimaryKey != nil && len(spec.PrimaryKey.Columns) > 1 {
		parts.Add(d.primaryKeyDDL(*spec.PrimaryKey))
	}
	parts.Add(lo.Map(spec.ForeignKeys, func(fk dbx.ForeignKeyMeta, _ int) string {
		return d.foreignKeyDDL(fk)
	})...)
	parts.Add(lo.Map(spec.Checks, func(check dbx.CheckMeta, _ int) string {
		return d.checkDDL(check)
	})...)
	return dbx.BoundQuery{
		SQL: "CREATE TABLE IF NOT EXISTS " + d.QuoteIdent(spec.Name) + " (" + strings.Join(parts.Values(), ", ") + ")",
	}, nil
}

// BuildAddColumn builds an ALTER TABLE ADD COLUMN statement.
func (d Dialect) BuildAddColumn(table string, column dbx.ColumnMeta) (dbx.BoundQuery, error) {
	return dbx.BoundQuery{
		SQL: "ALTER TABLE " + d.QuoteIdent(table) + " ADD COLUMN " + d.columnDDL(column, false, true),
	}, nil
}

// BuildCreateIndex builds a CREATE INDEX statement.
func (d Dialect) BuildCreateIndex(index dbx.IndexMeta) (dbx.BoundQuery, error) {
	columns := lo.Map(index.Columns, func(column string, _ int) string {
		return d.QuoteIdent(column)
	})
	prefix := "CREATE INDEX IF NOT EXISTS "
	if index.Unique {
		prefix = "CREATE UNIQUE INDEX IF NOT EXISTS "
	}
	return dbx.BoundQuery{
		SQL: prefix + d.QuoteIdent(index.Name) + " ON " + d.QuoteIdent(index.Table) + " (" + strings.Join(columns, ", ") + ")",
	}, nil
}

// BuildAddForeignKey builds an ALTER TABLE ADD CONSTRAINT statement for a foreign key.
func (d Dialect) BuildAddForeignKey(table string, foreignKey dbx.ForeignKeyMeta) (dbx.BoundQuery, error) {
	return dbx.BoundQuery{
		SQL: "ALTER TABLE " + d.QuoteIdent(table) + " ADD " + d.foreignKeyDDL(foreignKey),
	}, nil
}

// BuildAddCheck builds an ALTER TABLE ADD CONSTRAINT statement for a check.
func (d Dialect) BuildAddCheck(table string, check dbx.CheckMeta) (dbx.BoundQuery, error) {
	return dbx.BoundQuery{
		SQL: "ALTER TABLE " + d.QuoteIdent(table) + " ADD " + d.checkDDL(check),
	}, nil
}

// InspectTable inspects a PostgreSQL table definition from system catalogs.
func (d Dialect) InspectTable(ctx context.Context, executor dbx.Executor, table string) (dbx.TableState, error) {
	exists, err := inspectPostgresTableExists(ctx, executor, table)
	if err != nil {
		return dbx.TableState{}, err
	}
	if !exists {
		return dbx.TableState{Name: table, Exists: false}, nil
	}

	primaryKey, primaryColumns, err := inspectPostgresPrimaryKey(ctx, executor, table)
	if err != nil {
		return dbx.TableState{}, err
	}

	columns, err := d.inspectColumns(ctx, executor, table, primaryColumns)
	if err != nil {
		return dbx.TableState{}, err
	}

	indexes, err := d.inspectIndexes(ctx, executor, table)
	if err != nil {
		return dbx.TableState{}, err
	}

	foreignKeys, err := d.inspectForeignKeys(ctx, executor, table)
	if err != nil {
		return dbx.TableState{}, err
	}

	checks, err := d.inspectChecks(ctx, executor, table)
	if err != nil {
		return dbx.TableState{}, err
	}

	return dbx.TableState{
		Exists:      true,
		Name:        table,
		Columns:     columns,
		Indexes:     indexes,
		PrimaryKey:  primaryKey,
		ForeignKeys: foreignKeys,
		Checks:      checks,
	}, nil
}

// NormalizeType normalizes database type names into dbx logical types.
func (Dialect) NormalizeType(value string) string {
	typeName := strings.ToLower(strings.TrimSpace(value))
	if normalized, ok := postgresNormalizedTypes[typeName]; ok {
		return normalized
	}
	return typeName
}

func inspectPostgresTableExists(ctx context.Context, executor dbx.Executor, table string) (exists bool, resultErr error) {
	const action = "inspect postgres table existence"

	rows, err := queryPostgresRows(ctx, executor, action, postgresTableExistsQuery, table)
	if err != nil {
		return false, err
	}
	defer func() {
		if closeErr := closePostgresRows(action, rows); closeErr != nil && resultErr == nil {
			resultErr = closeErr
		}
	}()

	if rows.Next() {
		if scanErr := rows.Scan(&exists); scanErr != nil {
			return false, fmt.Errorf("%s: scan row: %w", action, scanErr)
		}
	}
	if rowsErr := postgresRowsError(action, rows); rowsErr != nil {
		return false, rowsErr
	}

	return exists, nil
}

func inspectPostgresPrimaryKey(ctx context.Context, executor dbx.Executor, table string) (_ *dbx.PrimaryKeyState, _ map[string]struct{}, resultErr error) {
	const action = "inspect postgres primary key"

	rows, err := queryPostgresRows(ctx, executor, action, postgresPrimaryKeyQuery, table)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if closeErr := closePostgresRows(action, rows); closeErr != nil && resultErr == nil {
			resultErr = closeErr
		}
	}()

	columns := make([]string, 0, 2)
	columnSet := make(map[string]struct{}, 2)
	name := ""
	for rows.Next() {
		constraintName, column, scanErr := scanPostgresPrimaryKey(rows)
		if scanErr != nil {
			return nil, nil, scanErr
		}
		name = constraintName
		columns = append(columns, column)
		columnSet[column] = struct{}{}
	}

	if rowsErr := postgresRowsError(action, rows); rowsErr != nil {
		return nil, nil, rowsErr
	}

	return postgresPrimaryKeyState(name, columns), columnSet, nil
}

func (d Dialect) inspectColumns(ctx context.Context, executor dbx.Executor, table string, primaryColumns map[string]struct{}) (_ []dbx.ColumnState, resultErr error) {
	const action = "inspect postgres columns"

	rows, err := queryPostgresRows(ctx, executor, action, postgresColumnsQuery, table)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := closePostgresRows(action, rows); closeErr != nil && resultErr == nil {
			resultErr = closeErr
		}
	}()

	columns := make([]dbx.ColumnState, 0, 8)
	for rows.Next() {
		column, scanErr := scanPostgresColumn(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		column.PrimaryKey = postgresPrimaryColumn(primaryColumns, column.Name)
		columns = append(columns, column)
	}

	if rowsErr := postgresRowsError(action, rows); rowsErr != nil {
		return nil, rowsErr
	}

	return columns, nil
}

func (d Dialect) inspectIndexes(ctx context.Context, executor dbx.Executor, table string) (_ []dbx.IndexState, resultErr error) {
	const action = "inspect postgres indexes"

	rows, err := queryPostgresRows(ctx, executor, action, postgresIndexesQuery, table)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := closePostgresRows(action, rows); closeErr != nil && resultErr == nil {
			resultErr = closeErr
		}
	}()

	indexes := make([]dbx.IndexState, 0, 4)
	for rows.Next() {
		index, skip, scanErr := scanPostgresIndex(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		if skip {
			continue
		}
		indexes = append(indexes, index)
	}

	if rowsErr := postgresRowsError(action, rows); rowsErr != nil {
		return nil, rowsErr
	}

	return indexes, nil
}

func (d Dialect) inspectForeignKeys(ctx context.Context, executor dbx.Executor, table string) (_ []dbx.ForeignKeyState, resultErr error) {
	const action = "inspect postgres foreign keys"

	rows, err := queryPostgresRows(ctx, executor, action, postgresForeignKeysQuery, table)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := closePostgresRows(action, rows); closeErr != nil && resultErr == nil {
			resultErr = closeErr
		}
	}()

	groups := collectionx.NewOrderedMap[string, dbx.ForeignKeyState]()
	for rows.Next() {
		name, state, scanErr := scanPostgresForeignKey(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		appendPostgresForeignKey(groups, name, state)
	}

	if rowsErr := postgresRowsError(action, rows); rowsErr != nil {
		return nil, rowsErr
	}

	foreignKeys := make([]dbx.ForeignKeyState, 0, groups.Len())
	groups.Range(func(_ string, value dbx.ForeignKeyState) bool {
		foreignKeys = append(foreignKeys, value)
		return true
	})
	return foreignKeys, nil
}

func (d Dialect) inspectChecks(ctx context.Context, executor dbx.Executor, table string) (_ []dbx.CheckState, resultErr error) {
	const action = "inspect postgres checks"

	rows, err := queryPostgresRows(ctx, executor, action, postgresChecksQuery, table)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := closePostgresRows(action, rows); closeErr != nil && resultErr == nil {
			resultErr = closeErr
		}
	}()

	checks := make([]dbx.CheckState, 0, 4)
	for rows.Next() {
		check, scanErr := scanPostgresCheck(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		checks = append(checks, check)
	}

	if rowsErr := postgresRowsError(action, rows); rowsErr != nil {
		return nil, rowsErr
	}

	return checks, nil
}

func scanPostgresPrimaryKey(rows *sql.Rows) (string, string, error) {
	var name string
	var column string

	if err := rows.Scan(&name, &column); err != nil {
		return "", "", fmt.Errorf("scan postgres primary key: %w", err)
	}
	return name, column, nil
}

func scanPostgresColumn(rows *sql.Rows) (dbx.ColumnState, error) {
	var name string
	var udtName string
	var isNullable string
	var defaultValue sql.NullString
	var isIdentity bool

	if err := rows.Scan(&name, &udtName, &isNullable, &defaultValue, &isIdentity); err != nil {
		return dbx.ColumnState{}, fmt.Errorf("scan postgres column: %w", err)
	}

	return dbx.ColumnState{
		Name:          name,
		Type:          udtName,
		Nullable:      strings.EqualFold(isNullable, "YES"),
		AutoIncrement: isIdentity || strings.Contains(strings.ToLower(defaultValue.String), "nextval"),
		DefaultValue:  defaultValue.String,
	}, nil
}

func scanPostgresIndex(rows *sql.Rows) (dbx.IndexState, bool, error) {
	var name string
	var definition string

	if err := rows.Scan(&name, &definition); err != nil {
		return dbx.IndexState{}, false, fmt.Errorf("scan postgres index: %w", err)
	}

	upperDefinition := strings.ToUpper(definition)
	if strings.Contains(upperDefinition, "PRIMARY KEY") {
		return dbx.IndexState{}, true, nil
	}

	return dbx.IndexState{
		Name:    name,
		Columns: parseIndexColumns(definition),
		Unique:  strings.Contains(upperDefinition, "CREATE UNIQUE INDEX"),
	}, false, nil
}

func scanPostgresForeignKey(rows *sql.Rows) (string, dbx.ForeignKeyState, error) {
	var name string
	var column string
	var targetTable string
	var targetColumn string
	var updateRule string
	var deleteRule string

	if err := rows.Scan(&name, &column, &targetTable, &targetColumn, &updateRule, &deleteRule); err != nil {
		return "", dbx.ForeignKeyState{}, fmt.Errorf("scan postgres foreign key: %w", err)
	}

	return name, dbx.ForeignKeyState{
		Name:          name,
		TargetTable:   targetTable,
		Columns:       []string{column},
		TargetColumns: []string{targetColumn},
		OnDelete:      referentialAction(deleteRule),
		OnUpdate:      referentialAction(updateRule),
	}, nil
}

func scanPostgresCheck(rows *sql.Rows) (dbx.CheckState, error) {
	var name string
	var clause string

	if err := rows.Scan(&name, &clause); err != nil {
		return dbx.CheckState{}, fmt.Errorf("scan postgres check: %w", err)
	}

	return dbx.CheckState{Name: name, Expression: clause}, nil
}

func postgresPrimaryKeyState(name string, columns []string) *dbx.PrimaryKeyState {
	if len(columns) == 0 {
		return nil
	}
	return &dbx.PrimaryKeyState{Name: name, Columns: columns}
}

func postgresPrimaryColumn(columns map[string]struct{}, name string) bool {
	_, ok := columns[name]
	return ok
}

func appendPostgresForeignKey(groups collectionx.OrderedMap[string, dbx.ForeignKeyState], name string, state dbx.ForeignKeyState) {
	current, ok := groups.Get(name)
	if !ok {
		groups.Set(name, state)
		return
	}
	current.Columns = append(current.Columns, state.Columns...)
	current.TargetColumns = append(current.TargetColumns, state.TargetColumns...)
	groups.Set(name, current)
}

func queryPostgresRows(ctx context.Context, executor dbx.Executor, action, query string, args ...any) (*sql.Rows, error) {
	rows, err := executor.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", action, err)
	}
	return rows, nil
}

func closePostgresRows(action string, rows *sql.Rows) error {
	if rows == nil {
		return nil
	}
	if closeErr := rows.Close(); closeErr != nil {
		return fmt.Errorf("%s: close rows: %w", action, closeErr)
	}
	return nil
}

func postgresRowsError(action string, rows *sql.Rows) error {
	if err := rows.Err(); err != nil {
		return fmt.Errorf("%s: rows err: %w", action, err)
	}
	return nil
}

func (d Dialect) columnDDL(column dbx.ColumnMeta, inlinePrimaryKey, includeReference bool) string {
	parts := []string{
		d.QuoteIdent(column.Name),
		postgresColumnTypeDDL(column),
	}

	parts = append(parts, postgresColumnConstraintParts(column, inlinePrimaryKey)...)
	if includeReference {
		parts = append(parts, d.postgresReferenceParts(column)...)
	}

	return strings.Join(parts, " ")
}

func postgresColumnTypeDDL(column dbx.ColumnMeta) string {
	typeName := resolvedPostgresType(column)
	if column.AutoIncrement {
		return typeName + " GENERATED BY DEFAULT AS IDENTITY"
	}
	return typeName
}

func postgresColumnConstraintParts(column dbx.ColumnMeta, inlinePrimaryKey bool) []string {
	parts := make([]string, 0, 3)
	if inlinePrimaryKey {
		parts = append(parts, "PRIMARY KEY")
	}
	if !column.Nullable && !inlinePrimaryKey {
		parts = append(parts, "NOT NULL")
	}
	if column.DefaultValue != "" && !column.AutoIncrement {
		parts = append(parts, "DEFAULT "+column.DefaultValue)
	}
	return parts
}

func (d Dialect) postgresReferenceParts(column dbx.ColumnMeta) []string {
	if column.References == nil {
		return nil
	}

	parts := []string{
		"REFERENCES " + d.QuoteIdent(column.References.TargetTable) + " (" + d.QuoteIdent(column.References.TargetColumn) + ")",
	}
	if column.References.OnDelete != "" {
		parts = append(parts, "ON DELETE "+string(column.References.OnDelete))
	}
	if column.References.OnUpdate != "" {
		parts = append(parts, "ON UPDATE "+string(column.References.OnUpdate))
	}
	return parts
}

func resolvedPostgresType(column dbx.ColumnMeta) string {
	if column.SQLType != "" {
		return column.SQLType
	}
	return postgresType(column)
}

func (d Dialect) primaryKeyDDL(primaryKey dbx.PrimaryKeyMeta) string {
	columns := lo.Map(primaryKey.Columns, func(column string, _ int) string {
		return d.QuoteIdent(column)
	})
	return "CONSTRAINT " + d.QuoteIdent(primaryKey.Name) + " PRIMARY KEY (" + strings.Join(columns, ", ") + ")"
}

func (d Dialect) foreignKeyDDL(foreignKey dbx.ForeignKeyMeta) string {
	columns := lo.Map(foreignKey.Columns, func(column string, _ int) string {
		return d.QuoteIdent(column)
	})
	targetColumns := lo.Map(foreignKey.TargetColumns, func(column string, _ int) string {
		return d.QuoteIdent(column)
	})
	parts := collectionx.NewList[string]()
	parts.Add("CONSTRAINT " + d.QuoteIdent(foreignKey.Name))
	parts.Add("FOREIGN KEY (" + strings.Join(columns, ", ") + ")")
	parts.Add("REFERENCES " + d.QuoteIdent(foreignKey.TargetTable) + " (" + strings.Join(targetColumns, ", ") + ")")
	if foreignKey.OnDelete != "" {
		parts.Add("ON DELETE " + string(foreignKey.OnDelete))
	}
	if foreignKey.OnUpdate != "" {
		parts.Add("ON UPDATE " + string(foreignKey.OnUpdate))
	}
	return strings.Join(parts.Values(), " ")
}

func (d Dialect) checkDDL(check dbx.CheckMeta) string {
	return "CONSTRAINT " + d.QuoteIdent(check.Name) + " CHECK (" + check.Expression + ")"
}

func postgresType(column dbx.ColumnMeta) string {
	if column.SQLType != "" {
		return column.SQLType
	}
	if column.GoType == nil {
		return "TEXT"
	}

	typ := dereferencePostgresType(column.GoType)
	if isPostgresTimeType(typ) {
		return "TIMESTAMPTZ"
	}
	if isPostgresBlobType(typ) {
		return "BYTEA"
	}
	if mapped, ok := postgresKindType(typ.Kind()); ok {
		return mapped
	}
	return fallbackPostgresType(typ)
}

func postgresKindType(kind reflect.Kind) (string, bool) {
	switch {
	case kind == reflect.Bool:
		return "BOOLEAN", true
	case slices.Contains(postgresIntKinds, kind):
		return "INTEGER", true
	case kind == reflect.Int64:
		return "BIGINT", true
	case slices.Contains(postgresUnsignedIntKinds, kind):
		return "INTEGER", true
	case kind == reflect.Uint64:
		return "BIGINT", true
	case kind == reflect.Float32:
		return "REAL", true
	case kind == reflect.Float64:
		return "DOUBLE PRECISION", true
	case kind == reflect.String:
		return "TEXT", true
	default:
		return "", false
	}
}

func dereferencePostgresType(typ reflect.Type) reflect.Type {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ
}

func isPostgresTimeType(typ reflect.Type) bool {
	return typ.PkgPath() == "time" && typ.Name() == "Time"
}

func isPostgresBlobType(typ reflect.Type) bool {
	return typ.Kind() == reflect.Slice && typ.Elem().Kind() == reflect.Uint8
}

func fallbackPostgresType(typ reflect.Type) string {
	if name := typ.Name(); name != "" {
		return strings.ToUpper(name)
	}
	return "TEXT"
}

func parseIndexColumns(definition string) []string {
	start := strings.Index(definition, "(")
	end := strings.LastIndex(definition, ")")
	if start < 0 || end <= start {
		return nil
	}

	parts := strings.Split(definition[start+1:end], ",")
	return lo.Compact(lo.Map(parts, func(part string, _ int) string {
		return strings.TrimSpace(strings.Trim(part, `"`))
	}))
}

func singlePrimaryKeyColumn(primaryKey *dbx.PrimaryKeyMeta) string {
	if primaryKey == nil || len(primaryKey.Columns) != 1 {
		return ""
	}
	return primaryKey.Columns[0]
}

func referentialAction(value string) dbx.ReferentialAction {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case string(dbx.ReferentialCascade):
		return dbx.ReferentialCascade
	case string(dbx.ReferentialRestrict):
		return dbx.ReferentialRestrict
	case string(dbx.ReferentialSetNull):
		return dbx.ReferentialSetNull
	case string(dbx.ReferentialSetDefault):
		return dbx.ReferentialSetDefault
	case string(dbx.ReferentialNoAction):
		return dbx.ReferentialNoAction
	default:
		return ""
	}
}

var _ dbx.SchemaDialect = Dialect{}
