package dbx

import (
	"context"
	"hash/fnv"
	"strconv"
	"strings"

	atlasmigrate "ariga.io/atlas/sql/migrate"
	atlasmysql "ariga.io/atlas/sql/mysql"
	atlaspostgres "ariga.io/atlas/sql/postgres"
	atlasschema "ariga.io/atlas/sql/schema"
	atlassqlite "ariga.io/atlas/sql/sqlite"
	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/hot"
)

var compiledSchemaCache = hot.NewHotCache[string, *atlasCompiledSchema](hot.LRU, 128).Build()

type atlasCompiledSchema struct {
	schema    *atlasschema.Schema
	tables    collectionx.Map[string, *atlasCompiledTable]
	externals collectionx.Map[string, *atlasschema.Table]
	order     collectionx.List[string]
}

type atlasCompiledTable struct {
	spec              TableSpec
	table             *atlasschema.Table
	columnsByName     collectionx.Map[string, ColumnMeta]
	indexesByName     collectionx.Map[string, IndexMeta]
	indexesByKey      collectionx.Map[string, IndexMeta]
	foreignKeysByName collectionx.Map[string, ForeignKeyMeta]
	foreignKeysByKey  collectionx.Map[string, ForeignKeyMeta]
	checksByName      collectionx.Map[string, CheckMeta]
	checksByExpr      collectionx.Map[string, CheckMeta]
}

func schemaFingerprint(schemas []SchemaResource) string {
	if len(schemas) == 0 {
		return ""
	}
	var buffer renderBuffer
	for _, resource := range schemas {
		writeSchemaFingerprint(&buffer, buildTableSpec(resource.schemaRef()))
	}
	if err := buffer.Err("build schema fingerprint"); err != nil {
		return ""
	}
	return fingerprintString(buffer.String())
}

func writeSchemaFingerprint(buffer *renderBuffer, spec TableSpec) {
	buffer.writeString("T:")
	buffer.writeString(spec.Name)
	buffer.writeString("|")
	spec.Columns.Range(func(_ int, column ColumnMeta) bool {
		writeFingerprintColumn(buffer, column)
		return true
	})
	spec.Indexes.Range(func(_ int, index IndexMeta) bool {
		writeFingerprintIndex(buffer, index)
		return true
	})
	if spec.PrimaryKey != nil {
		buffer.writeString("PK:")
		buffer.writeString(columnsKey(spec.PrimaryKey.Columns))
		buffer.writeString("|")
	}
	spec.ForeignKeys.Range(func(_ int, foreignKey ForeignKeyMeta) bool {
		buffer.writeString("FK:")
		buffer.writeString(foreignKeyKey(foreignKey))
		buffer.writeString("|")
		return true
	})
	spec.Checks.Range(func(_ int, check CheckMeta) bool {
		buffer.writeString("CK:")
		buffer.writeString(check.Name)
		buffer.writeString(":")
		buffer.writeString(checkKey(check.Expression))
		buffer.writeString("|")
		return true
	})
}

func writeFingerprintColumn(buffer *renderBuffer, column ColumnMeta) {
	buffer.writeString("C:")
	buffer.writeString(column.Name)
	buffer.writeString(":")
	buffer.writeString(fingerprintColumnType(column))
	buffer.writeString(":")
	buffer.writeString(strconv.FormatBool(column.Nullable))
	buffer.writeString(":")
	buffer.writeString(column.DefaultValue)
	buffer.writeString(":")
	buffer.writeString(strconv.FormatBool(column.PrimaryKey))
	buffer.writeString(":")
	buffer.writeString(strconv.FormatBool(column.AutoIncrement))
	if column.References != nil {
		buffer.writeString(":ref:")
		buffer.writeString(column.References.TargetTable)
		buffer.writeString(".")
		buffer.writeString(column.References.TargetColumn)
	}
	buffer.writeString("|")
}

func fingerprintColumnType(column ColumnMeta) string {
	if column.SQLType != "" {
		return column.SQLType
	}
	return inferTypeName(column)
}

func writeFingerprintIndex(buffer *renderBuffer, index IndexMeta) {
	buffer.writeString("I:")
	buffer.writeString(index.Name)
	buffer.writeString(":")
	buffer.writeString(columnsKey(index.Columns))
	buffer.writeString(":")
	buffer.writeString(strconv.FormatBool(index.Unique))
	buffer.writeString("|")
}

func fingerprintString(value string) string {
	h := fnv.New64a()
	if _, err := h.Write([]byte(value)); err != nil {
		return ""
	}
	return strconv.FormatUint(h.Sum64(), 16)
}

func planSchemaChangesWithAtlas(ctx context.Context, session Session, schemas ...SchemaResource) (MigrationPlan, bool, error) {
	if len(schemas) == 0 {
		return MigrationPlan{}, true, nil
	}
	if err := validateAtlasPlanningSession(session); err != nil {
		return MigrationPlan{}, true, err
	}

	driver, ok, err := atlasDriverForSession(session)
	if err != nil || !ok {
		return MigrationPlan{}, ok, err
	}

	current, schemaName, err := atlasCurrentSchema(ctx, driver, session, schemas)
	if err != nil {
		return MigrationPlan{}, true, err
	}
	compiled, err := atlasCompiledSchemaForSession(session, driver, schemaName, schemas)
	if err != nil {
		return MigrationPlan{}, true, err
	}
	if current == nil {
		current = atlasschema.New(schemaName)
	}
	return atlasSchemaDiffPlan(ctx, driver, compiled, current)
}

func validateAtlasPlanningSession(session Session) error {
	if session == nil {
		return ErrNilDB
	}
	if session.Dialect() == nil {
		return ErrNilDialect
	}
	return nil
}

func atlasCurrentSchema(ctx context.Context, driver atlasmigrate.Driver, session Session, schemas []SchemaResource) (*atlasschema.Schema, string, error) {
	tableNames := make([]string, 0, len(schemas))
	for _, schema := range schemas {
		tableNames = append(tableNames, schema.tableRef().TableName())
	}
	current, err := atlasInspectCurrentSchema(ctx, driver, tableNames)
	if err != nil {
		return nil, "", err
	}
	schemaName := atlasDefaultSchemaName(session.Dialect().Name())
	if current != nil && strings.TrimSpace(current.Name) != "" {
		schemaName = current.Name
	}
	return current, schemaName, nil
}

func atlasCompiledSchemaForSession(session Session, driver atlasmigrate.Driver, schemaName string, schemas []SchemaResource) (*atlasCompiledSchema, error) {
	dialectName := session.Dialect().Name()
	cacheKey := dialectName + ":" + schemaName + ":" + schemaFingerprint(schemas)
	if compiled, ok, err := compiledSchemaCache.Get(cacheKey); err != nil {
		return nil, wrapDBError("read compiled schema cache", err)
	} else if ok {
		return compiled, nil
	}

	compiled := compileAtlasSchema(dialectName, driver, schemaName, schemas)
	compiledSchemaCache.Set(cacheKey, compiled)
	return compiled, nil
}

func atlasSchemaDiffPlan(ctx context.Context, driver atlasmigrate.Driver, compiled *atlasCompiledSchema, current *atlasschema.Schema) (MigrationPlan, bool, error) {
	changes, err := driver.SchemaDiff(current, compiled.schema)
	if err != nil {
		return MigrationPlan{}, true, wrapDBError("diff atlas schema", err)
	}
	report := atlasReportFromChanges(changes, compiled, current)
	if len(changes) == 0 {
		return MigrationPlan{Actions: collectionx.NewList[MigrationAction](), Report: report}, true, nil
	}
	safeChanges, manualActions := atlasSplitChanges(changes)
	actions, err := atlasPlanActions(ctx, driver, safeChanges)
	if err != nil {
		return MigrationPlan{}, true, err
	}
	return MigrationPlan{
		Actions: collectionx.NewListWithCapacity(len(actions)+len(manualActions), append(actions, manualActions...)...),
		Report:  report,
	}, true, nil
}

func atlasDriverForSession(session Session) (atlasmigrate.Driver, bool, error) {
	switch strings.ToLower(strings.TrimSpace(session.Dialect().Name())) {
	case "sqlite":
		driver, err := atlassqlite.Open(session)
		return driver, true, wrapDBError("open atlas sqlite driver", err)
	case "mysql":
		driver, err := atlasmysql.Open(session)
		return driver, true, wrapDBError("open atlas mysql driver", err)
	case "postgres":
		driver, err := atlaspostgres.Open(session)
		return driver, true, wrapDBError("open atlas postgres driver", err)
	default:
		return nil, false, nil
	}
}

func atlasInspectCurrentSchema(ctx context.Context, driver atlasmigrate.Driver, tables []string) (*atlasschema.Schema, error) {
	current, err := driver.InspectSchema(ctx, "", &atlasschema.InspectOptions{Mode: atlasschema.InspectTables, Tables: tables})
	if err != nil {
		if atlasschema.IsNotExistError(err) {
			var empty *atlasschema.Schema
			return empty, nil
		}
		return nil, wrapDBError("inspect current atlas schema", err)
	}
	return current, nil
}

func atlasDefaultSchemaName(dialectName string) string {
	switch strings.ToLower(strings.TrimSpace(dialectName)) {
	case "sqlite":
		return "main"
	case "postgres":
		return "public"
	default:
		return ""
	}
}
