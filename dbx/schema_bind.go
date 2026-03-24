package dbx

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
)

func bindSchema[S any](name, alias string, schema S) (S, error) {
	value := reflect.ValueOf(&schema).Elem()
	if value.Kind() != reflect.Struct {
		return schema, fmt.Errorf("dbx: schema must be a struct")
	}

	var binder schemaBinder
	var binderField reflect.Value
	schemaType := value.Type()
	defTable := tableDefinition{
		name:       strings.TrimSpace(name),
		alias:      strings.TrimSpace(alias),
		schemaType: schemaType,
	}
	columns := collectionx.NewListWithCapacity[ColumnMeta](value.NumField())
	relations := collectionx.NewListWithCapacity[RelationMeta](value.NumField())
	indexes := collectionx.NewListWithCapacity[IndexMeta](value.NumField())
	checks := collectionx.NewListWithCapacity[CheckMeta](value.NumField())
	var primaryKey *PrimaryKeyMeta

	for i := 0; i < value.NumField(); i++ {
		fieldValue := value.Field(i)
		fieldType := schemaType.Field(i)
		if !fieldValue.CanSet() {
			continue
		}

		if candidate, ok := fieldValue.Interface().(schemaBinder); ok {
			binder = candidate
			binderField = fieldValue
			defTable.entityType = candidate.entityType()
			continue
		}

		if candidate, ok := fieldValue.Interface().(columnBinder); ok {
			meta, metaErr := resolveColumnMeta(defTable, fieldType, fieldValue.Interface())
			if metaErr != nil {
				return schema, metaErr
			}
			bound := candidate.bindColumn(columnBinding{meta: meta})
			fieldValue.Set(reflect.ValueOf(bound))
			if accessor, ok := bound.(columnAccessor); ok {
				columns.Add(accessor.columnRef())
			} else {
				columns.Add(meta)
			}
			continue
		}

		if candidate, ok := fieldValue.Interface().(relationBinder); ok {
			meta := resolveRelationMeta(defTable, fieldType, candidate)
			fieldValue.Set(reflect.ValueOf(candidate.bindRelation(relationBinding{meta: meta})))
			relations.Add(meta)
			continue
		}

		if candidate, ok := fieldValue.Interface().(constraintBinder); ok {
			binding, bindErr := resolveConstraintBinding(defTable, fieldType, fieldValue.Interface())
			if bindErr != nil {
				return schema, bindErr
			}
			fieldValue.Set(reflect.ValueOf(candidate.bindConstraint(binding)))
			indexes.MergeSlice(binding.indexes)
			if binding.primaryKey != nil {
				primaryKey = new(clonePrimaryKeyMeta(*binding.primaryKey))
			}
			if binding.check != nil {
				checks.Add(*binding.check)
			}
		}
	}

	if binder == nil {
		return schema, fmt.Errorf("dbx: schema %s must embed dbx.Schema[T]", schemaType.Name())
	}

	binderField.Set(reflect.ValueOf(binder.bindSchema(schemaDefinition{
		table:      defTable,
		columns:    columns.Values(),
		relations:  relations.Values(),
		indexes:    indexes.Values(),
		primaryKey: primaryKey,
		checks:     checks.Values(),
	})))
	return schema, nil
}

func resolveColumnMeta(def tableDefinition, field reflect.StructField, value any) (ColumnMeta, error) {
	name, options := resolveTagNameAndOptions(field)
	meta := ColumnMeta{
		Name:          name,
		Table:         def.name,
		Alias:         def.alias,
		FieldName:     field.Name,
		GoType:        resolveColumnGoType(value),
		SQLType:       optionValue(options, "type"),
		PrimaryKey:    optionEnabled(options, "pk"),
		AutoIncrement: optionEnabled(options, "auto") || optionEnabled(options, "autoincrement"),
		Nullable:      optionEnabled(options, "nullable") || optionEnabled(options, "null"),
		Unique:        optionEnabled(options, "unique"),
		Indexed:       optionEnabled(options, "index") || optionEnabled(options, "indexed"),
		DefaultValue:  optionValue(options, "default"),
	}

	if refValue := optionValue(options, "ref"); refValue != "" {
		targetTable, targetColumn, ok := splitReference(refValue)
		if ok {
			meta.References = &ForeignKeyRef{
				TargetTable:  targetTable,
				TargetColumn: targetColumn,
				OnDelete:     parseReferentialAction(optionValue(options, "ondelete")),
				OnUpdate:     parseReferentialAction(optionValue(options, "onupdate")),
			}
		}
	}

	return normalizeIDPolicy(meta)
}

func resolveColumnGoType(value any) reflect.Type {
	reporter, ok := value.(columnTypeReporter)
	if !ok {
		return nil
	}
	return reporter.valueType()
}

func resolveRelationMeta(def tableDefinition, field reflect.StructField, binder relationBinder) RelationMeta {
	options := parseTagOptions(field.Tag.Get("rel"))
	name := optionValue(options, "name")
	if name == "" {
		name = toSnakeCase(field.Name)
	}
	return RelationMeta{
		Name:                name,
		FieldName:           field.Name,
		Kind:                binder.relationKind(),
		SourceTable:         def.name,
		SourceAlias:         def.alias,
		TargetTable:         optionValue(options, "table"),
		LocalColumn:         optionValue(options, "local"),
		TargetColumn:        optionValue(options, "target"),
		ThroughTable:        optionValue(options, "join"),
		ThroughLocalColumn:  optionValue(options, "join_local"),
		ThroughTargetColumn: optionValue(options, "join_target"),
		TargetType:          binder.targetType(),
	}
}
