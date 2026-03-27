package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/DaiYuANg/arcgo/dbx/dialect"
	"github.com/samber/lo"
)

const (
	sqliteTableExistsQuery = "SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?"
	sqliteCreateSQLQuery   = "SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?"
)

var (
	sqliteIntegerKinds = []reflect.Kind{
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
	}
	sqliteRealKinds = []reflect.Kind{reflect.Float32, reflect.Float64}
)

// Dialect implements SQLite rendering and schema inspection.
type Dialect struct{}

// New returns a SQLite dialect implementation.
func New() Dialect { return Dialect{} }

// Name returns the dialect name.
func (Dialect) Name() string { return "sqlite" }

// BindVar returns the bind placeholder for a parameter index.
func (Dialect) BindVar(_ int) string { return "?" }

// QuoteIdent quotes an identifier for SQLite.
func (Dialect) QuoteIdent(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}

// RenderLimitOffset renders a LIMIT/OFFSET clause for SQLite.
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
	return fmt.Sprintf("LIMIT -1 OFFSET %d", *offset), nil
}

// QueryFeatures returns the supported query feature set.
func (Dialect) QueryFeatures() dialect.QueryFeatures {
	return dialect.DefaultQueryFeatures("sqlite")
}

// BuildCreateTable builds a CREATE TABLE statement.
func (d Dialect) BuildCreateTable(spec dbx.TableSpec) (dbx.BoundQuery, error) {
	parts := collectionx.NewListWithCapacity[string](len(spec.Columns) + len(spec.ForeignKeys) + len(spec.Checks) + 1)
	inlinePrimaryKey := singlePrimaryKeyColumn(spec.PrimaryKey)

	for i := range spec.Columns {
		column := spec.Columns[i]
		ddl, err := d.columnDDL(column, columnDDLConfig{
			AllowAutoIncrement: true,
			InlinePrimaryKey:   inlinePrimaryKey == column.Name,
		})
		if err != nil {
			return dbx.BoundQuery{}, fmt.Errorf("build sqlite column ddl: %w", err)
		}
		parts.Add(ddl)
	}

	if spec.PrimaryKey != nil && len(spec.PrimaryKey.Columns) > 1 {
		parts.Add(d.primaryKeyDDL(*spec.PrimaryKey))
	}

	for i := range spec.ForeignKeys {
		parts.Add(d.foreignKeyDDL(spec.ForeignKeys[i]))
	}
	for i := range spec.Checks {
		parts.Add(d.checkDDL(spec.Checks[i]))
	}

	return dbx.BoundQuery{
		SQL: "CREATE TABLE IF NOT EXISTS " + d.QuoteIdent(spec.Name) + " (" + strings.Join(parts.Values(), ", ") + ")",
	}, nil
}

// BuildAddColumn builds an ALTER TABLE ADD COLUMN statement.
func (d Dialect) BuildAddColumn(table string, column dbx.ColumnMeta) (dbx.BoundQuery, error) {
	if column.PrimaryKey {
		return dbx.BoundQuery{}, fmt.Errorf("dbx/sqlite: cannot add primary key column %s with ALTER TABLE", column.Name)
	}

	ddl, err := d.columnDDL(column, columnDDLConfig{IncludeReference: true})
	if err != nil {
		return dbx.BoundQuery{}, fmt.Errorf("build sqlite column ddl: %w", err)
	}

	return dbx.BoundQuery{
		SQL: "ALTER TABLE " + d.QuoteIdent(table) + " ADD COLUMN " + ddl,
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

// BuildAddForeignKey reports that SQLite foreign keys require a table rebuild.
func (Dialect) BuildAddForeignKey(string, dbx.ForeignKeyMeta) (dbx.BoundQuery, error) {
	return dbx.BoundQuery{}, errors.New("dbx/sqlite: adding foreign keys requires table rebuild")
}

// BuildAddCheck reports that SQLite check constraints require a table rebuild.
func (Dialect) BuildAddCheck(string, dbx.CheckMeta) (dbx.BoundQuery, error) {
	return dbx.BoundQuery{}, errors.New("dbx/sqlite: adding check constraints requires table rebuild")
}

// InspectTable inspects a SQLite table definition from PRAGMA metadata.
func (d Dialect) InspectTable(ctx context.Context, executor dbx.Executor, table string) (dbx.TableState, error) {
	exists, err := inspectSQLiteTableExists(ctx, executor, table)
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

	checks, autoincrementColumns, err := inspectSQLiteCreateMetadata(ctx, executor, table)
	if err != nil {
		return dbx.TableState{}, err
	}

	return dbx.TableState{
		Exists:      true,
		Name:        table,
		Columns:     markSQLiteAutoincrementColumns(columns, autoincrementColumns),
		Indexes:     indexes,
		PrimaryKey:  primaryKey,
		ForeignKeys: foreignKeys,
		Checks:      checks,
	}, nil
}

// NormalizeType normalizes database type names into dbx logical types.
func (Dialect) NormalizeType(value string) string {
	typeName := strings.ToUpper(strings.TrimSpace(value))
	switch {
	case strings.Contains(typeName, "INT"):
		return "INTEGER"
	case strings.Contains(typeName, "CHAR"), strings.Contains(typeName, "CLOB"), strings.Contains(typeName, "TEXT"):
		return "TEXT"
	case strings.Contains(typeName, "BLOB"):
		return "BLOB"
	case strings.Contains(typeName, "REAL"), strings.Contains(typeName, "FLOA"), strings.Contains(typeName, "DOUBLE"):
		return "REAL"
	case strings.Contains(typeName, "BOOL"):
		return "BOOLEAN"
	case strings.Contains(typeName, "TIMESTAMP"), strings.Contains(typeName, "DATETIME"):
		return "TIMESTAMP"
	default:
		return typeName
	}
}

type columnDDLConfig struct {
	AllowAutoIncrement bool
	InlinePrimaryKey   bool
	IncludeReference   bool
}

func inspectSQLiteTableExists(ctx context.Context, executor dbx.Executor, table string) (exists bool, resultErr error) {
	const action = "inspect sqlite table existence"

	rows, err := querySQLiteRows(ctx, executor, action, sqliteTableExistsQuery, table)
	if err != nil {
		return false, err
	}
	defer func() {
		if closeErr := closeSQLiteRows(action, rows); closeErr != nil && resultErr == nil {
			resultErr = closeErr
		}
	}()

	exists = rows.Next()
	if rowsErr := sqliteRowsError(action, rows); rowsErr != nil {
		return false, rowsErr
	}

	return exists, nil
}

func (d Dialect) inspectColumns(ctx context.Context, executor dbx.Executor, table string) (_ []dbx.ColumnState, _ *dbx.PrimaryKeyState, resultErr error) {
	const action = "inspect sqlite columns"

	rows, err := querySQLiteRows(ctx, executor, action, "PRAGMA table_info("+d.QuoteIdent(table)+")")
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if closeErr := closeSQLiteRows(action, rows); closeErr != nil && resultErr == nil {
			resultErr = closeErr
		}
	}()

	columns := make([]dbx.ColumnState, 0, 8)
	primaryPositions := make(map[int]string, 2)
	for rows.Next() {
		column, primaryPosition, scanErr := scanSQLiteColumn(rows)
		if scanErr != nil {
			return nil, nil, scanErr
		}
		columns = append(columns, column)
		if primaryPosition > 0 {
			primaryPositions[primaryPosition] = column.Name
		}
	}

	if rowsErr := sqliteRowsError(action, rows); rowsErr != nil {
		return nil, nil, rowsErr
	}

	return columns, sqlitePrimaryKeyState(primaryPositions), nil
}

func (d Dialect) inspectIndexes(ctx context.Context, executor dbx.Executor, table string) (_ []dbx.IndexState, resultErr error) {
	const action = "inspect sqlite indexes"

	rows, err := querySQLiteRows(ctx, executor, action, "PRAGMA index_list("+d.QuoteIdent(table)+")")
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := closeSQLiteRows(action, rows); closeErr != nil && resultErr == nil {
			resultErr = closeErr
		}
	}()

	indexes := make([]dbx.IndexState, 0, 4)
	for rows.Next() {
		index, skip, indexErr := d.loadSQLiteIndex(ctx, executor, rows)
		if indexErr != nil {
			return nil, indexErr
		}
		if !skip {
			indexes = append(indexes, index)
		}
	}

	if rowsErr := sqliteRowsError(action, rows); rowsErr != nil {
		return nil, rowsErr
	}

	return indexes, nil
}

func (d Dialect) loadSQLiteIndex(ctx context.Context, executor dbx.Executor, rows *sql.Rows) (dbx.IndexState, bool, error) {
	name, unique, origin, err := scanSQLiteIndexList(rows)
	if err != nil {
		return dbx.IndexState{}, false, err
	}
	if origin == "pk" {
		return dbx.IndexState{}, true, nil
	}

	index, err := d.inspectIndex(ctx, executor, name, unique)
	if err != nil {
		return dbx.IndexState{}, false, err
	}
	return index, false, nil
}

func (d Dialect) inspectIndex(ctx context.Context, executor dbx.Executor, name string, unique bool) (dbx.IndexState, error) {
	columns, err := d.inspectIndexColumns(ctx, executor, name)
	if err != nil {
		return dbx.IndexState{}, err
	}
	return dbx.IndexState{Name: name, Columns: columns, Unique: unique}, nil
}

func (d Dialect) inspectIndexColumns(ctx context.Context, executor dbx.Executor, name string) (_ []string, resultErr error) {
	const action = "inspect sqlite index columns"

	rows, err := querySQLiteRows(ctx, executor, action, "PRAGMA index_info("+d.QuoteIdent(name)+")")
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := closeSQLiteRows(action, rows); closeErr != nil && resultErr == nil {
			resultErr = closeErr
		}
	}()

	columns := make([]string, 0, 2)
	for rows.Next() {
		column, scanErr := scanSQLiteIndexColumn(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		columns = append(columns, column)
	}

	if rowsErr := sqliteRowsError(action, rows); rowsErr != nil {
		return nil, rowsErr
	}

	return columns, nil
}

func (d Dialect) inspectForeignKeys(ctx context.Context, executor dbx.Executor, table string) (_ []dbx.ForeignKeyState, resultErr error) {
	const action = "inspect sqlite foreign keys"

	rows, err := querySQLiteRows(ctx, executor, action, "PRAGMA foreign_key_list("+d.QuoteIdent(table)+")")
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := closeSQLiteRows(action, rows); closeErr != nil && resultErr == nil {
			resultErr = closeErr
		}
	}()

	groups := collectionx.NewOrderedMap[int, dbx.ForeignKeyState]()
	for rows.Next() {
		id, state, scanErr := scanSQLiteForeignKey(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		appendSQLiteForeignKey(groups, id, state)
	}

	if rowsErr := sqliteRowsError(action, rows); rowsErr != nil {
		return nil, rowsErr
	}

	foreignKeys := make([]dbx.ForeignKeyState, 0, groups.Len())
	groups.Range(func(_ int, value dbx.ForeignKeyState) bool {
		foreignKeys = append(foreignKeys, value)
		return true
	})
	return foreignKeys, nil
}

func inspectSQLiteCreateMetadata(ctx context.Context, executor dbx.Executor, table string) (_ []dbx.CheckState, _ map[string]struct{}, resultErr error) {
	const action = "inspect sqlite create metadata"

	rows, err := querySQLiteRows(ctx, executor, action, sqliteCreateSQLQuery, table)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if closeErr := closeSQLiteRows(action, rows); closeErr != nil && resultErr == nil {
			resultErr = closeErr
		}
	}()

	checks := make([]dbx.CheckState, 0, 2)
	autoincrementColumns := make(map[string]struct{}, 1)
	for rows.Next() {
		createSQL, scanErr := scanSQLiteCreateSQL(rows)
		if scanErr != nil {
			return nil, nil, scanErr
		}

		for _, column := range parseCreateTableAutoincrementColumns(createSQL) {
			autoincrementColumns[column] = struct{}{}
		}
		checks = append(checks, parseCreateTableChecks(createSQL)...)
	}

	if rowsErr := sqliteRowsError(action, rows); rowsErr != nil {
		return nil, nil, rowsErr
	}

	return checks, autoincrementColumns, nil
}

func scanSQLiteColumn(rows *sql.Rows) (dbx.ColumnState, int, error) {
	var cid int
	var name string
	var typeName string
	var notNull int
	var defaultValue sql.NullString
	var primaryPosition int

	if err := rows.Scan(&cid, &name, &typeName, &notNull, &defaultValue, &primaryPosition); err != nil {
		return dbx.ColumnState{}, 0, fmt.Errorf("scan sqlite column: %w", err)
	}

	return dbx.ColumnState{
		Name:         name,
		Type:         typeName,
		Nullable:     notNull == 0,
		PrimaryKey:   primaryPosition > 0,
		DefaultValue: defaultValue.String,
	}, primaryPosition, nil
}

func scanSQLiteIndexList(rows *sql.Rows) (string, bool, string, error) {
	var seq int
	var name string
	var unique int
	var origin string
	var partial int

	if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
		return "", false, "", fmt.Errorf("scan sqlite index list: %w", err)
	}

	return name, unique == 1, origin, nil
}

func scanSQLiteIndexColumn(rows *sql.Rows) (string, error) {
	var seqno int
	var cid int
	var column string

	if err := rows.Scan(&seqno, &cid, &column); err != nil {
		return "", fmt.Errorf("scan sqlite index column: %w", err)
	}

	return column, nil
}

func scanSQLiteForeignKey(rows *sql.Rows) (int, dbx.ForeignKeyState, error) {
	var id int
	var seq int
	var targetTable string
	var from string
	var to string
	var onUpdate string
	var onDelete string
	var match string

	if err := rows.Scan(&id, &seq, &targetTable, &from, &to, &onUpdate, &onDelete, &match); err != nil {
		return 0, dbx.ForeignKeyState{}, fmt.Errorf("scan sqlite foreign key: %w", err)
	}

	return id, dbx.ForeignKeyState{
		TargetTable:   targetTable,
		Columns:       []string{from},
		TargetColumns: []string{to},
		OnDelete:      referentialAction(onDelete),
		OnUpdate:      referentialAction(onUpdate),
	}, nil
}

func scanSQLiteCreateSQL(rows *sql.Rows) (string, error) {
	var createSQL sql.NullString

	if err := rows.Scan(&createSQL); err != nil {
		return "", fmt.Errorf("scan sqlite create sql: %w", err)
	}
	return createSQL.String, nil
}

func sqlitePrimaryKeyState(positions map[int]string) *dbx.PrimaryKeyState {
	if len(positions) == 0 {
		return nil
	}

	keys := make([]int, 0, len(positions))
	for position := range positions {
		keys = append(keys, position)
	}
	slices.Sort(keys)

	columns := make([]string, 0, len(keys))
	for _, position := range keys {
		columns = append(columns, positions[position])
	}

	return &dbx.PrimaryKeyState{Columns: columns}
}

func appendSQLiteForeignKey(groups collectionx.OrderedMap[int, dbx.ForeignKeyState], id int, state dbx.ForeignKeyState) {
	current, ok := groups.Get(id)
	if !ok {
		groups.Set(id, state)
		return
	}
	current.Columns = append(current.Columns, state.Columns...)
	current.TargetColumns = append(current.TargetColumns, state.TargetColumns...)
	groups.Set(id, current)
}

func querySQLiteRows(ctx context.Context, executor dbx.Executor, action, query string, args ...any) (*sql.Rows, error) {
	rows, err := executor.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", action, err)
	}
	return rows, nil
}

func closeSQLiteRows(action string, rows *sql.Rows) error {
	if rows == nil {
		return nil
	}
	if closeErr := rows.Close(); closeErr != nil {
		return fmt.Errorf("%s: close rows: %w", action, closeErr)
	}
	return nil
}

func sqliteRowsError(action string, rows *sql.Rows) error {
	if err := rows.Err(); err != nil {
		return fmt.Errorf("%s: rows err: %w", action, err)
	}
	return nil
}

func markSQLiteAutoincrementColumns(columns []dbx.ColumnState, autoincrementColumns map[string]struct{}) []dbx.ColumnState {
	for i := range columns {
		if _, ok := autoincrementColumns[columns[i].Name]; ok {
			columns[i].AutoIncrement = true
		}
	}
	return columns
}

func (d Dialect) columnDDL(column dbx.ColumnMeta, config columnDDLConfig) (string, error) {
	parts := make([]string, 0, 5)
	parts = append(parts, d.QuoteIdent(column.Name))

	autoIncrementDDL, ok, err := d.sqliteAutoIncrementDDL(column, config)
	if err != nil {
		return "", err
	}
	if ok {
		return strings.Join(append(parts, autoIncrementDDL), " "), nil
	}

	parts = append(parts, resolvedSQLiteType(column))
	parts = append(parts, sqliteColumnConstraintParts(column, config)...)
	parts = append(parts, d.sqliteReferenceParts(column, config.IncludeReference)...)
	return strings.Join(parts, " "), nil
}

func (d Dialect) sqliteAutoIncrementDDL(column dbx.ColumnMeta, config columnDDLConfig) (string, bool, error) {
	if !config.InlinePrimaryKey || !column.AutoIncrement || !config.AllowAutoIncrement {
		return "", false, nil
	}

	typeName := resolvedSQLiteType(column)
	if d.NormalizeType(typeName) != "INTEGER" {
		return "", false, fmt.Errorf("dbx/sqlite: autoincrement requires INTEGER primary key for column %s", column.Name)
	}

	return "INTEGER PRIMARY KEY AUTOINCREMENT", true, nil
}

func sqliteColumnConstraintParts(column dbx.ColumnMeta, config columnDDLConfig) []string {
	parts := make([]string, 0, 3)
	if config.InlinePrimaryKey {
		parts = append(parts, "PRIMARY KEY")
	}
	if !column.Nullable && !config.InlinePrimaryKey {
		parts = append(parts, "NOT NULL")
	}
	if column.DefaultValue != "" {
		parts = append(parts, "DEFAULT "+column.DefaultValue)
	}
	return parts
}

func (d Dialect) sqliteReferenceParts(column dbx.ColumnMeta, includeReference bool) []string {
	if !includeReference || column.References == nil {
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

func resolvedSQLiteType(column dbx.ColumnMeta) string {
	if column.SQLType != "" {
		return column.SQLType
	}
	return sqliteType(column)
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

func sqliteType(column dbx.ColumnMeta) string {
	if column.SQLType != "" {
		return column.SQLType
	}
	if column.GoType == nil {
		return "TEXT"
	}

	typ := dereferenceSQLiteType(column.GoType)
	if isSQLiteTimeType(typ) {
		return "TIMESTAMP"
	}
	if isSQLiteBlobType(typ) {
		return "BLOB"
	}
	if mapped, ok := sqliteKindType(typ.Kind()); ok {
		return mapped
	}
	return fallbackSQLiteType(typ)
}

func sqliteKindType(kind reflect.Kind) (string, bool) {
	switch {
	case kind == reflect.Bool:
		return "BOOLEAN", true
	case slices.Contains(sqliteIntegerKinds, kind):
		return "INTEGER", true
	case slices.Contains(sqliteRealKinds, kind):
		return "REAL", true
	case kind == reflect.String:
		return "TEXT", true
	default:
		return "", false
	}
}

func dereferenceSQLiteType(typ reflect.Type) reflect.Type {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ
}

func isSQLiteTimeType(typ reflect.Type) bool {
	return typ.PkgPath() == "time" && typ.Name() == "Time"
}

func isSQLiteBlobType(typ reflect.Type) bool {
	return typ.Kind() == reflect.Slice && typ.Elem().Kind() == reflect.Uint8
}

func fallbackSQLiteType(typ reflect.Type) string {
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

func parseCreateTableChecks(createSQL string) []dbx.CheckState {
	upper := strings.ToUpper(createSQL)
	checks := make([]dbx.CheckState, 0, 2)

	for offset := 0; ; {
		expression, nextOffset, found := nextSQLiteCheckExpression(createSQL, upper, offset)
		if !found {
			return checks
		}
		if expression != "" {
			checks = append(checks, dbx.CheckState{Expression: expression})
		}
		offset = nextOffset
	}
}

func nextSQLiteCheckExpression(createSQL, upper string, offset int) (string, int, bool) {
	index := strings.Index(upper[offset:], "CHECK")
	if index < 0 {
		return "", 0, false
	}

	index += offset
	start := strings.Index(createSQL[index:], "(")
	if start < 0 {
		return "", index + len("CHECK"), true
	}
	start += index

	end := sqliteMatchingParen(createSQL, start)
	if end < 0 {
		return "", len(createSQL), false
	}

	return strings.TrimSpace(createSQL[start+1 : end]), end + 1, true
}

func sqliteMatchingParen(input string, start int) int {
	depth := 0
	for i := start; i < len(input); i++ {
		switch input[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func parseCreateTableAutoincrementColumns(createSQL string) []string {
	matches := sqliteAutoincrementPattern.FindAllStringSubmatch(createSQL, -1)
	columns := make([]string, 0, len(matches))
	for i := range matches {
		match := matches[i]
		if len(match) >= 2 {
			columns = append(columns, strings.TrimSpace(match[1]))
		}
	}
	return columns
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

var sqliteAutoincrementPattern = regexp.MustCompile(`(?i)"?([a-zA-Z0-9_]+)"?\s+INTEGER\s+PRIMARY\s+KEY\s+AUTOINCREMENT`)

var _ dbx.SchemaDialect = Dialect{}
