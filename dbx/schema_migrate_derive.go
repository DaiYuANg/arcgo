package dbx

import (
	"reflect"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
)

func deriveIndexes(def schemaDefinition) []IndexMeta {
	indexes := collectionx.NewOrderedMap[string, IndexMeta]()
	def.indexes.Range(func(_ int, index IndexMeta) bool {
		indexes.Set(indexKey(index.Unique, index.Columns), cloneIndexMeta(index))
		return true
	})
	deriveColumnIndexes(def, indexes)
	items := make([]IndexMeta, 0, indexes.Len())
	indexes.Range(func(_ string, value IndexMeta) bool {
		items = append(items, cloneIndexMeta(value))
		return true
	})
	return items
}

func deriveColumnIndexes(def schemaDefinition, indexes collectionx.OrderedMap[string, IndexMeta]) {
	def.columns.Range(func(_ int, column ColumnMeta) bool {
		if !shouldDeriveColumnIndex(column) {
			return true
		}
		meta := IndexMeta{
			Name:    indexNameForColumn(def.table.name, column),
			Table:   def.table.name,
			Columns: collectionx.NewList(column.Name),
			Unique:  column.Unique,
		}
		indexes.Set(indexKey(meta.Unique, meta.Columns), meta)
		return true
	})
}

func shouldDeriveColumnIndex(column ColumnMeta) bool {
	return !column.PrimaryKey && (column.Unique || column.Indexed)
}

func indexNameForColumn(table string, column ColumnMeta) string {
	prefix := "idx"
	if column.Unique {
		prefix = "ux"
	}
	return prefix + "_" + table + "_" + column.Name
}

func derivePrimaryKey(def schemaDefinition) *PrimaryKeyMeta {
	if def.primaryKey != nil {
		copyPrimary := clonePrimaryKeyMeta(*def.primaryKey)
		if copyPrimary.Name == "" {
			copyPrimary.Name = "pk_" + def.table.name
		}
		if copyPrimary.Table == "" {
			copyPrimary.Table = def.table.name
		}
		return &copyPrimary
	}

	columns := collectionx.FilterMapList(def.columns, func(_ int, column ColumnMeta) (string, bool) {
		return column.Name, column.PrimaryKey
	})
	if columns.Len() == 0 {
		return nil
	}
	return &PrimaryKeyMeta{
		Name:    "pk_" + def.table.name,
		Table:   def.table.name,
		Columns: columns,
	}
}

func deriveForeignKeys(def schemaDefinition) []ForeignKeyMeta {
	foreignKeys := collectionx.NewOrderedMap[string, ForeignKeyMeta]()
	explicitColumns := collectionx.NewSet[string]()
	deriveExplicitForeignKeys(def, foreignKeys, explicitColumns)
	deriveRelationForeignKeys(def, foreignKeys, explicitColumns)
	items := make([]ForeignKeyMeta, 0, foreignKeys.Len())
	foreignKeys.Range(func(_ string, value ForeignKeyMeta) bool {
		items = append(items, cloneForeignKeyMeta(value))
		return true
	})
	return items
}

func deriveExplicitForeignKeys(def schemaDefinition, foreignKeys collectionx.OrderedMap[string, ForeignKeyMeta], explicitColumns collectionx.Set[string]) {
	def.columns.Range(func(_ int, column ColumnMeta) bool {
		if column.References == nil {
			return true
		}
		explicitColumns.Add(column.Name)
		meta := ForeignKeyMeta{
			Name:          "fk_" + def.table.name + "_" + column.Name,
			Table:         def.table.name,
			Columns:       collectionx.NewList(column.Name),
			TargetTable:   column.References.TargetTable,
			TargetColumns: collectionx.NewList(column.References.TargetColumn),
			OnDelete:      column.References.OnDelete,
			OnUpdate:      column.References.OnUpdate,
		}
		foreignKeys.Set(foreignKeyKey(meta), meta)
		return true
	})
}

func deriveRelationForeignKeys(def schemaDefinition, foreignKeys collectionx.OrderedMap[string, ForeignKeyMeta], explicitColumns collectionx.Set[string]) {
	def.relations.Range(func(_ int, relation RelationMeta) bool {
		if !shouldDeriveRelationForeignKey(def, relation, explicitColumns) {
			return true
		}
		meta := ForeignKeyMeta{
			Name:          "fk_" + def.table.name + "_" + relation.LocalColumn,
			Table:         def.table.name,
			Columns:       collectionx.NewList(relation.LocalColumn),
			TargetTable:   relation.TargetTable,
			TargetColumns: collectionx.NewList(relation.TargetColumn),
		}
		key := foreignKeyKey(meta)
		if _, exists := foreignKeys.Get(key); !exists {
			foreignKeys.Set(key, meta)
		}
		return true
	})
}

func shouldDeriveRelationForeignKey(def schemaDefinition, relation RelationMeta, explicitColumns collectionx.Set[string]) bool {
	if relation.Kind != RelationBelongsTo {
		return false
	}
	if relation.LocalColumn == "" || relation.TargetColumn == "" || relation.TargetTable == "" {
		return false
	}
	if explicitColumns.Contains(relation.LocalColumn) {
		return false
	}
	_, ok := def.columnByName(relation.LocalColumn)
	return ok
}

func deriveChecks(def schemaDefinition) []CheckMeta {
	return collectionx.MapList(def.checks, func(_ int, check CheckMeta) CheckMeta {
		return cloneCheckMeta(check)
	}).Values()
}

func normalizeExpectedType(schemaDialect SchemaDialect, column ColumnMeta) string {
	if column.SQLType != "" {
		return schemaDialect.NormalizeType(column.SQLType)
	}
	return schemaDialect.NormalizeType(inferTypeName(column))
}

func inferTypeName(column ColumnMeta) string {
	if column.SQLType != "" {
		return column.SQLType
	}
	return inferTypeNameFromGoType(column.GoType)
}

func inferTypeNameFromGoType(goType reflect.Type) string {
	if goType == nil {
		return ""
	}
	typ := indirectGoType(goType)
	if isTimeGoType(typ) {
		return "timestamp"
	}
	if typeName, ok := inferBasicTypeName(typ); ok {
		return typeName
	}
	if isByteSliceType(typ) {
		return "blob"
	}
	return strings.ToLower(typ.Name())
}

func indirectGoType(typ reflect.Type) reflect.Type {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ
}

func isTimeGoType(typ reflect.Type) bool {
	return typ.PkgPath() == "time" && typ.Name() == "Time"
}

func inferBasicTypeName(typ reflect.Type) (string, bool) {
	kind := typ.Kind()
	if kind == reflect.Bool {
		return "boolean", true
	}
	if isSignedIntKind(kind) {
		return "integer", true
	}
	if kind == reflect.Int64 {
		return "bigint", true
	}
	if isUnsignedIntKind(kind) {
		return "integer", true
	}
	if kind == reflect.Uint64 {
		return "bigint", true
	}
	if kind == reflect.Float32 {
		return "real", true
	}
	if kind == reflect.Float64 {
		return "double", true
	}
	if kind == reflect.String {
		return "text", true
	}
	return "", false
}

func normalizeDefault(value string) string {
	return strings.TrimSpace(strings.Trim(value, "()"))
}

func normalizeCheckExpression(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}
