package postgres

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

func (Dialect) Name() string         { return "postgres" }
func (Dialect) BindVar(n int) string { return "$" + fmt.Sprint(n) }
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
	return fmt.Sprintf("OFFSET %d", *offset), nil
}

func (Dialect) QueryFeatures() dialect.QueryFeatures {
	return dialect.DefaultQueryFeatures("postgres")
}

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
	return dbx.BoundQuery{SQL: "CREATE TABLE IF NOT EXISTS " + d.QuoteIdent(spec.Name) + " (" + strings.Join(parts.Values(), ", ") + ")"}, nil
}

func (d Dialect) BuildAddColumn(table string, column dbx.ColumnMeta) (dbx.BoundQuery, error) {
	return dbx.BoundQuery{SQL: "ALTER TABLE " + d.QuoteIdent(table) + " ADD COLUMN " + d.columnDDL(column, false, true)}, nil
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

func (d Dialect) BuildAddForeignKey(table string, foreignKey dbx.ForeignKeyMeta) (dbx.BoundQuery, error) {
	return dbx.BoundQuery{SQL: "ALTER TABLE " + d.QuoteIdent(table) + " ADD " + d.foreignKeyDDL(foreignKey)}, nil
}

func (d Dialect) BuildAddCheck(table string, check dbx.CheckMeta) (dbx.BoundQuery, error) {
	return dbx.BoundQuery{SQL: "ALTER TABLE " + d.QuoteIdent(table) + " ADD " + d.checkDDL(check)}, nil
}

func (d Dialect) InspectTable(ctx context.Context, executor dbx.Executor, table string) (dbx.TableState, error) {
	existsRows, err := executor.QueryContext(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = $1)", table)
	if err != nil {
		return dbx.TableState{}, err
	}
	defer existsRows.Close()
	var exists bool
	if existsRows.Next() {
		if scanErr := existsRows.Scan(&exists); scanErr != nil {
			return dbx.TableState{}, scanErr
		}
	}
	if !exists {
		return dbx.TableState{Name: table, Exists: false}, nil
	}

	pkRows, err := executor.QueryContext(ctx, "SELECT tc.constraint_name, kcu.column_name FROM information_schema.table_constraints tc JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema AND tc.table_name = kcu.table_name WHERE tc.table_schema = current_schema() AND tc.table_name = $1 AND tc.constraint_type = 'PRIMARY KEY' ORDER BY kcu.ordinal_position", table)
	if err != nil {
		return dbx.TableState{}, err
	}
	defer pkRows.Close()
	primaryColumns := collectionx.NewList[string]()
	primaryName := ""
	pkColumns := collectionx.NewSet[string]()
	for pkRows.Next() {
		var name, column string
		if scanErr := pkRows.Scan(&name, &column); scanErr != nil {
			return dbx.TableState{}, scanErr
		}
		primaryName = name
		primaryColumns.Add(column)
		pkColumns.Add(column)
	}
	if err := pkRows.Err(); err != nil {
		return dbx.TableState{}, err
	}
	var primaryKey *dbx.PrimaryKeyState
	if primaryColumns.Len() > 0 {
		primaryKey = &dbx.PrimaryKeyState{Name: primaryName, Columns: primaryColumns.Values()}
	}

	columnsRows, err := executor.QueryContext(ctx, "SELECT c.column_name, c.udt_name, c.is_nullable, c.column_default, (c.is_identity = 'YES') AS is_identity FROM information_schema.columns c WHERE c.table_schema = current_schema() AND c.table_name = $1 ORDER BY c.ordinal_position", table)
	if err != nil {
		return dbx.TableState{}, err
	}
	defer columnsRows.Close()

	columns := collectionx.NewList[dbx.ColumnState]()
	for columnsRows.Next() {
		var name, udtName, isNullable string
		var defaultValue sql.NullString
		var isIdentity bool
		if scanErr := columnsRows.Scan(&name, &udtName, &isNullable, &defaultValue, &isIdentity); scanErr != nil {
			return dbx.TableState{}, scanErr
		}
		columns.Add(dbx.ColumnState{Name: name, Type: udtName, Nullable: strings.EqualFold(isNullable, "YES"), PrimaryKey: pkColumns.Contains(name), AutoIncrement: isIdentity || strings.Contains(strings.ToLower(defaultValue.String), "nextval"), DefaultValue: defaultValue.String})
	}
	if err := columnsRows.Err(); err != nil {
		return dbx.TableState{}, err
	}

	indexRows, err := executor.QueryContext(ctx, "SELECT indexname, indexdef FROM pg_indexes WHERE schemaname = current_schema() AND tablename = $1", table)
	if err != nil {
		return dbx.TableState{}, err
	}
	defer indexRows.Close()
	indexes := collectionx.NewList[dbx.IndexState]()
	for indexRows.Next() {
		var name, definition string
		if scanErr := indexRows.Scan(&name, &definition); scanErr != nil {
			return dbx.TableState{}, scanErr
		}
		if strings.Contains(strings.ToUpper(definition), "PRIMARY KEY") {
			continue
		}
		indexes.Add(dbx.IndexState{Name: name, Columns: parseIndexColumns(definition), Unique: strings.Contains(strings.ToUpper(definition), "CREATE UNIQUE INDEX")})
	}
	if err := indexRows.Err(); err != nil {
		return dbx.TableState{}, err
	}

	foreignKeyRows, err := executor.QueryContext(ctx, "SELECT tc.constraint_name, kcu.column_name, ccu.table_name, ccu.column_name, rc.update_rule, rc.delete_rule FROM information_schema.table_constraints tc JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema AND tc.table_name = kcu.table_name JOIN information_schema.constraint_column_usage ccu ON tc.constraint_name = ccu.constraint_name AND tc.table_schema = ccu.table_schema JOIN information_schema.referential_constraints rc ON tc.constraint_name = rc.constraint_name AND tc.table_schema = rc.constraint_schema WHERE tc.table_schema = current_schema() AND tc.table_name = $1 AND tc.constraint_type = 'FOREIGN KEY' ORDER BY tc.constraint_name, kcu.ordinal_position", table)
	if err != nil {
		return dbx.TableState{}, err
	}
	defer foreignKeyRows.Close()
	foreignKeysByName := collectionx.NewOrderedMap[string, dbx.ForeignKeyState]()
	for foreignKeyRows.Next() {
		var name, column, targetTable, targetColumn, updateRule, deleteRule string
		if scanErr := foreignKeyRows.Scan(&name, &column, &targetTable, &targetColumn, &updateRule, &deleteRule); scanErr != nil {
			return dbx.TableState{}, scanErr
		}
		state, ok := foreignKeysByName.Get(name)
		if !ok {
			state = dbx.ForeignKeyState{Name: name, TargetTable: targetTable, Columns: make([]string, 0, 1), TargetColumns: make([]string, 0, 1), OnDelete: referentialAction(deleteRule), OnUpdate: referentialAction(updateRule)}
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

	checkRows, err := executor.QueryContext(ctx, "SELECT tc.constraint_name, cc.check_clause FROM information_schema.table_constraints tc JOIN information_schema.check_constraints cc ON tc.constraint_name = cc.constraint_name AND tc.table_schema = cc.constraint_schema WHERE tc.table_schema = current_schema() AND tc.table_name = $1 AND tc.constraint_type = 'CHECK' ORDER BY tc.constraint_name", table)
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
	switch typeName {
	case "int2", "smallint":
		return "smallint"
	case "int4", "integer", "serial", "serial4":
		return "integer"
	case "int8", "bigint", "bigserial", "serial8":
		return "bigint"
	case "float4", "real":
		return "real"
	case "float8", "double precision", "numeric", "decimal":
		return "double"
	case "bool", "boolean":
		return "boolean"
	case "varchar", "bpchar", "text", "citext":
		return "text"
	case "bytea":
		return "blob"
	case "timestamp", "timestamptz", "timestamp with time zone", "timestamp without time zone":
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
		typeName = postgresType(column)
	}
	if column.AutoIncrement {
		parts.Add(typeName + " GENERATED BY DEFAULT AS IDENTITY")
	} else {
		parts.Add(typeName)
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

func postgresType(column dbx.ColumnMeta) string {
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
		return "TIMESTAMPTZ"
	}
	switch typ.Kind() {
	case reflect.Bool:
		return "BOOLEAN"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return "INTEGER"
	case reflect.Int64:
		return "BIGINT"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return "INTEGER"
	case reflect.Uint64:
		return "BIGINT"
	case reflect.Float32:
		return "REAL"
	case reflect.Float64:
		return "DOUBLE PRECISION"
	case reflect.String:
		return "TEXT"
	case reflect.Slice:
		if typ.Elem().Kind() == reflect.Uint8 {
			return "BYTEA"
		}
	}
	return strings.ToUpper(typ.Name())
}

func parseIndexColumns(definition string) []string {
	start := strings.Index(definition, "(")
	end := strings.LastIndex(definition, ")")
	if start < 0 || end <= start {
		return nil
	}
	parts := strings.Split(definition[start+1:end], ",")
	columns := collectionx.NewListWithCapacity[string](len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(strings.Trim(part, `"`))
		if trimmed != "" {
			columns.Add(trimmed)
		}
	}
	return columns.Values()
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
