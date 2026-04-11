package dbx

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/DaiYuANg/arcgo/collectionx"
)

func (m Mapper[E]) InsertAssignments(session Session, schema SchemaResource, entity *E) (collectionx.List[Assignment], error) {
	if session == nil {
		return nil, ErrNilDB
	}
	carrier, ok := any(session).(interface{ IDGenerator() IDGenerator })
	if !ok {
		return nil, errors.New("dbx: session does not expose id generator")
	}
	return m.InsertAssignmentsWithID(context.Background(), schema, entity, carrier.IDGenerator())
}

func (m Mapper[E]) InsertAssignmentsWithID(ctx context.Context, schema SchemaResource, entity *E, generator IDGenerator) (collectionx.List[Assignment], error) {
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

func (m Mapper[E]) UpdateAssignments(schema SchemaResource, entity *E) (collectionx.List[Assignment], error) {
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

	var predicate Predicate
	var resultErr error
	schema.schemaRef().columns.Range(func(_ int, column ColumnMeta) bool {
		if !column.PrimaryKey {
			return true
		}
		field, ok := m.byColumn.Get(column.Name)
		if !ok {
			resultErr = &PrimaryKeyUnmappedError{Column: column.Name}
			return false
		}
		fieldValue, err := fieldValueForRead(value, field)
		if err != nil {
			resultErr = err
			return false
		}
		boundValue, err := boundFieldValue(field, fieldValue)
		if err != nil {
			resultErr = err
			return false
		}
		predicate = metadataComparisonPredicate{
			left:  column,
			op:    OpEq,
			right: boundValue,
		}
		return false
	})
	if resultErr != nil {
		return nil, resultErr
	}
	if predicate != nil {
		return predicate, nil
	}

	return nil, ErrNoPrimaryKey
}

func (m Mapper[E]) entityAssignments(ctx context.Context, schema SchemaResource, entity *E, generator IDGenerator, include func(column ColumnMeta, field MappedField) bool) (collectionx.List[Assignment], error) {
	value, err := m.entityValue(entity)
	if err != nil {
		return nil, err
	}

	def := schema.schemaRef()
	assignments := collectionx.NewListWithCapacity[Assignment](def.columns.Len())
	var resultErr error
	def.columns.Range(func(_ int, column ColumnMeta) bool {
		field, ok := m.byColumn.Get(column.Name)
		if !ok || !include(column, field) {
			return true
		}
		assignment, ok, err := m.buildAssignment(ctx, value, column, field, generator)
		if err != nil {
			resultErr = err
			return false
		}
		if ok {
			assignments.Add(assignment)
		}
		return true
	})
	if resultErr != nil {
		return nil, resultErr
	}

	return assignments, nil
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
		return reflect.Value{}, false, fmt.Errorf("dbx: generate id for column %s: %w", column.Name, err)
	}

	targetField, err := ensureFieldValue(root, field)
	if err != nil {
		return reflect.Value{}, false, err
	}
	if !targetField.CanSet() {
		return reflect.Value{}, false, fmt.Errorf("dbx: cannot set generated id for column %s", column.Name)
	}

	assignedField, assigned, setErr := setGeneratedValue(targetField, generated, column)
	if setErr != nil {
		return reflect.Value{}, false, setErr
	}
	if assigned {
		return assignedField, true, nil
	}
	return reflect.Value{}, false, fmt.Errorf("dbx: generated id type %s cannot be assigned to %s for column %s", reflect.TypeOf(generated), targetField.Type(), column.Name)
}

func setGeneratedValue(targetField reflect.Value, generated any, column ColumnMeta) (reflect.Value, bool, error) {
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
	return reflect.Value{}, false, nil
}

func (m Mapper[E]) entityValue(entity *E) (reflect.Value, error) {
	if entity == nil {
		return reflect.Value{}, ErrNilEntity
	}
	return reflect.ValueOf(entity).Elem(), nil
}

func (m Mapper[E]) buildAssignment(ctx context.Context, root reflect.Value, column ColumnMeta, field MappedField, generator IDGenerator) (Assignment, bool, error) {
	if column.PrimaryKey && shouldGenerateID(column) {
		return m.generatedOrExistingAssignment(ctx, root, column, field, generator)
	}
	return buildFieldAssignment(root, column, field)
}

func (m Mapper[E]) generatedOrExistingAssignment(ctx context.Context, root reflect.Value, column ColumnMeta, field MappedField, generator IDGenerator) (Assignment, bool, error) {
	fieldValue, generated, err := m.ensureGeneratedID(ctx, root, field, column, generator)
	if err != nil {
		return nil, false, err
	}
	if generated {
		return assignmentFromValue(column, field, fieldValue)
	}
	return buildFieldAssignment(root, column, field)
}

func buildFieldAssignment(root reflect.Value, column ColumnMeta, field MappedField) (Assignment, bool, error) {
	fieldValue, err := fieldValueForRead(root, field)
	if err != nil {
		return nil, false, err
	}
	return assignmentFromValue(column, field, fieldValue)
}

func assignmentFromValue(column ColumnMeta, field MappedField, fieldValue reflect.Value) (Assignment, bool, error) {
	boundValue, err := boundFieldValue(field, fieldValue)
	if err != nil {
		return nil, false, err
	}
	return metadataAssignment{meta: column, value: boundValue}, true, nil
}
