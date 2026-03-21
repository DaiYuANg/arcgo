package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/DaiYuANg/arcgo/dbx/dialect"
	"github.com/samber/lo"
)

type Dialect struct{}

func (Dialect) Name() string         { return "mysql" }
func (Dialect) BindVar(_ int) string { return "?" }
func (Dialect) QuoteIdent(ident string) string {
	return "`" + strings.ReplaceAll(ident, "`", "``") + "`"
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
	return fmt.Sprintf("LIMIT 18446744073709551615 OFFSET %d", *offset), nil
}

func (Dialect) QueryFeatures() dialect.QueryFeatures {
	return dialect.DefaultQueryFeatures("mysql")
}

func (d Dialect) BuildCreateTable(spec dbx.TableSpec) (dbx.BoundQuery, error) {
	parts := collectionx.NewListWithCapacity[string](len(spec.Columns) + len(spec.ForeignKeys) + len(spec.Checks) + 1)
	inlinePrimaryKey := singlePrimaryKeyColumn(spec.PrimaryKey)
	for _, column := range spec.Columns {
		parts.Add(d.columnDDL(column, inlinePrimaryKey == column.Name, false))
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
	return dbx.BoundQuery{SQL: "ALTER TABLE " + d.QuoteIdent(table) + " ADD COLUMN " + d.columnDDL(column, false, true)}, nil
}

func (d Dialect) BuildCreateIndex(index dbx.IndexMeta) (dbx.BoundQuery, error) {
	columns := lo.Map(index.Columns, func(column string, _ int) string {
		return d.QuoteIdent(column)
	})
	prefix := "CREATE INDEX "
	if index.Unique {
		prefix = "CREATE UNIQUE INDEX "
	}
	return dbx.BoundQuery{SQL: prefix + d.QuoteIdent(index.Name) + " ON " + d.QuoteIdent(index.Table) + " (" + strings.Join(columns, ", ") + ")"}, nil
}

func (d Dialect) BuildAddForeignKey(table string, foreignKey dbx.ForeignKeyMeta) (dbx.BoundQuery, error) {
	return dbx.BoundQuery{SQL: "ALTER TABLE " + d.QuoteIdent(table) + " ADD " + d.foreignKeyDDL(foreignKey)}, nil
}

func (d Dialect) BuildAddCheck(table string, check dbx.CheckMeta) (dbx.BoundQuery, error) {
	return dbx.BoundQuery{SQL: "ALTER TABLE " + d.QuoteIdent(table) + " ADD " + d.checkDDL(check)}, nil
}

func (d Dialect) InspectTable(ctx context.Context, executor dbx.Executor, table string) (dbx.TableState, error) {
	existsRows, err := executor.QueryContext(ctx, "SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", table)
	if err != nil {
		return dbx.TableState{}, err
	}
	exists := existsRows.Next()
	_ = existsRows.Close()
	if !exists {
		return dbx.TableState{Name: table, Exists: false}, nil
	}

	columnsRows, err := executor.QueryContext(ctx, "SELECT column_name, column_type, is_nullable, column_default, column_key, extra FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? ORDER BY ordinal_position", table)
	if err != nil {
		return dbx.TableState{}, err
	}
	defer columnsRows.Close()

	columns := collectionx.NewList[dbx.ColumnState]()
	primaryColumns := collectionx.NewList[string]()
	for columnsRows.Next() {
		var name, columnType, isNullable, columnKey, extra string
		var defaultValue sql.NullString
		if scanErr := columnsRows.Scan(&name, &columnType, &isNullable, &defaultValue, &columnKey, &extra); scanErr != nil {
			return dbx.TableState{}, scanErr
		}
		isPrimary := strings.EqualFold(columnKey, "PRI")
		if isPrimary {
			primaryColumns.Add(name)
		}
		columns.Add(dbx.ColumnState{Name: name, Type: columnType, Nullable: strings.EqualFold(isNullable, "YES"), PrimaryKey: isPrimary, AutoIncrement: strings.Contains(strings.ToLower(extra), "auto_increment"), DefaultValue: defaultValue.String})
	}
	if err := columnsRows.Err(); err != nil {
		return dbx.TableState{}, err
	}

	var primaryKey *dbx.PrimaryKeyState
	if primaryColumns.Len() > 0 {
		primaryKey = &dbx.PrimaryKeyState{Name: "PRIMARY", Columns: primaryColumns.Values()}
	}

	indexRows, err := executor.QueryContext(ctx, "SELECT index_name, non_unique, column_name FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = ? ORDER BY index_name, seq_in_index", table)
	if err != nil {
		return dbx.TableState{}, err
	}
	defer indexRows.Close()

	groups := collectionx.NewOrderedMap[string, dbx.IndexState]()
	for indexRows.Next() {
		var name, column string
		var nonUnique int
		if scanErr := indexRows.Scan(&name, &nonUnique, &column); scanErr != nil {
			return dbx.TableState{}, scanErr
		}
		if strings.EqualFold(name, "PRIMARY") {
			continue
		}
		current, ok := groups.Get(name)
		if !ok {
			current = dbx.IndexState{Name: name, Columns: make([]string, 0, 2), Unique: nonUnique == 0}
		}
		current.Columns = append(current.Columns, column)
		groups.Set(name, current)
	}
	if err := indexRows.Err(); err != nil {
		return dbx.TableState{}, err
	}
	indexes := collectionx.NewListWithCapacity[dbx.IndexState](groups.Len())
	groups.Range(func(_ string, value dbx.IndexState) bool {
		indexes.Add(value)
		return true
	})

	foreignKeyRows, err := executor.QueryContext(ctx, "SELECT kcu.constraint_name, kcu.column_name, kcu.referenced_table_name, kcu.referenced_column_name, rc.UPDATE_RULE, rc.DELETE_RULE FROM information_schema.key_column_usage kcu JOIN information_schema.table_constraints tc ON kcu.constraint_name = tc.constraint_name AND kcu.table_schema = tc.table_schema AND kcu.table_name = tc.table_name LEFT JOIN information_schema.referential_constraints rc ON kcu.constraint_name = rc.constraint_name AND kcu.table_schema = rc.constraint_schema WHERE kcu.table_schema = DATABASE() AND kcu.table_name = ? AND tc.constraint_type = 'FOREIGN KEY' ORDER BY kcu.constraint_name, kcu.ordinal_position", table)
	if err != nil {
		return dbx.TableState{}, err
	}
	defer foreignKeyRows.Close()
	foreignKeysByName := collectionx.NewOrderedMap[string, dbx.ForeignKeyState]()
	for foreignKeyRows.Next() {
		var name, column, targetTable, targetColumn string
		var updateRule, deleteRule sql.NullString
		if scanErr := foreignKeyRows.Scan(&name, &column, &targetTable, &targetColumn, &updateRule, &deleteRule); scanErr != nil {
			return dbx.TableState{}, scanErr
		}
		state, ok := foreignKeysByName.Get(name)
		if !ok {
			state = dbx.ForeignKeyState{Name: name, TargetTable: targetTable, Columns: make([]string, 0, 1), TargetColumns: make([]string, 0, 1), OnDelete: referentialAction(deleteRule.String), OnUpdate: referentialAction(updateRule.String)}
		}
		state.Columns = append(state.Columns, column)
		state.TargetColumns = append(state.TargetColumns, targetColumn)
		foreignKeysByName.Set(name, state)
	}
	if err := foreignKeyRows.Err(); err != nil {
		return dbx.TableState{}, err
	}
	foreignKeys := collectionx.NewListWithCapacity[dbx.ForeignKeyState](foreignKeysByName.Len())
	foreignKeysByName.Range(func(_ string, value dbx.ForeignKeyState) bool {
		foreignKeys.Add(value)
		return true
	})

	checkRows, err := executor.QueryContext(ctx, "SELECT tc.constraint_name, cc.check_clause FROM information_schema.table_constraints tc JOIN information_schema.check_constraints cc ON tc.constraint_name = cc.constraint_name AND tc.constraint_schema = cc.constraint_schema WHERE tc.table_schema = DATABASE() AND tc.table_name = ? AND tc.constraint_type = 'CHECK' ORDER BY tc.constraint_name", table)
	if err != nil {
		return dbx.TableState{}, err
	}
	defer checkRows.Close()
	checks := collectionx.NewList[dbx.CheckState]()
	for checkRows.Next() {
		var name, clause string
		if scanErr := checkRows.Scan(&name, &clause); scanErr != nil {
			return dbx.TableState{}, scanErr
		}
		checks.Add(dbx.CheckState{Name: name, Expression: clause})
	}
	if err := checkRows.Err(); err != nil {
		return dbx.TableState{}, err
	}

	return dbx.TableState{Exists: true, Name: table, Columns: columns.Values(), Indexes: indexes.Values(), PrimaryKey: primaryKey, ForeignKeys: foreignKeys.Values(), Checks: checks.Values()}, nil
}

func (Dialect) NormalizeType(value string) string {
	typeName := strings.ToLower(strings.TrimSpace(value))
	if strings.HasPrefix(typeName, "tinyint(1)") || typeName == "boolean" || typeName == "bool" {
		return "boolean"
	}
	if index := strings.Index(typeName, "("); index >= 0 {
		typeName = typeName[:index]
	}
	switch typeName {
	case "int", "integer", "smallint", "mediumint", "tinyint":
		return "integer"
	case "bigint":
		return "bigint"
	case "float", "real":
		return "real"
	case "double", "decimal", "numeric":
		return "double"
	case "varchar", "char", "text", "tinytext", "mediumtext", "longtext":
		return "text"
	case "blob", "tinyblob", "mediumblob", "longblob", "binary", "varbinary":
		return "blob"
	case "timestamp", "datetime":
		return "timestamp"
	default:
		return typeName
	}
}

func (d Dialect) columnDDL(column dbx.ColumnMeta, inlinePrimaryKey bool, includeReference bool) string {
	parts := collectionx.NewList[string]()
	parts.Add(d.QuoteIdent(column.Name))
	typeName := column.SQLType
	if typeName == "" {
		typeName = mysqlType(column)
	}
	parts.Add(typeName)
	if column.AutoIncrement {
		parts.Add("AUTO_INCREMENT")
	}
	if inlinePrimaryKey {
		parts.Add("PRIMARY KEY")
	}
	if !column.Nullable && !inlinePrimaryKey {
		parts.Add("NOT NULL")
	}
	if column.DefaultValue != "" && !column.AutoIncrement {
		parts.Add("DEFAULT " + column.DefaultValue)
	}
	if includeReference && column.References != nil {
		parts.Add("REFERENCES " + d.QuoteIdent(column.References.TargetTable) + " (" + d.QuoteIdent(column.References.TargetColumn) + ")")
		if column.References.OnDelete != "" {
			parts.Add("ON DELETE " + string(column.References.OnDelete))
		}
		if column.References.OnUpdate != "" {
			parts.Add("ON UPDATE " + string(column.References.OnUpdate))
		}
	}
	return strings.Join(parts.Values(), " ")
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
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return "INT"
	case reflect.Int64:
		return "BIGINT"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return "INT UNSIGNED"
	case reflect.Uint64:
		return "BIGINT UNSIGNED"
	case reflect.Float32:
		return "FLOAT"
	case reflect.Float64:
		return "DOUBLE"
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
