package dbx

import (
	"reflect"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/lo"
)

func deriveIndexes(def schemaDefinition) []IndexMeta {
	indexes := collectionx.NewOrderedMap[string, IndexMeta]()
	for _, index := range def.indexes {
		indexes.Set(indexKey(index.Unique, index.Columns), cloneIndexMeta(index))
	}
	deriveColumnIndexes(def, indexes)
	items := make([]IndexMeta, 0, indexes.Len())
	indexes.Range(func(_ string, value IndexMeta) bool {
		items = append(items, cloneIndexMeta(value))
		return true
	})
	return items
}

func deriveColumnIndexes(def schemaDefinition, indexes collectionx.OrderedMap[string, IndexMeta]) {
	for i := range def.columns {
		column := &def.columns[i]
		if !shouldDeriveColumnIndex(column) {
			continue
		}
		meta := IndexMeta{
			Name:    indexNameForColumn(def.table.name, *column),
			Table:   def.table.name,
			Columns: []string{column.Name},
			Unique:  column.Unique,
		}
		indexes.Set(indexKey(meta.Unique, meta.Columns), meta)
	}
}

func shouldDeriveColumnIndex(column *ColumnMeta) bool {
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

	columns := lo.FilterMap(def.columns, func(column ColumnMeta, _ int) (string, bool) {
		return column.Name, column.PrimaryKey
	})
	if len(columns) == 0 {
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
	for i := range def.columns {
		column := &def.columns[i]
		if column.References == nil {
			continue
		}
		explicitColumns.Add(column.Name)
		meta := ForeignKeyMeta{
			Name:          "fk_" + def.table.name + "_" + column.Name,
			Table:         def.table.name,
			Columns:       []string{column.Name},
			TargetTable:   column.References.TargetTable,
			TargetColumns: []string{column.References.TargetColumn},
			OnDelete:      column.References.OnDelete,
			OnUpdate:      column.References.OnUpdate,
		}
		foreignKeys.Set(foreignKeyKey(meta), meta)
	}
}

func deriveRelationForeignKeys(def schemaDefinition, foreignKeys collectionx.OrderedMap[string, ForeignKeyMeta], explicitColumns collectionx.Set[string]) {
	for i := range def.relations {
		relation := def.relations[i]
		if !shouldDeriveRelationForeignKey(def.columns, relation, explicitColumns) {
			continue
		}
		meta := ForeignKeyMeta{
			Name:          "fk_" + def.table.name + "_" + relation.LocalColumn,
			Table:         def.table.name,
			Columns:       []string{relation.LocalColumn},
			TargetTable:   relation.TargetTable,
			TargetColumns: []string{relation.TargetColumn},
		}
		key := foreignKeyKey(meta)
		if _, exists := foreignKeys.Get(key); !exists {
			foreignKeys.Set(key, meta)
		}
	}
}

func shouldDeriveRelationForeignKey(columns []ColumnMeta, relation RelationMeta, explicitColumns collectionx.Set[string]) bool {
	if relation.Kind != RelationBelongsTo {
		return false
	}
	if relation.LocalColumn == "" || relation.TargetColumn == "" || relation.TargetTable == "" {
		return false
	}
	if explicitColumns.Contains(relation.LocalColumn) {
		return false
	}
	return hasColumn(columns, relation.LocalColumn)
}

func deriveChecks(def schemaDefinition) []CheckMeta {
	return lo.Map(def.checks, func(check CheckMeta, _ int) CheckMeta {
		return cloneCheckMeta(check)
	})
}

func hasColumn(columns []ColumnMeta, name string) bool {
	return lo.SomeBy(columns, func(column ColumnMeta) bool {
		return column.Name == name
	})
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
