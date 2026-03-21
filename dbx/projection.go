package dbx

import (
	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/lo"
)

type fieldMapper interface {
	Fields() []MappedField
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

	items := collectionx.NewListWithCapacity[SelectItem](len(fields))
	for _, field := range fields {
		column, ok := columns[field.Column]
		if !ok {
			continue
		}
		items.Add(schemaSelectItem{meta: column})
	}
	if items.Len() != len(fields) {
		for _, field := range fields {
			if _, ok := columns[field.Column]; !ok {
				return nil, &UnmappedColumnError{Column: field.Column}
			}
		}
	}
	return items.Values(), nil
}
