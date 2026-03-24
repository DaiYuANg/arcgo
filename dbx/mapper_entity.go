package dbx

import (
	"context"
	"fmt"
	"reflect"

	"github.com/DaiYuANg/arcgo/collectionx"
)

func (m Mapper[E]) InsertAssignments(session Session, schema SchemaResource, entity *E) ([]Assignment, error) {
	if session == nil {
		return nil, ErrNilDB
	}
	carrier, ok := any(session).(interface{ IDGenerator() IDGenerator })
	if !ok {
		return nil, fmt.Errorf("dbx: session does not expose id generator")
	}
	return m.InsertAssignmentsWithID(context.Background(), schema, entity, carrier.IDGenerator())
}

func (m Mapper[E]) InsertAssignmentsWithID(ctx context.Context, schema SchemaResource, entity *E, generator IDGenerator) ([]Assignment, error) {
	return m.entityAssignments(ctx, schema, entity, generator, func(column ColumnMeta, field MappedField) bool {
		if !field.Insertable {
			return false
		}
		if !column.PrimaryKey {
			return true
		}
		return column.IDStrategy != IDStrategyDBAuto && !column.AutoIncrement
	})
}

func (m Mapper[E]) UpdateAssignments(schema SchemaResource, entity *E) ([]Assignment, error) {
	return m.entityAssignments(context.Background(), schema, entity, nil, func(column ColumnMeta, field MappedField) bool {
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

func (m Mapper[E]) entityAssignments(ctx context.Context, schema SchemaResource, entity *E, generator IDGenerator, include func(column ColumnMeta, field MappedField) bool) ([]Assignment, error) {
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
		if column.PrimaryKey && shouldGenerateID(column) {
			fieldValue, generated, genErr := m.ensureGeneratedID(ctx, value, field, column, generator)
			if genErr != nil {
				return nil, genErr
			}
			if generated {
				boundValue, boundErr := boundFieldValue(field, fieldValue)
				if boundErr != nil {
					return nil, boundErr
				}
				assignments.Add(metadataAssignment{
					meta:  column,
					value: boundValue,
				})
				continue
			}
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

func shouldGenerateID(column ColumnMeta) bool {
	return column.IDStrategy == IDStrategySnowflake ||
		column.IDStrategy == IDStrategyUUID ||
		column.IDStrategy == IDStrategyULID ||
		column.IDStrategy == IDStrategyKSUID
}

func (m Mapper[E]) ensureGeneratedID(ctx context.Context, root reflect.Value, field MappedField, column ColumnMeta, generator IDGenerator) (reflect.Value, bool, error) {
	fieldValue, err := fieldValueForRead(root, field)
	if err != nil {
		return reflect.Value{}, false, err
	}
	if fieldValue.IsValid() && !fieldValue.IsZero() {
		return fieldValue, false, nil
	}
	if generator == nil {
		return reflect.Value{}, false, fmt.Errorf("dbx: id generator is nil for column %s", column.Name)
	}
	generated, err := generator.GenerateID(ctx, column)
	if err != nil {
		return reflect.Value{}, false, err
	}

	targetField, err := ensureFieldValue(root, field)
	if err != nil {
		return reflect.Value{}, false, err
	}
	if !targetField.CanSet() {
		return reflect.Value{}, false, fmt.Errorf("dbx: cannot set generated id for column %s", column.Name)
	}

	generatedValue := reflect.ValueOf(generated)
	if !generatedValue.IsValid() {
		return reflect.Value{}, false, fmt.Errorf("dbx: generated id is invalid for column %s", column.Name)
	}
	if generatedValue.Type().AssignableTo(targetField.Type()) {
		targetField.Set(generatedValue)
		return targetField, true, nil
	}
	if generatedValue.Type().ConvertibleTo(targetField.Type()) {
		targetField.Set(generatedValue.Convert(targetField.Type()))
		return targetField, true, nil
	}
	return reflect.Value{}, false, fmt.Errorf("dbx: generated id type %s cannot be assigned to %s for column %s", generatedValue.Type(), targetField.Type(), column.Name)
}

func (m Mapper[E]) entityValue(entity *E) (reflect.Value, error) {
	if entity == nil {
		return reflect.Value{}, ErrNilEntity
	}
	return reflect.ValueOf(entity).Elem(), nil
}
