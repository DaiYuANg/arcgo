package dbx

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
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
	seen := rt.seenSetPool.Get().(collectionx.Map[any, struct{}])
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
	items := collectionx.NewListWithCapacity[E](len(keys))
	for _, chunk := range chunkRelationKeys(keys, relationChunkSize(session)) {
		bound, err := buildRelationTargetsBoundQuery(session, rt, schema, targetColumn, chunk)
		if err != nil {
			return nil, err
		}
		rows, err := QueryAllBound[E](ctx, session, bound, mapper)
		if err != nil {
			return nil, err
		}
		items.Add(rows...)
	}
	return items.Values(), nil
}

func buildRelationTargetsBoundQuery(session Session, rt *relationRuntime, schema relationSchemaSource, targetColumn ColumnMeta, keys []any) (BoundQuery, error) {
	def := schema.schemaRef()
	dialectName := session.Dialect().Name()
	tableName := schema.tableRef().Name()
	selectSig := strings.Join(lo.Map(def.columns, func(c ColumnMeta, _ int) string { return c.Name }), ",")
	cacheKey := fmt.Sprintf("rel:%s:%s:%s:%s:%d", dialectName, tableName, selectSig, targetColumn.Name, len(keys))
	if cachedSQL, ok, _ := rt.queryCache.Get(cacheKey); ok {
		args := make([]any, len(keys))
		copy(args, keys)
		return BoundQuery{SQL: cachedSQL, Args: args}, nil
	}
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

func indexRelationTargets[E any](targets []E, mapper Mapper[E], column string, relationName string, enforceUnique bool) (map[any]E, error) {
	indexed := make(map[any]E, len(targets))
	counts := make(map[any]int, len(targets))
	for index := range targets {
		key, err := entityRelationKey(mapper, &targets[index], column)
		if err != nil {
			return nil, err
		}
		if !key.present {
			continue
		}
		counts[key.key]++
		if enforceUnique && counts[key.key] > 1 {
			return nil, &RelationCardinalityError{Relation: relationName, Key: key.key, Count: counts[key.key]}
		}
		indexed[key.key] = targets[index]
	}
	return indexed, nil
}

func groupRelationTargets[E any](rt *relationRuntime, targets []E, mapper Mapper[E], column string) (map[any][]E, error) {
	counts := rt.countsMapPool.Get().(collectionx.Map[any, int])
	defer func() {
		counts.Clear()
		rt.countsMapPool.Put(counts)
	}()
	for index := range targets {
		key, err := entityRelationKey(mapper, &targets[index], column)
		if err != nil {
			return nil, err
		}
		if !key.present {
			continue
		}
		v, _ := counts.Get(key.key)
		counts.Set(key.key, v+1)
	}
	grouped := make(map[any][]E, counts.Len())
	counts.Range(func(k any, cap int) bool {
		grouped[k] = make([]E, 0, cap)
		return true
	})
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

func queryManyToManyPairs(ctx context.Context, session Session, rt *relationRuntime, meta RelationMeta, sourceKeys []any, sourceType, targetType reflect.Type) ([]relationKeyPair, error) {
	if meta.ThroughTable == "" {
		return nil, fmt.Errorf("dbx: many-to-many relation %s requires join table", meta.Name)
	}
	if meta.ThroughLocalColumn == "" || meta.ThroughTargetColumn == "" {
		return nil, fmt.Errorf("dbx: many-to-many relation %s requires join_local and join_target", meta.Name)
	}

	pairs := collectionx.NewListWithCapacity[relationKeyPair](len(sourceKeys))
	for _, chunk := range chunkRelationKeys(sourceKeys, relationChunkSize(session)) {
		bound, err := buildManyToManyPairsBoundQuery(session, rt, meta, chunk)
		if err != nil {
			return nil, err
		}
		rows, err := session.QueryBoundContext(ctx, bound)
		if err != nil {
			return nil, err
		}
		scanned, scanErr := scanRelationPairs(rows, sourceType, targetType)
		_ = rows.Close()
		if scanErr != nil {
			return nil, scanErr
		}
		pairs.Add(scanned...)
	}
	return pairs.Values(), nil
}

func buildManyToManyPairsBoundQuery(session Session, rt *relationRuntime, meta RelationMeta, sourceKeys []any) (BoundQuery, error) {
	dialectName := session.Dialect().Name()
	cacheKey := fmt.Sprintf("m2m:%s:%s:%s:%s:%d", dialectName, meta.ThroughTable, meta.ThroughLocalColumn, meta.ThroughTargetColumn, len(sourceKeys))
	if cachedSQL, ok, _ := rt.queryCache.Get(cacheKey); ok {
		args := make([]any, len(sourceKeys))
		copy(args, sourceKeys)
		return BoundQuery{SQL: cachedSQL, Args: args}, nil
	}

	through := Table{def: tableDefinition{name: meta.ThroughTable}}
	localColumn := ColumnMeta{Name: meta.ThroughLocalColumn, Table: through.Name(), GoType: nil}
	targetColumn := ColumnMeta{Name: meta.ThroughTargetColumn, Table: through.Name(), GoType: nil}
	query := Select(
		schemaSelectItem{meta: localColumn},
		schemaSelectItem{meta: targetColumn},
	).From(through).Where(metadataComparisonPredicate{
		left:  localColumn,
		op:    OpIn,
		right: sourceKeys,
	}).OrderBy(
		NamedColumn[any](through, meta.ThroughLocalColumn).Asc(),
		NamedColumn[any](through, meta.ThroughTargetColumn).Asc(),
	)

	bound, err := Build(session, query)
	if err != nil {
		return BoundQuery{}, err
	}
	rt.queryCache.Set(cacheKey, bound.SQL)
	return bound, nil
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

func uniqueRelationKeysFromPairs(rt *relationRuntime, pairs []relationKeyPair, useSource bool) []any {
	keys := collectionx.NewListWithCapacity[any](len(pairs))
	seen := rt.seenSetPool.Get().(collectionx.Map[any, struct{}])
	defer func() {
		seen.Clear()
		rt.seenSetPool.Put(seen)
	}()
	for _, pair := range pairs {
		key := pair.target
		if useSource {
			key = pair.source
		}
		if _, ok := seen.Get(key); ok {
			continue
		}
		seen.Set(key, struct{}{})
		keys.Add(key)
	}
	return keys.Values()
}

func groupManyToManyTargets[E any](rt *relationRuntime, pairs []relationKeyPair, indexed map[any]E) map[any][]E {
	counts := rt.countsMapPool.Get().(collectionx.Map[any, int])
	defer func() {
		counts.Clear()
		rt.countsMapPool.Put(counts)
	}()
	for _, pair := range pairs {
		if _, ok := indexed[pair.target]; ok {
			v, _ := counts.Get(pair.source)
			counts.Set(pair.source, v+1)
		}
	}
	grouped := make(map[any][]E, counts.Len())
	counts.Range(func(k any, cap int) bool {
		grouped[k] = make([]E, 0, cap)
		return true
	})
	for _, pair := range pairs {
		target, ok := indexed[pair.target]
		if !ok {
			continue
		}
		grouped[pair.source] = append(grouped[pair.source], target)
	}
	return grouped
}

func relationChunkSize(session Session) int {
	if session == nil || session.Dialect() == nil {
		return 256
	}
	switch strings.ToLower(strings.TrimSpace(session.Dialect().Name())) {
	case "sqlite":
		return 900
	case "postgres", "mysql":
		return 4096
	default:
		return 512
	}
}

func chunkRelationKeys(keys []any, chunkSize int) [][]any {
	if len(keys) == 0 {
		return nil
	}
	if chunkSize <= 0 || len(keys) <= chunkSize {
		return [][]any{keys}
	}
	chunks := make([][]any, 0, (len(keys)+chunkSize-1)/chunkSize)
	for start := 0; start < len(keys); start += chunkSize {
		end := start + chunkSize
		if end > len(keys) {
			end = len(keys)
		}
		chunks = append(chunks, keys[start:end])
	}
	return chunks
}

func relationTargetOrders(schema relationSchemaSource, targetColumn ColumnMeta) []Order {
	orders := []Order{NamedColumn[any](schema, targetColumn.Name).Asc()}
	if primaryKey := derivePrimaryKey(schema.schemaRef()); primaryKey != nil && len(primaryKey.Columns) == 1 && primaryKey.Columns[0] != targetColumn.Name {
		orders = append(orders, NamedColumn[any](schema, primaryKey.Columns[0]).Asc())
	}
	return orders
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
