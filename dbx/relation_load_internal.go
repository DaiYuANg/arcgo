package dbx

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/lo"
)

type relationLookupValue struct {
	present bool
	key     any
}

type relationKeyPair struct {
	source any
	target any
}

func collectSourceRelationKeys[E any](entities []E, mapper Mapper[E], schema schemaDefinition, meta RelationMeta) ([]any, []relationLookupValue, error) {
	localColumn, err := relationSourceColumn(schemaAdapter[E]{def: schema}, meta)
	if err != nil {
		return nil, nil, err
	}

	lookup := make([]relationLookupValue, len(entities))
	keys := collectionx.NewListWithCapacity[any](len(entities))
	seen := make(map[any]struct{}, len(entities))
	for index := range entities {
		key, err := entityRelationKey(mapper, &entities[index], localColumn.Name)
		if err != nil {
			return nil, nil, err
		}
		lookup[index] = key
		if !key.present {
			continue
		}
		if _, ok := seen[key.key]; ok {
			continue
		}
		seen[key.key] = struct{}{}
		keys.Add(key.key)
	}
	return keys.Values(), lookup, nil
}

func entityRelationKey[E any](mapper Mapper[E], entity *E, column string) (relationLookupValue, error) {
	field, ok := mapper.FieldByColumn(column)
	if !ok {
		return relationLookupValue{}, fmt.Errorf("%w: %s", ErrUnmappedColumn, column)
	}

	value, err := mapper.entityValue(entity)
	if err != nil {
		return relationLookupValue{}, err
	}
	fieldValue, err := fieldValueForRead(value, field)
	if err != nil {
		return relationLookupValue{}, err
	}
	return normalizeRelationLookupValue(normalizeFieldValue(fieldValue))
}

func normalizeRelationLookupValue(value any) (relationLookupValue, error) {
	if value == nil {
		return relationLookupValue{}, nil
	}

	current := reflect.ValueOf(value)
	for current.IsValid() && current.Kind() == reflect.Pointer {
		if current.IsNil() {
			return relationLookupValue{}, nil
		}
		current = current.Elem()
	}
	if !current.IsValid() {
		return relationLookupValue{}, nil
	}
	if !current.Type().Comparable() {
		return relationLookupValue{}, fmt.Errorf("dbx: relation key type %s is not comparable", current.Type())
	}
	return relationLookupValue{present: true, key: current.Interface()}, nil
}

func relationTargetColumnForSchema(schema relationSchemaSource, meta RelationMeta) (ColumnMeta, error) {
	name := meta.TargetColumn
	if name == "" {
		primaryKey := derivePrimaryKey(schema.schemaRef())
		if primaryKey == nil || len(primaryKey.Columns) != 1 {
			return ColumnMeta{}, fmt.Errorf("dbx: relation %s requires target column or single-column primary key", meta.Name)
		}
		name = primaryKey.Columns[0]
	}

	column, ok := sourceColumnByName(schema.schemaRef(), name)
	if !ok {
		return ColumnMeta{}, fmt.Errorf("dbx: relation %s target column %s not found", meta.Name, name)
	}
	return column, nil
}

func queryRelationTargets[E any](ctx context.Context, session Session, schema SchemaSource[E], mapper Mapper[E], targetColumn ColumnMeta, keys []any) ([]E, error) {
	query := Select(allSelectItems(schema.schemaRef())...).
		From(schema).
		Where(metadataComparisonPredicate{
			left:  targetColumn,
			op:    OpIn,
			right: keys,
		})
	return QueryAll(ctx, session, query, mapper)
}

func allSelectItems(def schemaDefinition) []SelectItem {
	return lo.Map(def.columns, func(column ColumnMeta, _ int) SelectItem {
		return schemaSelectItem{meta: column}
	})
}

func indexRelationTargets[E any](targets []E, mapper Mapper[E], column string) (map[any]E, error) {
	indexed := make(map[any]E, len(targets))
	for index := range targets {
		key, err := entityRelationKey(mapper, &targets[index], column)
		if err != nil {
			return nil, err
		}
		if !key.present {
			continue
		}
		indexed[key.key] = targets[index]
	}
	return indexed, nil
}

func groupRelationTargets[E any](targets []E, mapper Mapper[E], column string) (map[any][]E, error) {
	grouped := make(map[any][]E, len(targets))
	for index := range targets {
		key, err := entityRelationKey(mapper, &targets[index], column)
		if err != nil {
			return nil, err
		}
		if !key.present {
			continue
		}
		grouped[key.key] = append(grouped[key.key], targets[index])
	}
	return grouped, nil
}

func relationKeyTypeForMeta(def schemaDefinition, column string) reflect.Type {
	if column == "" {
		primaryKey := derivePrimaryKey(def)
		if primaryKey == nil || len(primaryKey.Columns) != 1 {
			return nil
		}
		column = primaryKey.Columns[0]
	}
	columnMeta, ok := sourceColumnByName(def, column)
	if !ok {
		return nil
	}
	return columnMeta.GoType
}

func queryManyToManyPairs(ctx context.Context, session Session, meta RelationMeta, sourceKeys []any, sourceType, targetType reflect.Type) ([]relationKeyPair, error) {
	if meta.ThroughTable == "" {
		return nil, fmt.Errorf("dbx: many-to-many relation %s requires join table", meta.Name)
	}
	if meta.ThroughLocalColumn == "" || meta.ThroughTargetColumn == "" {
		return nil, fmt.Errorf("dbx: many-to-many relation %s requires join_local and join_target", meta.Name)
	}

	through := Table{def: tableDefinition{name: meta.ThroughTable}}
	localColumn := ColumnMeta{Name: meta.ThroughLocalColumn, Table: through.Name(), GoType: sourceType}
	targetColumn := ColumnMeta{Name: meta.ThroughTargetColumn, Table: through.Name(), GoType: targetType}
	query := Select(
		schemaSelectItem{meta: localColumn},
		schemaSelectItem{meta: targetColumn},
	).From(through).Where(metadataComparisonPredicate{
		left:  localColumn,
		op:    OpIn,
		right: sourceKeys,
	})

	bound, err := Build(session, query)
	if err != nil {
		return nil, err
	}
	rows, err := session.QueryBoundContext(ctx, bound)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRelationPairs(rows, sourceType, targetType)
}

func scanRelationPairs(rows *sql.Rows, sourceType, targetType reflect.Type) ([]relationKeyPair, error) {
	pairs := collectionx.NewList[relationKeyPair]()
	for rows.Next() {
		sourceDest, sourceValue := relationScanDestination(sourceType)
		targetDest, targetValue := relationScanDestination(targetType)
		if err := rows.Scan(sourceDest, targetDest); err != nil {
			return nil, err
		}

		sourceKey, err := normalizeRelationLookupValue(sourceValue())
		if err != nil {
			return nil, err
		}
		targetKey, err := normalizeRelationLookupValue(targetValue())
		if err != nil {
			return nil, err
		}
		if !sourceKey.present || !targetKey.present {
			continue
		}
		pairs.Add(relationKeyPair{source: sourceKey.key, target: targetKey.key})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return pairs.Values(), nil
}

func relationScanDestination(typ reflect.Type) (any, func() any) {
	baseType := typ
	for baseType != nil && baseType.Kind() == reflect.Pointer {
		baseType = baseType.Elem()
	}
	if baseType == nil {
		var value any
		return &value, func() any { return value }
	}
	holder := reflect.New(baseType)
	return holder.Interface(), func() any { return holder.Elem().Interface() }
}

func uniqueRelationKeysFromPairs(pairs []relationKeyPair, useSource bool) []any {
	keys := collectionx.NewListWithCapacity[any](len(pairs))
	seen := make(map[any]struct{}, len(pairs))
	for _, pair := range pairs {
		key := pair.target
		if useSource {
			key = pair.source
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys.Add(key)
	}
	return keys.Values()
}

func groupManyToManyTargets[E any](pairs []relationKeyPair, indexed map[any]E) map[any][]E {
	grouped := make(map[any][]E, len(pairs))
	for _, pair := range pairs {
		target, ok := indexed[pair.target]
		if !ok {
			continue
		}
		grouped[pair.source] = append(grouped[pair.source], target)
	}
	return grouped
}

type schemaAdapter[E any] struct {
	def schemaDefinition
}

func (s schemaAdapter[E]) tableRef() Table {
	return Table{def: s.def.table}
}

func (s schemaAdapter[E]) schemaRef() schemaDefinition {
	return s.def
}
