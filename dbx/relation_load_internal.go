package dbx

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

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

func collectSourceRelationKeys[E any](rt *relationRuntime, entities []E, mapper Mapper[E], schema schemaDefinition, meta RelationMeta) ([]any, []relationLookupValue, error) {
	localColumn, err := relationSourceColumn(schemaAdapter[E]{def: schema}, meta)
	if err != nil {
		return nil, nil, err
	}

	lookup := make([]relationLookupValue, len(entities))
	keys := collectionx.NewListWithCapacity[any](len(entities))
	seen, err := relationSeenSet(rt)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		seen.Clear()
		rt.seenSetPool.Put(seen)
	}()
	for index := range entities {
		key, err := entityRelationKey(mapper, &entities[index], localColumn.Name)
		if err != nil {
			return nil, nil, err
		}
		lookup[index] = key
		if !key.present {
			continue
		}
		if _, ok := seen.Get(key.key); ok {
			continue
		}
		seen.Set(key.key, struct{}{})
		keys.Add(key.key)
	}
	return keys.Values(), lookup, nil
}

func entityRelationKey[E any](mapper Mapper[E], entity *E, column string) (relationLookupValue, error) {
	field, ok := mapper.FieldByColumn(column)
	if !ok {
		return relationLookupValue{}, &UnmappedColumnError{Column: column}
	}

	value, err := mapper.entityValue(entity)
	if err != nil {
		return relationLookupValue{}, err
	}
	fieldValue, err := fieldValueForRead(value, field)
	if err != nil {
		return relationLookupValue{}, err
	}
	boundValue, err := boundFieldValue(field, fieldValue)
	if err != nil {
		return relationLookupValue{}, err
	}
	return normalizeRelationLookupValue(boundValue)
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

func queryRelationTargets[E any](ctx context.Context, session Session, rt *relationRuntime, schema SchemaSource[E], mapper Mapper[E], targetColumn ColumnMeta, keys []any) ([]E, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	chunks := chunkRelationKeys(keys, relationChunkSize(session))
	logRuntimeNode(session,
		"relation.targets.query.start",
		"table", schema.tableRef().TableName(),
		"target_column", targetColumn.Name,
		"keys", len(keys),
		"chunks", len(chunks),
	)
	items := collectionx.NewListWithCapacity[E](len(keys))
	for index, chunk := range chunks {
		logRuntimeNode(session, "relation.targets.query.chunk", "index", index, "size", len(chunk))
		bound, err := buildRelationTargetsBoundQuery(session, rt, schema, targetColumn, chunk)
		if err != nil {
			logRuntimeNode(session, "relation.targets.query.error", "stage", "build_bound", "error", err)
			return nil, err
		}
		rows, err := QueryAllBound[E](ctx, session, bound, mapper)
		if err != nil {
			logRuntimeNode(session, "relation.targets.query.error", "stage", "query_rows", "index", index, "error", err)
			return nil, err
		}
		items.Merge(rows)
	}
	logRuntimeNode(session, "relation.targets.query.done", "table", schema.tableRef().TableName(), "items", items.Len())
	return items.Values(), nil
}

func buildRelationTargetsBoundQuery(session Session, rt *relationRuntime, schema relationSchemaSource, targetColumn ColumnMeta, keys []any) (BoundQuery, error) {
	def := schema.schemaRef()
	dialectName := session.Dialect().Name()
	tableName := schema.tableRef().Name()
	selectSig := strings.Join(lo.Map(def.columns, func(c ColumnMeta, _ int) string { return c.Name }), ",")
	cacheKey := fmt.Sprintf("rel:%s:%s:%s:%s:%d", dialectName, tableName, selectSig, targetColumn.Name, len(keys))
	cachedSQL, ok, err := relationCachedQuery(rt, cacheKey)
	if err != nil {
		return BoundQuery{}, err
	}
	if ok {
		logRuntimeNode(session, "relation.targets.bound.cache_hit", "table", tableName, "target_column", targetColumn.Name, "keys", len(keys))
		return BoundQuery{SQL: cachedSQL, Args: slices.Clone(keys)}, nil
	}
	logRuntimeNode(session, "relation.targets.bound.cache_miss", "table", tableName, "target_column", targetColumn.Name, "keys", len(keys))
	query := Select(allSelectItems(def)...).
		From(schema).
		Where(metadataComparisonPredicate{
			left:  targetColumn,
			op:    OpIn,
			right: keys,
		}).
		OrderBy(relationTargetOrders(schema, targetColumn)...)
	bound, err := Build(session, query)
	if err != nil {
		logRuntimeNode(session, "relation.targets.bound.error", "table", tableName, "error", err)
		return BoundQuery{}, err
	}
	rt.queryCache.Set(cacheKey, bound.SQL)
	return bound, nil
}

func allSelectItems(def schemaDefinition) []SelectItem {
	return lo.Map(def.columns, func(column ColumnMeta, _ int) SelectItem {
		return schemaSelectItem{meta: column}
	})
}

func indexRelationTargets[E any](targets []E, mapper Mapper[E], column, relationName string, enforceUnique bool) (map[any]E, error) {
	indexed := make(map[any]E, len(targets))
	counts := make(map[any]int, len(targets))
	for index := range targets {
		key, err := presentEntityRelationKey(mapper, &targets[index], column)
		if err != nil {
			return nil, err
		}
		if !key.ok {
			continue
		}
		counts[key.value]++
		if enforceUnique && counts[key.value] > 1 {
			return nil, &RelationCardinalityError{Relation: relationName, Key: key.value, Count: counts[key.value]}
		}
		indexed[key.value] = targets[index]
	}
	return indexed, nil
}

func groupRelationTargets[E any](_ *relationRuntime, targets []E, mapper Mapper[E], column string) (collectionx.MultiMap[any, E], error) {
	grouped := collectionx.NewMultiMapWithCapacity[any, E](len(targets))
	for index := range targets {
		key, err := presentEntityRelationKey(mapper, &targets[index], column)
		if err != nil {
			return nil, err
		}
		if !key.ok {
			continue
		}
		grouped.Put(key.value, targets[index])
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
