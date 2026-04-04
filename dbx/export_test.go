package dbx

import (
	"database/sql"

	atlasschema "ariga.io/atlas/sql/schema"
	"github.com/DaiYuANg/arcgo/collectionx"
)

type AtlasCompiledSchemaTestView struct {
	Schema *atlasschema.Schema
	tables map[string]*atlasschema.Table
}

func CompileAtlasSchemaForTest(dialectName string, schemas ...SchemaResource) *AtlasCompiledSchemaTestView {
	compiled := compileAtlasSchema(dialectName, nil, "main", schemas)
	if compiled == nil {
		return nil
	}
	view := &AtlasCompiledSchemaTestView{
		Schema: compiled.schema,
		tables: make(map[string]*atlasschema.Table, compiled.tables.Len()),
	}
	compiled.tables.Range(func(name string, table *atlasCompiledTable) bool {
		view.tables[name] = table.table
		return true
	})
	return view
}

func (v *AtlasCompiledSchemaTestView) Table(name string) (*atlasschema.Table, bool) {
	if v == nil {
		return nil, false
	}
	table, ok := v.tables[name]
	return table, ok
}

func AtlasSplitChangesForTest(changes []atlasschema.Change) ([]atlasschema.Change, []MigrationAction) {
	return atlasSplitChanges(changes)
}

func TableSpecForTest(schema SchemaResource) TableSpec {
	return buildTableSpec(schema.schemaRef())
}

func IndexesForTest(schema SchemaResource) collectionx.List[IndexMeta] {
	indexes := deriveIndexes(schema.schemaRef())
	return collectionx.NewListWithCapacity(len(indexes), indexes...)
}

func InferTypeNameForTest(column ColumnMeta) string {
	return inferTypeName(column)
}

func ErrorRowForTest(err error) *Row {
	return errorRow(err)
}

func CloseRowsForTest(rows *sql.Rows) error {
	return closeRows(rows)
}

func RowsIterErrorForTest(rows *sql.Rows) error {
	return rowsIterError(rows)
}

func StructMapperScanPlanForTest[E any](mapper StructMapper[E], columns []string) error {
	_, err := mapper.scanPlan(columns)
	return err
}

func ClonePrimaryKeyMetaForTest(meta PrimaryKeyMeta) PrimaryKeyMeta {
	return clonePrimaryKeyMeta(meta)
}

func ClonePrimaryKeyStateForTest(state PrimaryKeyState) PrimaryKeyState {
	return clonePrimaryKeyState(state)
}

func NewSQLExecutorForTest(session Session) *SQLExecutor {
	return &SQLExecutor{session: session}
}
