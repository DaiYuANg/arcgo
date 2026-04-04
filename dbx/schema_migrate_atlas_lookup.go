package dbx

import (
	"fmt"

	atlasschema "ariga.io/atlas/sql/schema"
	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/lo"
)

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
		Columns:       collectionx.NewList(columns...),
		TargetTable:   lo.If(foreignKey.RefTable != nil, foreignKey.RefTable.Name).Else(""),
		TargetColumns: collectionx.NewList(targetColumns...),
		OnDelete:      ReferentialAction(foreignKey.OnDelete),
		OnUpdate:      ReferentialAction(foreignKey.OnUpdate),
	}
	return foreignKeyKey(meta)
}

func atlasIndexColumns(index *atlasschema.Index) collectionx.List[string] {
	columns := lo.FilterMap(index.Parts, func(part *atlasschema.IndexPart, _ int) (string, bool) {
		if part == nil || part.C == nil {
			return "", false
		}
		return part.C.Name, true
	})
	return collectionx.NewList(columns...)
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
