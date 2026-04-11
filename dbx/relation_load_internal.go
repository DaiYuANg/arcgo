package dbx

import (
	"context"
	"fmt"
	"reflect"

	"github.com/DaiYuANg/arcgo/collectionx"
)

type relationLookupValue struct {
	present bool
	key     any
}

type relationKeyPair struct {
	source any
	target any
}

func collectSourceRelationKeys[E any](rt *relationRuntime, entities []E, mapper Mapper[E], schema schemaDefinition, meta RelationMeta) (collectionx.List[any], []relationLookupValue, error) {
	localColumn, err := relationSourceColumn(schemaAdapter[E]{def: schema}, meta)
	if err != nil {
		return nil, nil, err
	}

	lookup := collectionx.NewListWithCapacity[relationLookupValue](len(entities))
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
		lookup.AddAt(index, key)
		if !key.present {
			continue
		}
		if _, ok := seen.Get(key.key); ok {
			continue
		}
		seen.Set(key.key, struct{}{})
		keys.Add(key.key)
	}
	return keys, lookup.Values(), nil
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
		if primaryKey == nil || primaryKey.Columns.Len() != 1 {
			return ColumnMeta{}, fmt.Errorf("dbx: relation %s requires target column or single-column primary key", meta.Name)
		}
		name, _ = primaryKey.Columns.GetFirst()
	}

	column, ok := sourceColumnByName(schema.schemaRef(), name)
	if !ok {
		return ColumnMeta{}, fmt.Errorf("dbx: relation %s target column %s not found", meta.Name, name)
	}
	return column, nil
}

func queryRelationTargets[E any](ctx context.Context, session Session, rt *relationRuntime, schema SchemaSource[E], mapper Mapper[E], targetColumn ColumnMeta, keys collectionx.List[any]) (collectionx.List[E], error) {
	if keys.Len() == 0 {
		return collectionx.NewList[E](), nil
	}
	chunks := chunkRelationKeys(keys, relationChunkSize(session))
	logRuntimeNode(session,
		"relation.targets.query.start",
		"table", schema.tableRef().TableName(),
		"target_column", targetColumn.Name,
		"keys", keys.Len(),
		"chunks", chunks.Len(),
	)
	items := collectionx.NewListWithCapacity[E](keys.Len())
	var resultErr error
	chunks.Range(func(index int, chunk collectionx.List[any]) bool {
		logRuntimeNode(session, "relation.targets.query.chunk", "index", index, "size", chunk.Len())
		bound, err := buildRelationTargetsBoundQuery(session, rt, schema, targetColumn, chunk)
		if err != nil {
			logRuntimeNode(session, "relation.targets.query.error", "stage", "build_bound", "error", err)
			resultErr = err
			return false
		}
		rows, err := QueryAllBoundList[E](ctx, session, bound, mapper)
		if err != nil {
			logRuntimeNode(session, "relation.targets.query.error", "stage", "query_rows", "index", index, "error", err)
			resultErr = err
			return false
		}
		items.Merge(rows)
		return true
	})
	if resultErr != nil {
		return nil, resultErr
	}
	logRuntimeNode(session, "relation.targets.query.done", "table", schema.tableRef().TableName(), "items", items.Len())
	return items, nil
}

func buildRelationTargetsBoundQuery(session Session, rt *relationRuntime, schema relationSchemaSource, targetColumn ColumnMeta, keys collectionx.List[any]) (BoundQuery, error) {
	def := schema.schemaRef()
	dialectName := session.Dialect().Name()
	tableName := schema.tableRef().Name()
	selectSigParts := collectionx.NewListWithCapacity[string](def.columns.Len())
	def.columns.Range(func(_ int, column ColumnMeta) bool {
		selectSigParts.Add(column.Name)
		return true
	})
	selectSig := selectSigParts.Join(",")
	cacheKey := fmt.Sprintf("rel:%s:%s:%s:%s:%d", dialectName, tableName, selectSig, targetColumn.Name, keys.Len())
	cachedSQL, ok, err := relationCachedQuery(rt, cacheKey)
	if err != nil {
		return BoundQuery{}, err
	}
	if ok {
		logRuntimeNode(session, "relation.targets.bound.cache_hit", "table", tableName, "target_column", targetColumn.Name, "keys", keys.Len())
		return BoundQuery{SQL: cachedSQL, Args: keys.Clone()}, nil
	}
	logRuntimeNode(session, "relation.targets.bound.cache_miss", "table", tableName, "target_column", targetColumn.Name, "keys", keys.Len())
	query := SelectList(allSelectItems(def)).
		From(schema).
		Where(metadataComparisonPredicate{
			left:  targetColumn,
			op:    OpIn,
			right: keys.Values(),
		}).
		OrderByList(relationTargetOrders(schema, targetColumn))
	bound, err := Build(session, query)
	if err != nil {
		logRuntimeNode(session, "relation.targets.bound.error", "table", tableName, "error", err)
		return BoundQuery{}, err
	}
	rt.queryCache.Set(cacheKey, bound.SQL)
	return bound, nil
}

func allSelectItems(def schemaDefinition) collectionx.List[SelectItem] {
	items := collectionx.NewListWithCapacity[SelectItem](def.columns.Len())
	def.columns.Range(func(_ int, column ColumnMeta) bool {
		items.Add(schemaSelectItem{meta: column})
		return true
	})
	return items
}

func indexRelationTargets[E any](targets collectionx.List[E], mapper Mapper[E], column, relationName string, enforceUnique bool) (map[any]E, error) {
	indexed := make(map[any]E, targets.Len())
	counts := make(map[any]int, targets.Len())
	var resultErr error
	targets.Range(func(_ int, target E) bool {
		key, err := presentEntityRelationKey(mapper, &target, column)
		if err != nil {
			resultErr = err
			return false
		}
		if !key.ok {
			return true
		}
		counts[key.value]++
		if enforceUnique && counts[key.value] > 1 {
			resultErr = &RelationCardinalityError{Relation: relationName, Key: key.value, Count: counts[key.value]}
			return false
		}
		indexed[key.value] = target
		return true
	})
	if resultErr != nil {
		return nil, resultErr
	}
	return indexed, nil
}

func groupRelationTargets[E any](_ *relationRuntime, targets collectionx.List[E], mapper Mapper[E], column string) (collectionx.MultiMap[any, E], error) {
	grouped := collectionx.NewMultiMapWithCapacity[any, E](targets.Len())
	var resultErr error
	targets.Range(func(_ int, target E) bool {
		key, err := presentEntityRelationKey(mapper, &target, column)
		if err != nil {
			resultErr = err
			return false
		}
		if !key.ok {
			return true
		}
		grouped.Put(key.value, target)
		return true
	})
	if resultErr != nil {
		return nil, resultErr
	}
	return grouped, nil
}

func relationKeyTypeForMeta(def schemaDefinition, column string) reflect.Type {
	if column == "" {
		primaryKey := derivePrimaryKey(def)
		if primaryKey == nil || primaryKey.Columns.Len() != 1 {
			return nil
		}
		column, _ = primaryKey.Columns.GetFirst()
	}
	columnMeta, ok := sourceColumnByName(def, column)
	if !ok {
		return nil
	}
	return columnMeta.GoType
}
