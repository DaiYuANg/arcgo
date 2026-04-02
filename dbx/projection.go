package dbx

import (
	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/lo"
)

type fieldMapper interface {
	Fields() collectionx.List[MappedField]
}

func ProjectionOf(schema SchemaResource, mapper fieldMapper) ([]SelectItem, error) {
	return projectionOfDefinition(schema.schemaRef(), mapper)
}

func MustProjectionOf(schema SchemaResource, mapper fieldMapper) []SelectItem {
	items, err := projectionOfDefinition(schema.schemaRef(), mapper)
	if err != nil {
		panic(err)
	}
	return items
}

func SelectMapped(schema SchemaResource, mapper fieldMapper) (*SelectQuery, error) {
	items, err := projectionOfDefinition(schema.schemaRef(), mapper)
	if err != nil {
		return nil, err
	}
	return Select(items...).From(schema), nil
}

func MustSelectMapped(schema SchemaResource, mapper fieldMapper) *SelectQuery {
	items, err := projectionOfDefinition(schema.schemaRef(), mapper)
	if err != nil {
		panic(err)
	}
	return Select(items...).From(schema)
}

func projectionOfDefinition(definition schemaDefinition, mapper fieldMapper) ([]SelectItem, error) {
	fields := mapper.Fields()
	columns := lo.Associate(definition.columns, func(column ColumnMeta) (string, ColumnMeta) {
		return column.Name, column
	})

	if unmapped, ok := collectionx.FindList(fields, func(_ int, field MappedField) bool {
		_, ok := columns[field.Column]
		return !ok
	}); ok {
		return nil, &UnmappedColumnError{Column: unmapped.Column}
	}

	return collectionx.FilterMapList(fields, func(_ int, field MappedField) (SelectItem, bool) {
		column, ok := columns[field.Column]
		if !ok {
			return nil, false
		}
		return schemaSelectItem{meta: column}, true
	}).Values(), nil
}
