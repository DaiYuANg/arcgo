package dbx

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"reflect"
	"slices"
	"strconv"
	"strings"

	atlasmigrate "ariga.io/atlas/sql/migrate"
	atlasmysql "ariga.io/atlas/sql/mysql"
	atlaspostgres "ariga.io/atlas/sql/postgres"
	atlasschema "ariga.io/atlas/sql/schema"
	atlassqlite "ariga.io/atlas/sql/sqlite"
	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/hot"
	"github.com/samber/lo"
)

var compiledSchemaCache = hot.NewHotCache[string, *atlasCompiledSchema](hot.LRU, 128).Build()

func schemaFingerprint(schemas []SchemaResource) string {
	if len(schemas) == 0 {
		return ""
	}
	var buffer renderBuffer
	for _, s := range schemas {
		spec := buildTableSpec(s.schemaRef())
		buffer.writeString("T:")
		buffer.writeString(spec.Name)
		buffer.writeString("|")
		for _, c := range spec.Columns {
			buffer.writeString("C:")
			buffer.writeString(c.Name)
			buffer.writeString(":")
			if c.SQLType != "" {
				buffer.writeString(c.SQLType)
			} else {
				buffer.writeString(inferTypeName(c))
			}
			buffer.writeString(":")
			buffer.writeString(strconv.FormatBool(c.Nullable))
			buffer.writeString(":")
			buffer.writeString(c.DefaultValue)
			buffer.writeString(":")
			buffer.writeString(strconv.FormatBool(c.PrimaryKey))
			buffer.writeString(":")
			buffer.writeString(strconv.FormatBool(c.AutoIncrement))
			if c.References != nil {
				buffer.writeString(":ref:")
				buffer.writeString(c.References.TargetTable)
				buffer.writeString(".")
				buffer.writeString(c.References.TargetColumn)
			}
			buffer.writeString("|")
		}
		for _, idx := range spec.Indexes {
			buffer.writeString("I:")
			buffer.writeString(idx.Name)
			buffer.writeString(":")
			buffer.writeString(strings.Join(idx.Columns, ","))
			buffer.writeString(":")
			buffer.writeString(strconv.FormatBool(idx.Unique))
			buffer.writeString("|")
		}
		if spec.PrimaryKey != nil {
			buffer.writeString("PK:")
			buffer.writeString(strings.Join(spec.PrimaryKey.Columns, ","))
			buffer.writeString("|")
		}
		for _, fk := range spec.ForeignKeys {
			buffer.writeString("FK:")
			buffer.writeString(foreignKeyKey(fk))
			buffer.writeString("|")
		}
		for _, ck := range spec.Checks {
			buffer.writeString("CK:")
			buffer.writeString(ck.Name)
			buffer.writeString(":")
			buffer.writeString(checkKey(ck.Expression))
			buffer.writeString("|")
		}
	}
	if err := buffer.Err("build schema fingerprint"); err != nil {
		return ""
	}
	h := fnv.New64a()
	if _, err := h.Write([]byte(buffer.String())); err != nil {
		return ""
	}
	return strconv.FormatUint(h.Sum64(), 16)
}

type atlasCompiledSchema struct {
	schema    *atlasschema.Schema
	tables    collectionx.Map[string, *atlasCompiledTable]
	externals collectionx.Map[string, *atlasschema.Table]
	order     []string
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

func planSchemaChangesWithAtlas(ctx context.Context, session Session, schemas ...SchemaResource) (MigrationPlan, bool, error) {
	if len(schemas) == 0 {
		return MigrationPlan{}, true, nil
	}
	if session == nil {
		return MigrationPlan{}, true, ErrNilDB
	}
	if session.Dialect() == nil {
		return MigrationPlan{}, true, ErrNilDialect
	}

	driver, ok, err := atlasDriverForSession(session)
	if err != nil || !ok {
		return MigrationPlan{}, ok, err
	}

	tableNames := lo.Map(schemas, func(schema SchemaResource, _ int) string {
		return schema.tableRef().TableName()
	})
	current, err := atlasInspectCurrentSchema(ctx, driver, tableNames)
	if err != nil {
		return MigrationPlan{}, true, err
	}
	schemaName := atlasDefaultSchemaName(session.Dialect().Name())
	if current != nil && strings.TrimSpace(current.Name) != "" {
		schemaName = current.Name
	}
	dialectName := session.Dialect().Name()
	cacheKey := dialectName + ":" + schemaName + ":" + schemaFingerprint(schemas)
	var compiled *atlasCompiledSchema
	if v, ok, cacheErr := compiledSchemaCache.Get(cacheKey); cacheErr != nil {
		return MigrationPlan{}, true, wrapDBError("read compiled schema cache", cacheErr)
	} else if ok {
		compiled = v
	} else {
		compiled = compileAtlasSchema(dialectName, driver, schemaName, schemas)
		compiledSchemaCache.Set(cacheKey, compiled)
	}
	if current == nil {
		current = atlasschema.New(schemaName)
	}

	changes, err := driver.SchemaDiff(current, compiled.schema)
	if err != nil {
		return MigrationPlan{}, true, wrapDBError("diff atlas schema", err)
	}
	if len(changes) == 0 {
		report := atlasReportFromChanges(nil, compiled, current)
		return MigrationPlan{Actions: nil, Report: report}, true, nil
	}
	report := atlasReportFromChanges(changes, compiled, current)
	safeChanges, manualActions := atlasSplitChanges(changes)
	actions, err := atlasPlanActions(ctx, driver, safeChanges)
	if err != nil {
		return MigrationPlan{}, true, err
	}
	return MigrationPlan{
		Actions: append(actions, manualActions...),
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

func compileAtlasSchema(dialectName string, driver atlasmigrate.Driver, schemaName string, schemas []SchemaResource) *atlasCompiledSchema {
	atlasSchema := atlasschema.New(schemaName)
	compiled := &atlasCompiledSchema{
		schema:    atlasSchema,
		tables:    collectionx.NewMapWithCapacity[string, *atlasCompiledTable](len(schemas)),
		externals: collectionx.NewMap[string, *atlasschema.Table](),
		order: lo.Map(schemas, func(schema SchemaResource, _ int) string {
			return schema.tableRef().TableName()
		}),
	}

	for _, resource := range schemas {
		spec := buildTableSpec(resource.schemaRef())
		table := atlasschema.NewTable(spec.Name).SetSchema(atlasSchema)
		compiledTable := &atlasCompiledTable{
			spec:              spec,
			table:             table,
			columnsByName:     collectionx.NewMapWithCapacity[string, ColumnMeta](len(spec.Columns)),
			indexesByName:     collectionx.NewMapWithCapacity[string, IndexMeta](len(spec.Indexes)),
			indexesByKey:      collectionx.NewMapWithCapacity[string, IndexMeta](len(spec.Indexes)),
			foreignKeysByName: collectionx.NewMapWithCapacity[string, ForeignKeyMeta](len(spec.ForeignKeys)),
			foreignKeysByKey:  collectionx.NewMapWithCapacity[string, ForeignKeyMeta](len(spec.ForeignKeys)),
			checksByName:      collectionx.NewMapWithCapacity[string, CheckMeta](len(spec.Checks)),
			checksByExpr:      collectionx.NewMapWithCapacity[string, CheckMeta](len(spec.Checks)),
		}
		cols := spec.Columns
		for i := range cols {
			column := &cols[i]
			atlasColumn := compileAtlasColumn(dialectName, driver, *column)
			table.AddColumns(atlasColumn)
			compiledTable.columnsByName.Set(column.Name, *column)
		}
		for _, index := range spec.Indexes {
			compiledTable.indexesByName.Set(index.Name, index)
			compiledTable.indexesByKey.Set(indexKey(index.Unique, index.Columns), index)
		}
		for _, foreignKey := range spec.ForeignKeys {
			compiledTable.foreignKeysByName.Set(foreignKey.Name, foreignKey)
			compiledTable.foreignKeysByKey.Set(foreignKeyKey(foreignKey), foreignKey)
		}
		for _, check := range spec.Checks {
			compiledTable.checksByName.Set(check.Name, check)
			compiledTable.checksByExpr.Set(checkKey(check.Expression), check)
		}
		atlasSchema.AddTables(table)
		compiled.tables.Set(spec.Name, compiledTable)
	}

	compiled.tables.Range(func(_ string, table *atlasCompiledTable) bool {
		if table.spec.PrimaryKey != nil {
			if pk := atlasPrimaryKeyForSpec(table.table, *table.spec.PrimaryKey); pk != nil {
				table.table.SetPrimaryKey(pk)
			}
		}
		for _, index := range table.spec.Indexes {
			if atlasIndex := atlasIndexForSpec(table.table, index); atlasIndex != nil {
				table.table.AddIndexes(atlasIndex)
			}
		}
		for _, foreignKey := range table.spec.ForeignKeys {
			if atlasForeignKey := compiled.atlasForeignKeyForSpec(table.table, foreignKey); atlasForeignKey != nil {
				table.table.AddForeignKeys(atlasForeignKey)
			}
		}
		for _, check := range table.spec.Checks {
			table.table.AddChecks(atlasschema.NewCheck().SetName(check.Name).SetExpr(check.Expression))
		}
		return true
	})

	return compiled
}

func compileAtlasColumn(dialectName string, driver atlasmigrate.Driver, column ColumnMeta) *atlasschema.Column {
	rawType := atlasColumnRawType(dialectName, column)
	atlasColumn := atlasschema.NewColumn(column.Name)
	atlasColumn.Type = &atlasschema.ColumnType{
		Type: atlasColumnType(driver, rawType, column),
		Raw:  rawType,
		Null: column.Nullable,
	}
	atlasColumn.SetNull(column.Nullable)
	if column.DefaultValue != "" {
		atlasColumn.SetDefault(&atlasschema.RawExpr{X: column.DefaultValue})
	}
	if column.AutoIncrement {
		atlasAddAutoIncrementAttr(dialectName, atlasColumn)
	}
	return atlasColumn
}

func atlasColumnRawType(dialectName string, column ColumnMeta) string {
	rawType := strings.TrimSpace(column.SQLType)
	if rawType == "" {
		rawType = inferTypeName(column)
	}
	if strings.EqualFold(strings.TrimSpace(dialectName), "sqlite") && column.AutoIncrement {
		return "integer"
	}
	return rawType
}

func atlasColumnType(driver atlasmigrate.Driver, rawType string, column ColumnMeta) atlasschema.Type {
	if parser, ok := driver.(atlasschema.TypeParser); ok && rawType != "" {
		if parsed, err := parser.ParseType(rawType); err == nil && parsed != nil {
			return parsed
		}
	}
	return atlasFallbackType(rawType, column)
}

func atlasFallbackType(rawType string, column ColumnMeta) atlasschema.Type {
	if rawType == "" {
		rawType = inferTypeName(column)
	}
	typeName := strings.ToLower(strings.TrimSpace(rawType))
	if column.GoType != nil {
		typ := column.GoType
		for typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
		if typ.PkgPath() == "time" && typ.Name() == "Time" {
			return &atlasschema.TimeType{T: rawType}
		}
		switch kind := typ.Kind(); {
		case kind == reflect.Bool:
			return &atlasschema.BoolType{T: rawType}
		case isSignedIntKind(kind) || kind == reflect.Int64:
			return &atlasschema.IntegerType{T: rawType}
		case isUnsignedIntKind(kind) || kind == reflect.Uint64:
			return &atlasschema.IntegerType{T: rawType, Unsigned: true}
		case kind == reflect.Float32 || kind == reflect.Float64:
			return &atlasschema.FloatType{T: rawType}
		case kind == reflect.String:
			return &atlasschema.StringType{T: rawType}
		case isByteSliceType(typ):
			return &atlasschema.BinaryType{T: rawType}
		case kind == reflect.Slice && strings.Contains(typeName, "json"):
			return &atlasschema.JSONType{T: rawType}
		case kind == reflect.Map || kind == reflect.Struct:
			if strings.Contains(typeName, "json") {
				return &atlasschema.JSONType{T: rawType}
			}
		}
	}
	switch {
	case strings.Contains(typeName, "bool"):
		return &atlasschema.BoolType{T: rawType}
	case strings.Contains(typeName, "json"):
		return &atlasschema.JSONType{T: rawType}
	case strings.Contains(typeName, "time") || strings.Contains(typeName, "date"):
		return &atlasschema.TimeType{T: rawType}
	case strings.Contains(typeName, "char") || strings.Contains(typeName, "text") || strings.Contains(typeName, "string"):
		return &atlasschema.StringType{T: rawType}
	case strings.Contains(typeName, "blob") || strings.Contains(typeName, "binary") || strings.Contains(typeName, "bytea"):
		return &atlasschema.BinaryType{T: rawType}
	case strings.Contains(typeName, "real") || strings.Contains(typeName, "double") || strings.Contains(typeName, "float"):
		return &atlasschema.FloatType{T: rawType}
	case strings.Contains(typeName, "numeric") || strings.Contains(typeName, "decimal"):
		return &atlasschema.DecimalType{T: rawType}
	case strings.Contains(typeName, "int"):
		return &atlasschema.IntegerType{T: rawType}
	default:
		return &atlasschema.UnsupportedType{T: rawType}
	}
}

func atlasAddAutoIncrementAttr(dialectName string, column *atlasschema.Column) {
	switch strings.ToLower(strings.TrimSpace(dialectName)) {
	case "mysql":
		column.AddAttrs(&atlasmysql.AutoIncrement{})
	case "sqlite":
		column.AddAttrs(&atlassqlite.AutoIncrement{})
	case "postgres":
		column.AddAttrs(&atlaspostgres.Identity{Generation: "BY DEFAULT"})
	}
}

func atlasPrimaryKeyForSpec(table *atlasschema.Table, primaryKey PrimaryKeyMeta) *atlasschema.Index {
	columns := lo.FilterMap(primaryKey.Columns, func(name string, _ int) (*atlasschema.Column, bool) {
		column, ok := table.Column(name)
		return column, ok
	})
	if len(columns) == 0 {
		return nil
	}
	return atlasschema.NewPrimaryKey(columns...).SetName(primaryKey.Name)
}

func atlasIndexForSpec(table *atlasschema.Table, index IndexMeta) *atlasschema.Index {
	columns := lo.FilterMap(index.Columns, func(name string, _ int) (*atlasschema.Column, bool) {
		column, ok := table.Column(name)
		return column, ok
	})
	if len(columns) == 0 {
		return nil
	}
	return atlasschema.NewIndex(index.Name).SetUnique(index.Unique).AddColumns(columns...)
}

func (c *atlasCompiledSchema) atlasForeignKeyForSpec(table *atlasschema.Table, foreignKey ForeignKeyMeta) *atlasschema.ForeignKey {
	columns := lo.FilterMap(foreignKey.Columns, func(name string, _ int) (*atlasschema.Column, bool) {
		column, ok := table.Column(name)
		return column, ok
	})
	if len(columns) == 0 {
		return nil
	}
	refTable := c.referenceTable(table.Schema, foreignKey.TargetTable, foreignKey.TargetColumns)
	refColumns := lo.FilterMap(foreignKey.TargetColumns, func(name string, _ int) (*atlasschema.Column, bool) {
		column, ok := refTable.Column(name)
		return column, ok
	})
	if len(refColumns) == 0 {
		return nil
	}
	return atlasschema.NewForeignKey(foreignKey.Name).
		SetTable(table).
		AddColumns(columns...).
		SetRefTable(refTable).
		AddRefColumns(refColumns...).
		SetOnDelete(atlasReferenceAction(foreignKey.OnDelete)).
		SetOnUpdate(atlasReferenceAction(foreignKey.OnUpdate))
}

func (c *atlasCompiledSchema) referenceTable(schema *atlasschema.Schema, tableName string, targetColumns []string) *atlasschema.Table {
	if compiled, ok := c.tables.Get(tableName); ok {
		return compiled.table
	}
	if external, ok := c.externals.Get(tableName); ok {
		return external
	}
	external := atlasschema.NewTable(tableName).SetSchema(schema)
	for i := range targetColumns {
		column := targetColumns[i]
		external.AddColumns(atlasschema.NewColumn(column))
	}
	c.externals.Set(tableName, external)
	return external
}

func atlasReferenceAction(action ReferentialAction) atlasschema.ReferenceOption {
	switch normalized := normalizeReferentialAction(action); normalized {
	case ReferentialCascade:
		return atlasschema.Cascade
	case ReferentialRestrict:
		return atlasschema.Restrict
	case ReferentialSetNull:
		return atlasschema.SetNull
	case ReferentialSetDefault:
		return atlasschema.SetDefault
	case ReferentialNoAction:
		return atlasschema.NoAction
	default:
		return atlasschema.NoAction
	}
}

func atlasReportFromChanges(changes []atlasschema.Change, compiled *atlasCompiledSchema, current *atlasschema.Schema) ValidationReport {
	diffs := collectionx.NewOrderedMap[string, *TableDiff]()
	for _, name := range compiled.order {
		diffs.Set(name, &TableDiff{Table: name})
	}

	currentTables := collectionx.NewMap[string, *atlasschema.Table]()
	if current != nil {
		for _, table := range current.Tables {
			currentTables.Set(table.Name, table)
		}
	}

	for _, change := range changes {
		switch c := change.(type) {
		case *atlasschema.AddTable:
			compiledTable, ok := compiled.tables.Get(c.T.Name)
			if !ok {
				continue
			}
			diff, _ := diffs.Get(c.T.Name)
			diff.MissingTable = true
			diff.MissingColumns = slices.Clone(compiledTable.spec.Columns)
			diff.MissingIndexes = slices.Clone(compiledTable.spec.Indexes)
			diff.MissingForeignKeys = slices.Clone(compiledTable.spec.ForeignKeys)
			diff.MissingChecks = slices.Clone(compiledTable.spec.Checks)
			if compiledTable.spec.PrimaryKey != nil {
				diff.PrimaryKeyDiff = &PrimaryKeyDiff{Expected: new(clonePrimaryKeyMeta(*compiledTable.spec.PrimaryKey)), Issues: []string{"table does not exist"}}
			}
		case *atlasschema.ModifyTable:
			compiledTable, ok := compiled.tables.Get(c.T.Name)
			if !ok {
				continue
			}
			diff, _ := diffs.Get(c.T.Name)
			currentTable, _ := currentTables.Get(c.T.Name)
			for _, tableChange := range c.Changes {
				atlasApplyTableChangeToDiff(diff, compiledTable, currentTable, tableChange)
			}
		}
	}

	reportTables := collectionx.NewListWithCapacity[TableDiff](diffs.Len())
	diffs.Range(func(_ string, diff *TableDiff) bool {
		reportTables.Add(*diff)
		return true
	})
	return ValidationReport{
		Tables:   reportTables.Values(),
		Backend:  ValidationBackendAtlas,
		Complete: true,
	}
}

func atlasApplyTableChangeToDiff(diff *TableDiff, compiled *atlasCompiledTable, current *atlasschema.Table, change atlasschema.Change) {
	switch c := change.(type) {
	case *atlasschema.AddColumn:
		if column, ok := compiled.columnsByName.Get(c.C.Name); ok {
			diff.MissingColumns = append(diff.MissingColumns, column)
		}
	case *atlasschema.ModifyColumn:
		name := c.To.Name
		if name == "" {
			name = c.From.Name
		}
		column, ok := compiled.columnsByName.Get(name)
		if !ok {
			column = ColumnMeta{Name: name, Table: diff.Table}
		}
		diff.ColumnDiffs = append(diff.ColumnDiffs, ColumnDiff{Column: column, Issues: []string{atlasColumnChangeIssue(c.Change)}})
	case *atlasschema.RenameColumn:
		diff.ColumnDiffs = append(diff.ColumnDiffs, ColumnDiff{Column: ColumnMeta{Name: c.To.Name, Table: diff.Table}, Issues: []string{"manual column rename migration required"}})
	case *atlasschema.DropColumn:
		diff.ColumnDiffs = append(diff.ColumnDiffs, ColumnDiff{Column: ColumnMeta{Name: c.C.Name, Table: diff.Table}, Issues: []string{"manual column removal migration required"}})
	case *atlasschema.AddIndex:
		if index, ok := atlasFindIndexMeta(compiled, c.I); ok {
			diff.MissingIndexes = append(diff.MissingIndexes, index)
		}
	case *atlasschema.ModifyIndex:
		if index, ok := atlasFindIndexMeta(compiled, c.To); ok {
			diff.MissingIndexes = append(diff.MissingIndexes, index)
		} else {
			diff.ColumnDiffs = append(diff.ColumnDiffs, ColumnDiff{Column: ColumnMeta{Name: c.To.Name, Table: diff.Table}, Issues: []string{"manual index modification required"}})
		}
	case *atlasschema.RenameIndex:
		diff.ColumnDiffs = append(diff.ColumnDiffs, ColumnDiff{Column: ColumnMeta{Name: c.To.Name, Table: diff.Table}, Issues: []string{"manual index rename migration required"}})
	case *atlasschema.DropIndex:
		diff.ColumnDiffs = append(diff.ColumnDiffs, ColumnDiff{Column: ColumnMeta{Name: c.I.Name, Table: diff.Table}, Issues: []string{"manual index removal migration required"}})
	case *atlasschema.AddForeignKey:
		if foreignKey, ok := atlasFindForeignKeyMeta(compiled, c.F); ok {
			diff.MissingForeignKeys = append(diff.MissingForeignKeys, foreignKey)
		}
	case *atlasschema.ModifyForeignKey:
		if foreignKey, ok := atlasFindForeignKeyMeta(compiled, c.To); ok {
			diff.MissingForeignKeys = append(diff.MissingForeignKeys, foreignKey)
		}
	case *atlasschema.DropForeignKey:
		diff.ColumnDiffs = append(diff.ColumnDiffs, ColumnDiff{Column: ColumnMeta{Name: c.F.Symbol, Table: diff.Table}, Issues: []string{"manual foreign key removal migration required"}})
	case *atlasschema.AddCheck:
		if check, ok := atlasFindCheckMeta(compiled, c.C); ok {
			diff.MissingChecks = append(diff.MissingChecks, check)
		}
	case *atlasschema.ModifyCheck:
		if check, ok := atlasFindCheckMeta(compiled, c.To); ok {
			diff.MissingChecks = append(diff.MissingChecks, check)
		}
	case *atlasschema.DropCheck:
		diff.ColumnDiffs = append(diff.ColumnDiffs, ColumnDiff{Column: ColumnMeta{Name: c.C.Name, Table: diff.Table}, Issues: []string{"manual check removal migration required"}})
	case *atlasschema.AddPrimaryKey:
		diff.PrimaryKeyDiff = &PrimaryKeyDiff{
			Expected: compiled.spec.PrimaryKey,
			Actual:   atlasPrimaryKeyState(current),
			Issues:   []string{"missing primary key"},
		}
	case *atlasschema.ModifyPrimaryKey, *atlasschema.DropPrimaryKey:
		var actual *PrimaryKeyState
		if current != nil {
			actual = atlasPrimaryKeyState(current)
		}
		var expected *PrimaryKeyMeta
		if compiled.spec.PrimaryKey != nil {
			expected = new(clonePrimaryKeyMeta(*compiled.spec.PrimaryKey))
		}
		diff.PrimaryKeyDiff = &PrimaryKeyDiff{Expected: expected, Actual: actual, Issues: []string{"primary key migration required"}}
	}
}

func atlasSplitChanges(changes []atlasschema.Change) ([]atlasschema.Change, []MigrationAction) {
	safeChanges := collectionx.NewList[atlasschema.Change]()
	manualActions := collectionx.NewList[MigrationAction]()

	for _, change := range changes {
		switch c := change.(type) {
		case *atlasschema.AddTable:
			safeChanges.Add(c)
		case *atlasschema.ModifyTable:
			for _, tableChange := range c.Changes {
				if atlasIsExecutableTableChange(tableChange) {
					safeChanges.Add(&atlasschema.ModifyTable{T: c.T, Changes: []atlasschema.Change{tableChange}})
					continue
				}
				manualActions.Add(atlasManualAction(c.T.Name, tableChange))
			}
		default:
			manualActions.Add(MigrationAction{Kind: MigrationActionManual, Table: atlasChangeTableName(change), Summary: atlasManualSummary(change)})
		}
	}
	return safeChanges.Values(), manualActions.Values()
}

func atlasIsExecutableTableChange(change atlasschema.Change) bool {
	switch change.(type) {
	case *atlasschema.AddColumn, *atlasschema.AddIndex, *atlasschema.AddForeignKey, *atlasschema.AddCheck:
		return true
	default:
		return false
	}
}

func atlasPlanActions(ctx context.Context, driver atlasmigrate.Driver, changes []atlasschema.Change) ([]MigrationAction, error) {
	actions := collectionx.NewListWithCapacity[MigrationAction](len(changes))
	for _, change := range changes {
		plan, err := driver.PlanChanges(ctx, "dbx_schema_plan", []atlasschema.Change{change})
		if err != nil {
			if errors.Is(err, atlasmigrate.ErrNoPlan) {
				continue
			}
			return nil, wrapDBError("plan atlas schema changes", err)
		}
		kind := atlasActionKind(change)
		table := atlasChangeTableName(change)
		fallbackSummary := atlasActionSummary(change)
		for _, planned := range plan.Changes {
			summary := strings.TrimSpace(planned.Comment)
			if summary == "" {
				summary = fallbackSummary
			}
			actions.Add(MigrationAction{
				Kind:       kind,
				Table:      table,
				Summary:    summary,
				Statement:  BoundQuery{SQL: planned.Cmd, Args: slices.Clone(planned.Args)},
				Executable: true,
			})
		}
	}
	return actions.Values(), nil
}

func atlasActionKind(change atlasschema.Change) MigrationActionKind {
	switch c := change.(type) {
	case *atlasschema.AddTable:
		return MigrationActionCreateTable
	case *atlasschema.ModifyTable:
		if len(c.Changes) == 1 {
			switch c.Changes[0].(type) {
			case *atlasschema.AddColumn:
				return MigrationActionAddColumn
			case *atlasschema.AddIndex:
				return MigrationActionCreateIndex
			case *atlasschema.AddForeignKey:
				return MigrationActionAddForeignKey
			case *atlasschema.AddCheck:
				return MigrationActionAddCheck
			}
		}
	}
	return MigrationActionManual
}

func atlasActionSummary(change atlasschema.Change) string {
	switch c := change.(type) {
	case *atlasschema.AddTable:
		return "create table " + c.T.Name
	case *atlasschema.ModifyTable:
		if len(c.Changes) != 1 {
			return "modify table " + c.T.Name
		}
		switch m := c.Changes[0].(type) {
		case *atlasschema.AddColumn:
			return "add column " + m.C.Name
		case *atlasschema.AddIndex:
			return "create index " + m.I.Name
		case *atlasschema.AddForeignKey:
			return "add foreign key " + m.F.Symbol
		case *atlasschema.AddCheck:
			return "add check " + m.C.Name
		default:
			return atlasManualSummary(m)
		}
	default:
		return atlasManualSummary(change)
	}
}

func atlasManualAction(table string, change atlasschema.Change) MigrationAction {
	return MigrationAction{Kind: MigrationActionManual, Table: table, Summary: atlasManualSummary(change)}
}

func atlasManualSummary(change atlasschema.Change) string {
	switch c := change.(type) {
	case *atlasschema.ModifyColumn:
		return "manual column migration required for " + c.To.Name
	case *atlasschema.RenameColumn:
		return "manual column rename migration required for " + c.From.Name + " -> " + c.To.Name
	case *atlasschema.DropColumn:
		return "manual column removal migration required for " + c.C.Name
	case *atlasschema.ModifyIndex:
		return "manual index migration required for " + c.To.Name
	case *atlasschema.RenameIndex:
		return "manual index rename migration required for " + c.From.Name + " -> " + c.To.Name
	case *atlasschema.DropIndex:
		return "manual index removal migration required for " + c.I.Name
	case *atlasschema.AddPrimaryKey:
		return "manual primary key migration required"
	case *atlasschema.ModifyPrimaryKey:
		return "manual primary key migration required"
	case *atlasschema.DropPrimaryKey:
		return "manual primary key migration required"
	case *atlasschema.ModifyForeignKey:
		return "manual foreign key migration required for " + c.To.Symbol
	case *atlasschema.DropForeignKey:
		return "manual foreign key removal migration required for " + c.F.Symbol
	case *atlasschema.ModifyCheck:
		return "manual check migration required for " + c.To.Name
	case *atlasschema.DropCheck:
		return "manual check removal migration required for " + c.C.Name
	default:
		return "manual schema migration required"
	}
}

func atlasChangeTableName(change atlasschema.Change) string {
	switch c := change.(type) {
	case *atlasschema.AddTable:
		return c.T.Name
	case *atlasschema.ModifyTable:
		return c.T.Name
	case *atlasschema.DropTable:
		return c.T.Name
	default:
		return ""
	}
}

func atlasFindIndexMeta(compiled *atlasCompiledTable, index *atlasschema.Index) (IndexMeta, bool) {
	if compiled == nil || index == nil {
		return IndexMeta{}, false
	}
	if index.Name != "" {
		if value, ok := compiled.indexesByName.Get(index.Name); ok {
			return value, true
		}
	}
	return compiled.indexesByKey.Get(indexKey(index.Unique, atlasIndexColumns(index)))
}

func atlasFindForeignKeyMeta(compiled *atlasCompiledTable, foreignKey *atlasschema.ForeignKey) (ForeignKeyMeta, bool) {
	if compiled == nil || foreignKey == nil {
		return ForeignKeyMeta{}, false
	}
	if foreignKey.Symbol != "" {
		if value, ok := compiled.foreignKeysByName.Get(foreignKey.Symbol); ok {
			return value, true
		}
	}
	return compiled.foreignKeysByKey.Get(atlasForeignKeyKey(foreignKey))
}

func atlasFindCheckMeta(compiled *atlasCompiledTable, check *atlasschema.Check) (CheckMeta, bool) {
	if compiled == nil || check == nil {
		return CheckMeta{}, false
	}
	if check.Name != "" {
		if value, ok := compiled.checksByName.Get(check.Name); ok {
			return value, true
		}
	}
	return compiled.checksByExpr.Get(checkKey(check.Expr))
}

func atlasForeignKeyKey(foreignKey *atlasschema.ForeignKey) string {
	columns := lo.FilterMap(foreignKey.Columns, func(column *atlasschema.Column, _ int) (string, bool) {
		return column.Name, column != nil
	})
	targetColumns := lo.FilterMap(foreignKey.RefColumns, func(column *atlasschema.Column, _ int) (string, bool) {
		return column.Name, column != nil
	})
	meta := ForeignKeyMeta{
		Columns:       columns,
		TargetTable:   lo.If(foreignKey.RefTable != nil, foreignKey.RefTable.Name).Else(""),
		TargetColumns: targetColumns,
		OnDelete:      ReferentialAction(foreignKey.OnDelete),
		OnUpdate:      ReferentialAction(foreignKey.OnUpdate),
	}
	return foreignKeyKey(meta)
}

func atlasIndexColumns(index *atlasschema.Index) []string {
	return lo.FilterMap(index.Parts, func(part *atlasschema.IndexPart, _ int) (string, bool) {
		if part == nil || part.C == nil {
			return "", false
		}
		return part.C.Name, true
	})
}

func atlasPrimaryKeyState(table *atlasschema.Table) *PrimaryKeyState {
	if table == nil || table.PrimaryKey == nil {
		return nil
	}
	return &PrimaryKeyState{Name: table.PrimaryKey.Name, Columns: atlasIndexColumns(table.PrimaryKey)}
}

func atlasColumnChangeIssue(change atlasschema.ChangeKind) string {
	if change == atlasschema.NoChange {
		return "column migration required"
	}
	return fmt.Sprintf("column migration required (%s)", change)
}
