package dbx

import (
	"database/sql"
	"errors"
	"reflect"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
)

func relationSeenSet(rt *relationRuntime) (collectionx.Map[any, struct{}], error) {
	seen, ok := rt.seenSetPool.Get().(collectionx.Map[any, struct{}])
	if !ok {
		return collectionx.NewMap[any, struct{}](), errors.New("dbx: invalid relation seen-set pool value")
	}
	return seen, nil
}

func relationCachedQuery(rt *relationRuntime, cacheKey string) (string, bool, error) {
	value, ok, err := rt.queryCache.Get(cacheKey)
	return value, ok, wrapDBError("read relation query cache", err)
}

func scanRelationPairs(rows *sql.Rows, sourceType, targetType reflect.Type) (collectionx.List[relationKeyPair], error) {
	pairs := collectionx.NewList[relationKeyPair]()
	for rows.Next() {
		pair, ok, err := scanRelationPairRow(rows, sourceType, targetType)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		pairs.Add(pair)
	}
	if err := rowsIterError(rows); err != nil {
		return nil, err
	}
	return pairs, nil
}

func scanRelationPairRow(rows *sql.Rows, sourceType, targetType reflect.Type) (relationKeyPair, bool, error) {
	sourceDest, sourceValue := relationScanDestination(sourceType)
	targetDest, targetValue := relationScanDestination(targetType)
	if err := rows.Scan(sourceDest, targetDest); err != nil {
		return relationKeyPair{}, false, wrapDBError("scan relation pair row", err)
	}
	sourceKey, targetKey, err := normalizeRelationPair(sourceValue(), targetValue())
	if err != nil {
		return relationKeyPair{}, false, err
	}
	if !sourceKey.present || !targetKey.present {
		return relationKeyPair{}, false, nil
	}
	return relationKeyPair{source: sourceKey.key, target: targetKey.key}, true, nil
}

func normalizeRelationPair(source, target any) (relationLookupValue, relationLookupValue, error) {
	sourceKey, err := normalizeRelationLookupValue(source)
	if err != nil {
		return relationLookupValue{}, relationLookupValue{}, err
	}
	targetKey, err := normalizeRelationLookupValue(target)
	if err != nil {
		return relationLookupValue{}, relationLookupValue{}, err
	}
	return sourceKey, targetKey, nil
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

func chunkRelationKeys(keys collectionx.List[any], chunkSize int) collectionx.List[collectionx.List[any]] {
	if keys.Len() == 0 {
		return collectionx.NewList[collectionx.List[any]]()
	}
	if chunkSize <= 0 || keys.Len() <= chunkSize {
		return collectionx.NewList(keys.Clone())
	}

	chunks := collectionx.NewListWithCapacity[collectionx.List[any]]((keys.Len() + chunkSize - 1) / chunkSize)
	current := collectionx.NewListWithCapacity[any](chunkSize)
	keys.Range(func(_ int, key any) bool {
		current.Add(key)
		if current.Len() < chunkSize {
			return true
		}
		chunks.Add(current)
		current = collectionx.NewListWithCapacity[any](chunkSize)
		return true
	})
	if current.Len() > 0 {
		chunks.Add(current)
	}
	return chunks
}

func relationTargetOrders(schema relationSchemaSource, targetColumn ColumnMeta) collectionx.List[Order] {
	orders := collectionx.NewList[Order](NamedColumn[any](schema, targetColumn.Name).Asc())
	if primaryKey := derivePrimaryKey(schema.schemaRef()); primaryKey != nil && primaryKey.Columns.Len() == 1 {
		if column, ok := primaryKey.Columns.GetFirst(); ok && column != targetColumn.Name {
			orders.Add(NamedColumn[any](schema, column).Asc())
		}
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
