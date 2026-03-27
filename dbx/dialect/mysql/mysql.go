package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/DaiYuANg/arcgo/dbx/dialect"
	"github.com/samber/lo"
)

const (
	mysqlTableExistsQuery = "SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?"
	mysqlColumnsQuery     = "SELECT column_name, column_type, is_nullable, column_default, column_key, extra FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? ORDER BY ordinal_position"
	mysqlIndexesQuery     = "SELECT index_name, non_unique, column_name FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = ? ORDER BY index_name, seq_in_index"
	mysqlForeignKeysQuery = "SELECT kcu.constraint_name, kcu.column_name, kcu.referenced_table_name, kcu.referenced_column_name, rc.UPDATE_RULE, rc.DELETE_RULE FROM information_schema.key_column_usage kcu JOIN information_schema.table_constraints tc ON kcu.constraint_name = tc.constraint_name AND kcu.table_schema = tc.table_schema AND kcu.table_name = tc.table_name LEFT JOIN information_schema.referential_constraints rc ON kcu.constraint_name = rc.constraint_name AND kcu.table_schema = rc.constraint_schema WHERE kcu.table_schema = DATABASE() AND kcu.table_name = ? AND tc.constraint_type = 'FOREIGN KEY' ORDER BY kcu.constraint_name, kcu.ordinal_position"
	mysqlChecksQuery      = "SELECT tc.constraint_name, cc.check_clause FROM information_schema.table_constraints tc JOIN information_schema.check_constraints cc ON tc.constraint_name = cc.constraint_name AND tc.constraint_schema = cc.constraint_schema WHERE tc.table_schema = DATABASE() AND tc.table_name = ? AND tc.constraint_type = 'CHECK' ORDER BY tc.constraint_name"
)

var mysqlNormalizedTypes = map[string]string{
	"int":        "integer",
	"integer":    "integer",
	"smallint":   "integer",
	"mediumint":  "integer",
	"tinyint":    "integer",
	"bigint":     "bigint",
	"float":      "real",
	"real":       "real",
	"double":     "double",
	"decimal":    "double",
	"numeric":    "double",
	"varchar":    "text",
	"char":       "text",
	"text":       "text",
	"tinytext":   "text",
	"mediumtext": "text",
	"longtext":   "text",
	"blob":       "blob",
	"tinyblob":   "blob",
	"mediumblob": "blob",
	"longblob":   "blob",
	"binary":     "blob",
	"varbinary":  "blob",
	"timestamp":  "timestamp",
	"datetime":   "timestamp",
}

var (
	mysqlIntKinds         = []reflect.Kind{reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32}
	mysqlUnsignedIntKinds = []reflect.Kind{reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32}
)

// Dialect implements MySQL rendering and schema inspection.
type Dialect struct{}

// New returns a MySQL dialect implementation.
func New() Dialect { return Dialect{} }

// Name returns the dialect name.
func (Dialect) Name() string { return "mysql" }

// BindVar returns the bind placeholder for a parameter index.
func (Dialect) BindVar(_ int) string { return "?" }

// QuoteIdent quotes an identifier for MySQL.
func (Dialect) QuoteIdent(ident string) string {
	return "`" + strings.ReplaceAll(ident, "`", "``") + "`"
}

// RenderLimitOffset renders a LIMIT/OFFSET clause for MySQL.
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
	return fmt.Sprintf("LIMIT 18446744073709551615 OFFSET %d", *offset), nil
}

// QueryFeatures returns the supported query feature set.
func (Dialect) QueryFeatures() dialect.QueryFeatures {
	return dialect.DefaultQueryFeatures("mysql")
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
	prefix := "CREATE INDEX "
	if index.Unique {
		prefix = "CREATE UNIQUE INDEX "
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

// InspectTable inspects a MySQL table definition from information_schema.
func (d Dialect) InspectTable(ctx context.Context, executor dbx.Executor, table string) (dbx.TableState, error) {
	exists, err := inspectMySQLTableExists(ctx, executor, table)
	if err != nil {
		return dbx.TableState{}, err
	}
	if !exists {
		return dbx.TableState{Name: table, Exists: false}, nil
	}

	columns, primaryKey, err := d.inspectColumns(ctx, executor, table)
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
	typeName := mysqlNormalizedTypeName(value)
	if normalized, ok := mysqlNormalizedTypes[typeName]; ok {
		return normalized
	}
	return typeName
}

func inspectMySQLTableExists(ctx context.Context, executor dbx.Executor, table string) (exists bool, resultErr error) {
	const action = "inspect mysql table existence"

	rows, err := queryMySQLRows(ctx, executor, action, mysqlTableExistsQuery, table)
	if err != nil {
		return false, err
	}
	defer func() {
		if closeErr := closeMySQLRows(action, rows); closeErr != nil && resultErr == nil {
			resultErr = closeErr
		}
	}()

	exists = rows.Next()
	if rowsErr := mysqlRowsError(action, rows); rowsErr != nil {
		return false, rowsErr
	}

	return exists, nil
}

func (d Dialect) inspectColumns(ctx context.Context, executor dbx.Executor, table string) (_ []dbx.ColumnState, _ *dbx.PrimaryKeyState, resultErr error) {
	const action = "inspect mysql columns"

	rows, err := queryMySQLRows(ctx, executor, action, mysqlColumnsQuery, table)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if closeErr := closeMySQLRows(action, rows); closeErr != nil && resultErr == nil {
			resultErr = closeErr
		}
	}()

	columns := make([]dbx.ColumnState, 0, 8)
	primaryColumns := make([]string, 0, 2)
	for rows.Next() {
		column, isPrimary, scanErr := scanMySQLColumn(rows)
		if scanErr != nil {
			return nil, nil, scanErr
		}
		columns = append(columns, column)
		if isPrimary {
			primaryColumns = append(primaryColumns, column.Name)
		}
	}

	if rowsErr := mysqlRowsError(action, rows); rowsErr != nil {
		return nil, nil, rowsErr
	}

	return columns, mysqlPrimaryKeyState(primaryColumns), nil
}

func (d Dialect) inspectIndexes(ctx context.Context, executor dbx.Executor, table string) (_ []dbx.IndexState, resultErr error) {
	const action = "inspect mysql indexes"

	rows, err := queryMySQLRows(ctx, executor, action, mysqlIndexesQuery, table)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := closeMySQLRows(action, rows); closeErr != nil && resultErr == nil {
			resultErr = closeErr
		}
	}()

	groups := collectionx.NewOrderedMap[string, dbx.IndexState]()
	for rows.Next() {
		name, state, scanErr := scanMySQLIndex(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		if strings.EqualFold(name, "PRIMARY") {
			continue
		}
		appendMySQLIndex(groups, name, state)
	}

	if rowsErr := mysqlRowsError(action, rows); rowsErr != nil {
		return nil, rowsErr
	}

	indexes := make([]dbx.IndexState, 0, groups.Len())
	groups.Range(func(_ string, value dbx.IndexState) bool {
		indexes = append(indexes, value)
		return true
	})
	return indexes, nil
}

func (d Dialect) inspectForeignKeys(ctx context.Context, executor dbx.Executor, table string) (_ []dbx.ForeignKeyState, resultErr error) {
	const action = "inspect mysql foreign keys"

	rows, err := queryMySQLRows(ctx, executor, action, mysqlForeignKeysQuery, table)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := closeMySQLRows(action, rows); closeErr != nil && resultErr == nil {
			resultErr = closeErr
		}
	}()

	groups := collectionx.NewOrderedMap[string, dbx.ForeignKeyState]()
	for rows.Next() {
		name, state, scanErr := scanMySQLForeignKey(rows)
		if scanErr != nil {
			return nil, scanErr
		}

		current, ok := groups.Get(name)
		if !ok {
			groups.Set(name, state)
			continue
		}

		current.Columns = append(current.Columns, state.Columns...)
		current.TargetColumns = append(current.TargetColumns, state.TargetColumns...)
		groups.Set(name, current)
	}

	if rowsErr := mysqlRowsError(action, rows); rowsErr != nil {
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
	const action = "inspect mysql checks"

	rows, err := queryMySQLRows(ctx, executor, action, mysqlChecksQuery, table)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := closeMySQLRows(action, rows); closeErr != nil && resultErr == nil {
			resultErr = closeErr
		}
	}()

	checks := make([]dbx.CheckState, 0, 4)
	for rows.Next() {
		check, scanErr := scanMySQLCheck(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		checks = append(checks, check)
	}

	if rowsErr := mysqlRowsError(action, rows); rowsErr != nil {
		return nil, rowsErr
	}

	return checks, nil
}

func scanMySQLColumn(rows *sql.Rows) (dbx.ColumnState, bool, error) {
	var name string
	var columnType string
	var isNullable string
	var columnKey string
	var extra string
	var defaultValue sql.NullString

	if err := rows.Scan(&name, &columnType, &isNullable, &defaultValue, &columnKey, &extra); err != nil {
		return dbx.ColumnState{}, false, fmt.Errorf("scan mysql column: %w", err)
	}

	isPrimary := strings.EqualFold(columnKey, "PRI")
	return dbx.ColumnState{
		Name:          name,
		Type:          columnType,
		Nullable:      strings.EqualFold(isNullable, "YES"),
		PrimaryKey:    isPrimary,
		AutoIncrement: strings.Contains(strings.ToLower(extra), "auto_increment"),
		DefaultValue:  defaultValue.String,
	}, isPrimary, nil
}

func scanMySQLIndex(rows *sql.Rows) (string, dbx.IndexState, error) {
	var name string
	var column string
	var nonUnique int

	if err := rows.Scan(&name, &nonUnique, &column); err != nil {
		return "", dbx.IndexState{}, fmt.Errorf("scan mysql index: %w", err)
	}

	return name, dbx.IndexState{
		Name:    name,
		Columns: []string{column},
		Unique:  nonUnique == 0,
	}, nil
}

func appendMySQLIndex(groups collectionx.OrderedMap[string, dbx.IndexState], name string, state dbx.IndexState) {
	current, ok := groups.Get(name)
	if !ok {
		groups.Set(name, state)
		return
	}
	current.Columns = append(current.Columns, state.Columns...)
	groups.Set(name, current)
}

func scanMySQLForeignKey(rows *sql.Rows) (string, dbx.ForeignKeyState, error) {
	var name string
	var column string
	var targetTable string
	var targetColumn string
	var updateRule sql.NullString
	var deleteRule sql.NullString

	if err := rows.Scan(&name, &column, &targetTable, &targetColumn, &updateRule, &deleteRule); err != nil {
		return "", dbx.ForeignKeyState{}, fmt.Errorf("scan mysql foreign key: %w", err)
	}

	return name, dbx.ForeignKeyState{
		Name:          name,
		TargetTable:   targetTable,
		Columns:       []string{column},
		TargetColumns: []string{targetColumn},
		OnDelete:      referentialAction(deleteRule.String),
		OnUpdate:      referentialAction(updateRule.String),
	}, nil
}

func scanMySQLCheck(rows *sql.Rows) (dbx.CheckState, error) {
	var name string
	var clause string

	if err := rows.Scan(&name, &clause); err != nil {
		return dbx.CheckState{}, fmt.Errorf("scan mysql check: %w", err)
	}

	return dbx.CheckState{Name: name, Expression: clause}, nil
}

func mysqlPrimaryKeyState(columns []string) *dbx.PrimaryKeyState {
	if len(columns) == 0 {
		return nil
	}
	return &dbx.PrimaryKeyState{Name: "PRIMARY", Columns: columns}
}

func queryMySQLRows(ctx context.Context, executor dbx.Executor, action, query string, args ...any) (*sql.Rows, error) {
	rows, err := executor.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", action, err)
	}
	return rows, nil
}

func closeMySQLRows(action string, rows *sql.Rows) error {
	if rows == nil {
		return nil
	}
	if closeErr := rows.Close(); closeErr != nil {
		return fmt.Errorf("%s: close rows: %w", action, closeErr)
	}
	return nil
}

func mysqlRowsError(action string, rows *sql.Rows) error {
	if err := rows.Err(); err != nil {
		return fmt.Errorf("%s: rows err: %w", action, err)
	}
	return nil
}

func mysqlNormalizedTypeName(value string) string {
	typeName := strings.ToLower(strings.TrimSpace(value))
	if strings.HasPrefix(typeName, "tinyint(1)") || typeName == "boolean" || typeName == "bool" {
		return "boolean"
	}

	prefix, _, found := strings.Cut(typeName, "(")
	if found {
		return prefix
	}

	return typeName
}

func (d Dialect) columnDDL(column dbx.ColumnMeta, inlinePrimaryKey, includeReference bool) string {
	parts := []string{
		d.QuoteIdent(column.Name),
		resolvedMySQLType(column),
	}

	parts = append(parts, mysqlColumnConstraintParts(column, inlinePrimaryKey)...)
	if includeReference {
		parts = append(parts, d.mysqlReferenceParts(column)...)
	}

	return strings.Join(parts, " ")
}

func mysqlColumnConstraintParts(column dbx.ColumnMeta, inlinePrimaryKey bool) []string {
	parts := make([]string, 0, 4)
	if column.AutoIncrement {
		parts = append(parts, "AUTO_INCREMENT")
	}
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

func (d Dialect) mysqlReferenceParts(column dbx.ColumnMeta) []string {
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

func resolvedMySQLType(column dbx.ColumnMeta) string {
	if column.SQLType != "" {
		return column.SQLType
	}
	return mysqlType(column)
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

func mysqlType(column dbx.ColumnMeta) string {
	if column.SQLType != "" {
		return column.SQLType
	}
	if column.GoType == nil {
		return "TEXT"
	}

	typ := dereferenceMySQLType(column.GoType)
	if isMySQLTimeType(typ) {
		return "TIMESTAMP"
	}
	if isMySQLBlobType(typ) {
		return "BLOB"
	}
	if mapped, ok := mysqlKindType(typ.Kind()); ok {
		return mapped
	}
	return fallbackMySQLType(typ)
}

func mysqlKindType(kind reflect.Kind) (string, bool) {
	switch {
	case kind == reflect.Bool:
		return "BOOLEAN", true
	case slices.Contains(mysqlIntKinds, kind):
		return "INT", true
	case kind == reflect.Int64:
		return "BIGINT", true
	case slices.Contains(mysqlUnsignedIntKinds, kind):
		return "INT UNSIGNED", true
	case kind == reflect.Uint64:
		return "BIGINT UNSIGNED", true
	case kind == reflect.Float32:
		return "FLOAT", true
	case kind == reflect.Float64:
		return "DOUBLE", true
	case kind == reflect.String:
		return "TEXT", true
	default:
		return "", false
	}
}

func dereferenceMySQLType(typ reflect.Type) reflect.Type {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ
}

func isMySQLTimeType(typ reflect.Type) bool {
	return typ.PkgPath() == "time" && typ.Name() == "Time"
}

func isMySQLBlobType(typ reflect.Type) bool {
	return typ.Kind() == reflect.Slice && typ.Elem().Kind() == reflect.Uint8
}

func fallbackMySQLType(typ reflect.Type) string {
	if name := typ.Name(); name != "" {
		return strings.ToUpper(name)
	}
	return "TEXT"
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
