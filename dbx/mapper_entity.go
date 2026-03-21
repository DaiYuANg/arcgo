package dbx

import (
	"reflect"

	"github.com/DaiYuANg/arcgo/collectionx"
)

func (m Mapper[E]) InsertAssignments(schema SchemaResource, entity *E) ([]Assignment, error) {
	return m.entityAssignments(schema, entity, func(column ColumnMeta, field MappedField) bool {
		if !field.Insertable {
			return false
		}
		return !(column.PrimaryKey && column.AutoIncrement)
	})
}

func (m Mapper[E]) UpdateAssignments(schema SchemaResource, entity *E) ([]Assignment, error) {
	return m.entityAssignments(schema, entity, func(column ColumnMeta, field MappedField) bool {
		if !field.Updatable {
			return false
		}
		return !column.PrimaryKey && !column.AutoIncrement
	})
}

func (m Mapper[E]) PrimaryPredicate(schema SchemaResource, entity *E) (Predicate, error) {
	value, err := m.entityValue(entity)
	if err != nil {
		return nil, err
	}

	for _, column := range schema.schemaRef().columns {
		if !column.PrimaryKey {
			continue
		}
		field, ok := m.byColumn.Get(column.Name)
		if !ok {
			return nil, &PrimaryKeyUnmappedError{Column: column.Name}
		}
		fieldValue, err := fieldValueForRead(value, field)
		if err != nil {
			return nil, err
		}
		boundValue, err := boundFieldValue(field, fieldValue)
		if err != nil {
			return nil, err
		}
		return metadataComparisonPredicate{
			left:  column,
			op:    OpEq,
			right: boundValue,
		}, nil
	}

	return nil, ErrNoPrimaryKey
}

func (m Mapper[E]) entityAssignments(schema SchemaResource, entity *E, include func(column ColumnMeta, field MappedField) bool) ([]Assignment, error) {
	value, err := m.entityValue(entity)
	if err != nil {
		return nil, err
	}

	assignments := collectionx.NewListWithCapacity[Assignment](len(schema.schemaRef().columns))
	for _, column := range schema.schemaRef().columns {
		field, ok := m.byColumn.Get(column.Name)
		if !ok || !include(column, field) {
			continue
		}
		fieldValue, err := fieldValueForRead(value, field)
		if err != nil {
			return nil, err
		}
		boundValue, err := boundFieldValue(field, fieldValue)
		if err != nil {
			return nil, err
		}
		assignments.Add(metadataAssignment{
			meta:  column,
			value: boundValue,
		})
	}

	return assignments.Values(), nil
}

func (m Mapper[E]) entityValue(entity *E) (reflect.Value, error) {
	if entity == nil {
		return reflect.Value{}, ErrNilEntity
	}
	return reflect.ValueOf(entity).Elem(), nil
}
