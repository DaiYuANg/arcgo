package dbx

import (
	"slices"

	atlasschema "ariga.io/atlas/sql/schema"
	"github.com/DaiYuANg/arcgo/collectionx"
)

func atlasReportFromChanges(changes []atlasschema.Change, compiled *atlasCompiledSchema, current *atlasschema.Schema) ValidationReport {
	diffs := atlasReportDiffMap(compiled.order)
	currentTables := atlasCurrentTablesByName(current)
	for _, change := range changes {
		atlasApplyChangeToReport(diffs, compiled, currentTables, change)
	}
	return atlasValidationReport(diffs)
}

func atlasReportDiffMap(order []string) collectionx.OrderedMap[string, *TableDiff] {
	diffs := collectionx.NewOrderedMapWithCapacity[string, *TableDiff](len(order))
	for _, name := range order {
		diff := newTableDiff(name)
		diffs.Set(name, &diff)
	}
	return diffs
}

func atlasCurrentTablesByName(current *atlasschema.Schema) collectionx.Map[string, *atlasschema.Table] {
	if current == nil {
		return collectionx.NewMap[string, *atlasschema.Table]()
	}
	currentTables := collectionx.NewMapWithCapacity[string, *atlasschema.Table](len(current.Tables))
	for _, table := range current.Tables {
		currentTables.Set(table.Name, table)
	}
	return currentTables
}

func atlasApplyChangeToReport(diffs collectionx.OrderedMap[string, *TableDiff], compiled *atlasCompiledSchema, currentTables collectionx.Map[string, *atlasschema.Table], change atlasschema.Change) {
	switch c := change.(type) {
	case *atlasschema.AddTable:
		atlasApplyAddTableChange(diffs, compiled, c)
	case *atlasschema.ModifyTable:
		atlasApplyModifyTableChange(diffs, compiled, currentTables, c)
	}
}

func atlasApplyAddTableChange(diffs collectionx.OrderedMap[string, *TableDiff], compiled *atlasCompiledSchema, change *atlasschema.AddTable) {
	compiledTable, ok := compiled.tables.Get(change.T.Name)
	if !ok {
		return
	}
	diff, _ := diffs.Get(change.T.Name)
	diff.MissingTable = true
	diff.MissingColumns = collectionx.NewListWithCapacity(len(compiledTable.spec.Columns), slices.Clone(compiledTable.spec.Columns)...)
	diff.MissingIndexes = collectionx.NewListWithCapacity(len(compiledTable.spec.Indexes), slices.Clone(compiledTable.spec.Indexes)...)
	diff.MissingForeignKeys = collectionx.NewListWithCapacity(len(compiledTable.spec.ForeignKeys), slices.Clone(compiledTable.spec.ForeignKeys)...)
	diff.MissingChecks = collectionx.NewListWithCapacity(len(compiledTable.spec.Checks), slices.Clone(compiledTable.spec.Checks)...)
	if compiledTable.spec.PrimaryKey != nil {
		diff.PrimaryKeyDiff = &PrimaryKeyDiff{
			Expected: new(clonePrimaryKeyMeta(*compiledTable.spec.PrimaryKey)),
			Issues:   collectionx.NewList("table does not exist"),
		}
	}
}

func atlasApplyModifyTableChange(diffs collectionx.OrderedMap[string, *TableDiff], compiled *atlasCompiledSchema, currentTables collectionx.Map[string, *atlasschema.Table], change *atlasschema.ModifyTable) {
	compiledTable, ok := compiled.tables.Get(change.T.Name)
	if !ok {
		return
	}
	diff, _ := diffs.Get(change.T.Name)
	currentTable, _ := currentTables.Get(change.T.Name)
	for _, tableChange := range change.Changes {
		atlasApplyTableChangeToDiff(diff, compiledTable, currentTable, tableChange)
	}
}

func atlasValidationReport(diffs collectionx.OrderedMap[string, *TableDiff]) ValidationReport {
	items := collectionx.NewListWithCapacity[TableDiff](diffs.Len())
	diffs.Range(func(_ string, diff *TableDiff) bool {
		items.Add(*diff)
		return true
	})
	return ValidationReport{
		Tables:   items,
		Backend:  ValidationBackendAtlas,
		Complete: true,
	}
}
