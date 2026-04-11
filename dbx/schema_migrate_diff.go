package dbx

import (
	"context"

	"github.com/DaiYuANg/arcgo/collectionx"
)

func diffSchema(ctx context.Context, schemaDialect SchemaDialect, session Session, schema SchemaResource) (TableDiff, error) {
	spec := buildTableSpec(schema.schemaRef())
	actual, err := schemaDialect.InspectTable(ctx, session, spec.Name)
	if err != nil {
		return TableDiff{}, wrapDBError("inspect schema table", err)
	}
	if !actual.Exists {
		return missingTableDiff(spec), nil
	}
	return existingTableDiff(schemaDialect, spec, actual), nil
}

func missingTableDiff(spec TableSpec) TableDiff {
	diff := newTableDiff(spec.Name)
	diff.MissingTable = true
	diff.MissingColumns = spec.Columns.Clone()
	diff.MissingIndexes = spec.Indexes.Clone()
	diff.MissingForeignKeys = spec.ForeignKeys.Clone()
	diff.MissingChecks = spec.Checks.Clone()
	if spec.PrimaryKey != nil {
		diff.PrimaryKeyDiff = &PrimaryKeyDiff{
			Expected: new(clonePrimaryKeyMeta(*spec.PrimaryKey)),
			Issues:   collectionx.NewList("table does not exist"),
		}
	}
	return diff
}

func existingTableDiff(schemaDialect SchemaDialect, spec TableSpec, actual TableState) TableDiff {
	diff := newTableDiff(spec.Name)
	actualColumns := collectionx.AssociateList(actual.Columns, func(_ int, column ColumnState) (string, ColumnState) {
		return column.Name, column
	})
	diffColumns(schemaDialect, spec.Columns, actualColumns, &diff)
	diffPrimaryKey(spec.PrimaryKey, actual.PrimaryKey, &diff)
	diffIndexes(spec.Indexes, actual.Indexes, &diff)
	diffForeignKeys(spec.ForeignKeys, actual.ForeignKeys, &diff)
	diffChecks(spec.Checks, actual.Checks, &diff)
	return diff
}

func diffColumns(schemaDialect SchemaDialect, expectedColumns collectionx.List[ColumnMeta], actualColumns collectionx.Map[string, ColumnState], diff *TableDiff) {
	missingColumns := collectionx.NewListWithCapacity[ColumnMeta](expectedColumns.Len())
	columnDiffs := collectionx.NewListWithCapacity[ColumnDiff](expectedColumns.Len())
	expectedColumns.Range(func(_ int, expected ColumnMeta) bool {
		actualColumn, ok := actualColumns.Get(expected.Name)
		if !ok {
			missingColumns.Add(expected)
			return true
		}
		issues := columnDiffIssues(schemaDialect, expected, actualColumn)
		if len(issues) == 0 {
			return true
		}
		columnDiffs.Add(ColumnDiff{Column: expected, Issues: collectionx.NewListWithCapacity(len(issues), issues...)})
		return true
	})
	diff.MissingColumns = missingColumns
	diff.ColumnDiffs = columnDiffs
}

func diffPrimaryKey(expected *PrimaryKeyMeta, actual *PrimaryKeyState, diff *TableDiff) {
	issues := primaryKeyIssues(expected, actual)
	if len(issues) == 0 {
		return
	}
	diff.PrimaryKeyDiff = &PrimaryKeyDiff{
		Expected: clonePrimaryKeyMetaPtr(expected),
		Actual:   clonePrimaryKeyStatePtr(actual),
		Issues:   collectionx.NewListWithCapacity(len(issues), issues...),
	}
}

func clonePrimaryKeyMetaPtr(meta *PrimaryKeyMeta) *PrimaryKeyMeta {
	if meta == nil {
		return nil
	}
	return new(clonePrimaryKeyMeta(*meta))
}

func clonePrimaryKeyStatePtr(state *PrimaryKeyState) *PrimaryKeyState {
	if state == nil {
		return nil
	}
	return new(clonePrimaryKeyState(*state))
}

func diffIndexes(expected collectionx.List[IndexMeta], actual collectionx.List[IndexState], diff *TableDiff) {
	actualIndexes := collectionx.AssociateList(actual, func(_ int, index IndexState) (string, IndexState) {
		return indexKey(index.Unique, index.Columns), index
	})
	diff.MissingIndexes = missingByKey(expected, actualIndexes, func(index IndexMeta) string {
		return indexKey(index.Unique, index.Columns)
	})
}

func diffForeignKeys(expected collectionx.List[ForeignKeyMeta], actual collectionx.List[ForeignKeyState], diff *TableDiff) {
	actualForeignKeys := collectionx.AssociateList(actual, func(_ int, foreignKey ForeignKeyState) (string, ForeignKeyState) {
		return foreignKeyKeyFromState(foreignKey), foreignKey
	})
	diff.MissingForeignKeys = missingByKey(expected, actualForeignKeys, foreignKeyKey)
}

func diffChecks(expected collectionx.List[CheckMeta], actual collectionx.List[CheckState], diff *TableDiff) {
	actualChecks := collectionx.AssociateList(actual, func(_ int, check CheckState) (string, CheckState) {
		return checkKey(check.Expression), check
	})
	diff.MissingChecks = missingByKey(expected, actualChecks, func(check CheckMeta) string {
		return checkKey(check.Expression)
	})
}

func missingByKey[T any, S any](expected collectionx.List[T], actual collectionx.Map[string, S], key func(T) string) collectionx.List[T] {
	return collectionx.FilterList(expected, func(_ int, item T) bool {
		_, ok := actual.Get(key(item))
		return !ok
	})
}

func buildTableSpec(def schemaDefinition) TableSpec {
	indexes := deriveIndexes(def)
	foreignKeys := deriveForeignKeys(def)
	checks := deriveChecks(def)
	return TableSpec{
		Name:        def.table.name,
		Columns:     cloneColumnMetas(def.columns),
		Indexes:     collectionx.NewListWithCapacity(len(indexes), indexes...),
		PrimaryKey:  derivePrimaryKey(def),
		ForeignKeys: collectionx.NewListWithCapacity(len(foreignKeys), foreignKeys...),
		Checks:      collectionx.NewListWithCapacity(len(checks), checks...),
	}
}
