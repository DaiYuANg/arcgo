package dbx

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/DaiYuANg/arcgo/collectionx"
)

func queryManyToManyPairs(ctx context.Context, session Session, rt *relationRuntime, meta RelationMeta, sourceKeys collectionx.List[any], sourceType, targetType reflect.Type) (collectionx.List[relationKeyPair], error) {
	if meta.ThroughTable == "" {
		return nil, fmt.Errorf("dbx: many-to-many relation %s requires join table", meta.Name)
	}
	if meta.ThroughLocalColumn == "" || meta.ThroughTargetColumn == "" {
		return nil, fmt.Errorf("dbx: many-to-many relation %s requires join_local and join_target", meta.Name)
	}

	pairs := collectionx.NewListWithCapacity[relationKeyPair](sourceKeys.Len())
	chunks := chunkRelationKeys(sourceKeys, relationChunkSize(session))
	logRuntimeNode(session, "relation.m2m.pairs.start", "relation", meta.Name, "keys", sourceKeys.Len(), "chunks", chunks.Len())
	var resultErr error
	chunks.Range(func(index int, chunk collectionx.List[any]) bool {
		logRuntimeNode(session, "relation.m2m.pairs.chunk", "relation", meta.Name, "index", index, "size", chunk.Len())
		bound, err := buildManyToManyPairsBoundQuery(session, rt, meta, chunk)
		if err != nil {
			logRuntimeNode(session, "relation.m2m.pairs.error", "stage", "build_bound", "relation", meta.Name, "error", err)
			resultErr = err
			return false
		}
		scanned, err := queryManyToManyPairChunk(ctx, session, bound, sourceType, targetType)
		if err != nil {
			logRuntimeNode(session, "relation.m2m.pairs.error", "stage", "query_rows", "relation", meta.Name, "index", index, "error", err)
			resultErr = err
			return false
		}
		pairs.Merge(scanned)
		return true
	})
	if resultErr != nil {
		return nil, resultErr
	}
	logRuntimeNode(session, "relation.m2m.pairs.done", "relation", meta.Name, "pairs", pairs.Len())
	return pairs, nil
}

func queryManyToManyPairChunk(ctx context.Context, session Session, bound BoundQuery, sourceType, targetType reflect.Type) (_ collectionx.List[relationKeyPair], err error) {
	rows, err := session.QueryBoundContext(ctx, bound)
	if err != nil {
		return nil, wrapDBError("query many-to-many rows", err)
	}
	defer func() {
		err = errors.Join(err, closeRows(rows))
	}()

	return scanRelationPairs(rows, sourceType, targetType)
}

func buildManyToManyPairsBoundQuery(session Session, rt *relationRuntime, meta RelationMeta, sourceKeys collectionx.List[any]) (BoundQuery, error) {
	dialectName := session.Dialect().Name()
	cacheKey := fmt.Sprintf("m2m:%s:%s:%s:%s:%d", dialectName, meta.ThroughTable, meta.ThroughLocalColumn, meta.ThroughTargetColumn, sourceKeys.Len())
	cachedSQL, ok, err := relationCachedQuery(rt, cacheKey)
	if err != nil {
		return BoundQuery{}, err
	}
	if ok {
		logRuntimeNode(session, "relation.m2m.bound.cache_hit", "relation", meta.Name, "through", meta.ThroughTable, "keys", sourceKeys.Len())
		return BoundQuery{SQL: cachedSQL, Args: sourceKeys.Clone()}, nil
	}
	logRuntimeNode(session, "relation.m2m.bound.cache_miss", "relation", meta.Name, "through", meta.ThroughTable, "keys", sourceKeys.Len())

	through := Table{def: tableDefinition{name: meta.ThroughTable}}
	localColumn := ColumnMeta{Name: meta.ThroughLocalColumn, Table: through.Name(), GoType: nil}
	targetColumn := ColumnMeta{Name: meta.ThroughTargetColumn, Table: through.Name(), GoType: nil}
	query := Select(
		schemaSelectItem{meta: localColumn},
		schemaSelectItem{meta: targetColumn},
	).From(through).Where(metadataComparisonPredicate{
		left:  localColumn,
		op:    OpIn,
		right: sourceKeys.Values(),
	}).OrderBy(
		NamedColumn[any](through, meta.ThroughLocalColumn).Asc(),
		NamedColumn[any](through, meta.ThroughTargetColumn).Asc(),
	)

	bound, err := Build(session, query)
	if err != nil {
		logRuntimeNode(session, "relation.m2m.bound.error", "relation", meta.Name, "error", err)
		return BoundQuery{}, err
	}
	rt.queryCache.Set(cacheKey, bound.SQL)
	return bound, nil
}

func uniqueRelationKeysFromPairs(rt *relationRuntime, pairs collectionx.List[relationKeyPair], useSource bool) collectionx.List[any] {
	keys := collectionx.NewListWithCapacity[any](pairs.Len())
	seen, err := relationSeenSet(rt)
	if err != nil {
		return nil
	}
	defer func() {
		seen.Clear()
		rt.seenSetPool.Put(seen)
	}()
	pairs.Range(func(_ int, pair relationKeyPair) bool {
		key := pair.target
		if useSource {
			key = pair.source
		}
		if _, ok := seen.Get(key); ok {
			return true
		}
		seen.Set(key, struct{}{})
		keys.Add(key)
		return true
	})
	return keys
}

func groupManyToManyTargets[E any](_ *relationRuntime, pairs collectionx.List[relationKeyPair], indexed map[any]E) collectionx.MultiMap[any, E] {
	grouped := collectionx.NewMultiMapWithCapacity[any, E](pairs.Len())
	pairs.Range(func(_ int, pair relationKeyPair) bool {
		target, ok := indexed[pair.target]
		if !ok {
			return true
		}
		grouped.Put(pair.source, target)
		return true
	})
	return grouped
}

type presentRelationKey struct {
	value any
	ok    bool
}

func presentEntityRelationKey[E any](mapper Mapper[E], entity *E, column string) (presentRelationKey, error) {
	key, err := entityRelationKey(mapper, entity, column)
	if err != nil {
		return presentRelationKey{}, err
	}
	if !key.present {
		return presentRelationKey{}, nil
	}
	return presentRelationKey{value: key.key, ok: true}, nil
}
