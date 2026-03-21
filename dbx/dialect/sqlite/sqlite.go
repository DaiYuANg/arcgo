package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/DaiYuANg/arcgo/dbx/dialect"
	"github.com/samber/lo"
)

type Dialect struct{}

func (Dialect) Name() string         { return "sqlite" }
func (Dialect) BindVar(_ int) string { return "?" }
func (Dialect) QuoteIdent(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}

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

func (Dialect) QueryFeatures() dialect.QueryFeatures {
	return dialect.DefaultQueryFeatures("sqlite")
}

func (d Dialect) BuildCreateTable(spec dbx.TableSpec) (dbx.BoundQuery, error) {
	parts := collectionx.NewListWithCapacity[string](len(spec.Columns) + len(spec.ForeignKeys) + len(spec.Checks) + 1)
	inlinePrimaryKey := singlePrimaryKeyColumn(spec.PrimaryKey)
	for _, column := range spec.Columns {
		ddl, err := d.columnDDL(column, columnDDLConfig{AllowAutoIncrement: true, InlinePrimaryKey: inlinePrimaryKey == column.Name})
		if err != nil {
			return dbx.BoundQuery{}, err
		}
		parts.Add(ddl)
	}
	if spec.PrimaryKey != nil && len(spec.PrimaryKey.Columns) > 1 {
		parts.Add(d.primaryKeyDDL(*spec.PrimaryKey))
	}
	for _, foreignKey := range spec.ForeignKeys {
		parts.Add(d.foreignKeyDDL(foreignKey))
	}
	for _, check := range spec.Checks {
		parts.Add(d.checkDDL(check))
	}
	return dbx.BoundQuery{SQL: "CREATE TABLE IF NOT EXISTS " + d.QuoteIdent(spec.Name) + " (" + strings.Join(parts.Values(), ", ") + ")"}, nil
}

func (d Dialect) BuildAddColumn(table string, column dbx.ColumnMeta) (dbx.BoundQuery, error) {
	if column.PrimaryKey {
		return dbx.BoundQuery{}, fmt.Errorf("dbx/sqlite: cannot add primary key column %s with ALTER TABLE", column.Name)
	}
	ddl, err := d.columnDDL(column, columnDDLConfig{IncludeReference: true})
	if err != nil {
		return dbx.BoundQuery{}, err
	}
	return dbx.BoundQuery{SQL: "ALTER TABLE " + d.QuoteIdent(table) + " ADD COLUMN " + ddl}, nil
}

func (d Dialect) BuildCreateIndex(index dbx.IndexMeta) (dbx.BoundQuery, error) {
	columns := lo.Map(index.Columns, func(column string, _ int) string {
		return d.QuoteIdent(column)
	})
	prefix := "CREATE INDEX IF NOT EXISTS "
	if index.Unique {
		prefix = "CREATE UNIQUE INDEX IF NOT EXISTS "
	}
	return dbx.BoundQuery{SQL: prefix + d.QuoteIdent(index.Name) + " ON " + d.QuoteIdent(index.Table) + " (" + strings.Join(columns, ", ") + ")"}, nil
}

func (Dialect) BuildAddForeignKey(string, dbx.ForeignKeyMeta) (dbx.BoundQuery, error) {
	return dbx.BoundQuery{}, fmt.Errorf("dbx/sqlite: adding foreign keys requires table rebuild")
}

func (Dialect) BuildAddCheck(string, dbx.CheckMeta) (dbx.BoundQuery, error) {
	return dbx.BoundQuery{}, fmt.Errorf("dbx/sqlite: adding check constraints requires table rebuild")
}

func (d Dialect) InspectTable(ctx context.Context, executor dbx.Executor, table string) (dbx.TableState, error) {
	existsRows, err := executor.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?", table)
	if err != nil {
		return dbx.TableState{}, err
	}
	exists := existsRows.Next()
	_ = existsRows.Close()
	if !exists {
		return dbx.TableState{Name: table, Exists: false}, nil
	}

	columnsRows, err := executor.QueryContext(ctx, "PRAGMA table_info("+d.QuoteIdent(table)+")")
	if err != nil {
		return dbx.TableState{}, err
	}
	defer columnsRows.Close()

	columns := collectionx.NewList[dbx.ColumnState]()
	primaryPositions := collectionx.NewOrderedMap[int, string]()
	for columnsRows.Next() {
		var cid, notNull, pk int
		var name, typeName string
		var defaultVal sql.NullString
		if scanErr := columnsRows.Scan(&cid, &name, &typeName, &notNull, &defaultVal, &pk); scanErr != nil {
			return dbx.TableState{}, scanErr
		}
		columns.Add(dbx.ColumnState{Name: name, Type: typeName, Nullable: notNull == 0, PrimaryKey: pk > 0, DefaultValue: defaultVal.String})
		if pk > 0 {
			primaryPositions.Set(pk, name)
		}
	}
	if err := columnsRows.Err(); err != nil {
		return dbx.TableState{}, err
	}

	var primaryKey *dbx.PrimaryKeyState
	if primaryPositions.Len() > 0 {
		primaryColumns := collectionx.NewListWithCapacity[string](primaryPositions.Len())
		primaryPositions.Range(func(_ int, value string) bool {
			primaryColumns.Add(value)
			return true
		})
		primaryKey = &dbx.PrimaryKeyState{Columns: primaryColumns.Values()}
	}

	indexListRows, err := executor.QueryContext(ctx, "PRAGMA index_list("+d.QuoteIdent(table)+")")
	if err != nil {
		return dbx.TableState{}, err
	}
	defer indexListRows.Close()

	indexes := collectionx.NewList[dbx.IndexState]()
	for indexListRows.Next() {
		var seq, unique, partial int
		var name, origin string
		if scanErr := indexListRows.Scan(&seq, &name, &unique, &origin, &partial); scanErr != nil {
			return dbx.TableState{}, scanErr
		}
		if origin == "pk" {
			continue
		}
		indexInfoRows, infoErr := executor.QueryContext(ctx, "PRAGMA index_info("+d.QuoteIdent(name)+")")
		if infoErr != nil {
			return dbx.TableState{}, infoErr
		}
		cols := collectionx.NewList[string]()
		for indexInfoRows.Next() {
			var seqno, cid int
			var column string
			if scanErr := indexInfoRows.Scan(&seqno, &cid, &column); scanErr != nil {
				_ = indexInfoRows.Close()
				return dbx.TableState{}, scanErr
			}
			cols.Add(column)
		}
		if err := indexInfoRows.Close(); err != nil {
			return dbx.TableState{}, err
		}
		indexes.Add(dbx.IndexState{Name: name, Columns: cols.Values(), Unique: unique == 1})
	}
	if err := indexListRows.Err(); err != nil {
		return dbx.TableState{}, err
	}

	foreignKeyRows, err := executor.QueryContext(ctx, "PRAGMA foreign_key_list("+d.QuoteIdent(table)+")")
	if err != nil {
		return dbx.TableState{}, err
	}
	defer foreignKeyRows.Close()

	foreignKeysByID := collectionx.NewOrderedMap[int, dbx.ForeignKeyState]()
	for foreignKeyRows.Next() {
		var id, seq int
		var targetTable, from, to, onUpdate, onDelete, match string
		if scanErr := foreignKeyRows.Scan(&id, &seq, &targetTable, &from, &to, &onUpdate, &onDelete, &match); scanErr != nil {
			return dbx.TableState{}, scanErr
		}
		state, ok := foreignKeysByID.Get(id)
		if !ok {
			state = dbx.ForeignKeyState{TargetTable: targetTable, Columns: make([]string, 0, 1), TargetColumns: make([]string, 0, 1), OnDelete: referentialAction(onDelete), OnUpdate: referentialAction(onUpdate)}
		}
		state.Columns = append(state.Columns, from)
		state.TargetColumns = append(state.TargetColumns, to)
		foreignKeysByID.Set(id, state)
	}
	if err := foreignKeyRows.Err(); err != nil {
		return dbx.TableState{}, err
	}
	foreignKeys := collectionx.NewListWithCapacity[dbx.ForeignKeyState](foreignKeysByID.Len())
	foreignKeysByID.Range(func(_ int, value dbx.ForeignKeyState) bool {
		foreignKeys.Add(value)
		return true
	})

	createRows, err := executor.QueryContext(ctx, "SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?", table)
	if err != nil {
		return dbx.TableState{}, err
	}
	defer createRows.Close()
	checks := collectionx.NewList[dbx.CheckState]()
	autoincrementColumns := collectionx.NewSet[string]()
	for createRows.Next() {
		var createSQL sql.NullString
		if scanErr := createRows.Scan(&createSQL); scanErr != nil {
			return dbx.TableState{}, scanErr
		}
		for _, column := range parseCreateTableAutoincrementColumns(createSQL.String) {
			autoincrementColumns.Add(column)
		}
		for _, check := range parseCreateTableChecks(createSQL.String) {
			checks.Add(check)
		}
	}
	if err := createRows.Err(); err != nil {
		return dbx.TableState{}, err
	}

	items := columns.Values()
	for i := range items {
		if autoincrementColumns.Contains(items[i].Name) {
			items[i].AutoIncrement = true
		}
	}

	return dbx.TableState{Exists: true, Name: table, Columns: items, Indexes: indexes.Values(), PrimaryKey: primaryKey, ForeignKeys: foreignKeys.Values(), Checks: checks.Values()}, nil
}

func (Dialect) NormalizeType(value string) string {
	typeName := strings.ToUpper(strings.TrimSpace(value))
	switch {
	case strings.Contains(typeName, "INT"):
		return "INTEGER"
	case strings.Contains(typeName, "CHAR"), strings.Contains(typeName, "CLOB"), strings.Contains(typeName, "TEXT"):
		return "TEXT"
	case strings.Contains(typeName, "BLOB"):
		return "BLOB"
	case strings.Contains(typeName, "REAL"), strings.Contains(typeName, "FLOA"), strings.Contains(typeName, "DOUB"):
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

func (d Dialect) columnDDL(column dbx.ColumnMeta, config columnDDLConfig) (string, error) {
	parts := collectionx.NewList[string]()
	parts.Add(d.QuoteIdent(column.Name))

	typeName := column.SQLType
	if typeName == "" {
		typeName = sqliteType(column)
	}
	if config.InlinePrimaryKey && column.AutoIncrement && config.AllowAutoIncrement {
		if d.NormalizeType(typeName) != "INTEGER" {
			return "", fmt.Errorf("dbx/sqlite: autoincrement requires INTEGER primary key for column %s", column.Name)
		}
		parts.Add("INTEGER PRIMARY KEY AUTOINCREMENT")
		return strings.Join(parts.Values(), " "), nil
	}

	parts.Add(typeName)
	if config.InlinePrimaryKey {
		parts.Add("PRIMARY KEY")
	}
	if !column.Nullable && !config.InlinePrimaryKey {
		parts.Add("NOT NULL")
	}
	if column.DefaultValue != "" {
		parts.Add("DEFAULT " + column.DefaultValue)
	}
	if config.IncludeReference && column.References != nil {
		parts.Add("REFERENCES " + d.QuoteIdent(column.References.TargetTable) + " (" + d.QuoteIdent(column.References.TargetColumn) + ")")
		if column.References.OnDelete != "" {
			parts.Add("ON DELETE " + string(column.References.OnDelete))
		}
		if column.References.OnUpdate != "" {
			parts.Add("ON UPDATE " + string(column.References.OnUpdate))
		}
	}
	return strings.Join(parts.Values(), " "), nil
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
	if column.GoType == nil {
		return "TEXT"
	}
	typ := column.GoType
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.PkgPath() == "time" && typ.Name() == "Time" {
		return "TIMESTAMP"
	}
	switch typ.Kind() {
	case reflect.Bool:
		return "BOOLEAN"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "INTEGER"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "INTEGER"
	case reflect.Float32, reflect.Float64:
		return "REAL"
	case reflect.String:
		return "TEXT"
	case reflect.Slice:
		if typ.Elem().Kind() == reflect.Uint8 {
			return "BLOB"
		}
	}
	return strings.ToUpper(typ.Name())
}

func singlePrimaryKeyColumn(primaryKey *dbx.PrimaryKeyMeta) string {
	if primaryKey == nil || len(primaryKey.Columns) != 1 {
		return ""
	}
	return primaryKey.Columns[0]
}

func parseCreateTableChecks(createSQL string) []dbx.CheckState {
	upper := strings.ToUpper(createSQL)
	checks := collectionx.NewList[dbx.CheckState]()
	for offset := 0; ; {
		index := strings.Index(upper[offset:], "CHECK")
		if index < 0 {
			break
		}
		index += offset
		start := strings.Index(createSQL[index:], "(")
		if start < 0 {
			offset = index + len("CHECK")
			continue
		}
		start += index
		depth := 0
		end := -1
		for i := start; i < len(createSQL); i++ {
			switch createSQL[i] {
			case '(':
				depth++
			case ')':
				depth--
				if depth == 0 {
					end = i
					break
				}
			}
		}
		if end < 0 {
			break
		}
		checks.Add(dbx.CheckState{Expression: strings.TrimSpace(createSQL[start+1 : end])})
		offset = end + 1
	}
	return checks.Values()
}

func parseCreateTableAutoincrementColumns(createSQL string) []string {
	matches := sqliteAutoincrementPattern.FindAllStringSubmatch(createSQL, -1)
	columns := collectionx.NewListWithCapacity[string](len(matches))
	for _, match := range matches {
		if len(match) >= 2 {
			columns.Add(strings.TrimSpace(match[1]))
		}
	}
	return columns.Values()
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
