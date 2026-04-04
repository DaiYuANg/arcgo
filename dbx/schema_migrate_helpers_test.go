package dbx_test

import (
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
)

func toColumnState(column ColumnMeta) ColumnState {
	typeName := column.SQLType
	if typeName == "" {
		typeName = InferTypeNameForTest(column)
	}
	return ColumnState{
		Name:          column.Name,
		Type:          strings.ToLower(typeName),
		Nullable:      column.Nullable,
		PrimaryKey:    column.PrimaryKey,
		AutoIncrement: column.AutoIncrement,
		DefaultValue:  column.DefaultValue,
	}
}

func toIndexStates(indexes collectionx.List[IndexMeta]) collectionx.List[IndexState] {
	items := collectionx.NewListWithCapacity[IndexState](indexes.Len())
	indexes.Range(func(_ int, index IndexMeta) bool {
		items.Add(IndexState{
			Name:    index.Name,
			Columns: index.Columns.Clone(),
			Unique:  index.Unique,
		})
		return true
	})
	return items
}

func toForeignKeyStates(foreignKeys collectionx.List[ForeignKeyMeta]) collectionx.List[ForeignKeyState] {
	items := collectionx.NewListWithCapacity[ForeignKeyState](foreignKeys.Len())
	foreignKeys.Range(func(_ int, foreignKey ForeignKeyMeta) bool {
		items.Add(ForeignKeyState{
			Name:          foreignKey.Name,
			Columns:       foreignKey.Columns.Clone(),
			TargetTable:   foreignKey.TargetTable,
			TargetColumns: foreignKey.TargetColumns.Clone(),
			OnDelete:      foreignKey.OnDelete,
			OnUpdate:      foreignKey.OnUpdate,
		})
		return true
	})
	return items
}

func toCheckStates(checks collectionx.List[CheckMeta]) collectionx.List[CheckState] {
	items := collectionx.NewListWithCapacity[CheckState](checks.Len())
	checks.Range(func(_ int, check CheckMeta) bool {
		items.Add(CheckState{Name: check.Name, Expression: check.Expression})
		return true
	})
	return items
}
