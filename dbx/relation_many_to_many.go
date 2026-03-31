package dbx

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"

	"github.com/DaiYuANg/arcgo/collectionx"
)

func queryManyToManyPairs(ctx context.Context, session Session, rt *relationRuntime, meta RelationMeta, sourceKeys []any, sourceType, targetType reflect.Type) ([]relationKeyPair, error) {
	if meta.ThroughTable == "" {
		return nil, fmt.Errorf("dbx: many-to-many relation %s requires join table", meta.Name)
	}
	if meta.ThroughLocalColumn == "" || meta.ThroughTargetColumn == "" {
		return nil, fmt.Errorf("dbx: many-to-many relation %s requires join_local and join_target", meta.Name)
	}

	pairs := collectionx.NewListWithCapacity[relationKeyPair](len(sourceKeys))
	chunks := chunkRelationKeys(sourceKeys, relationChunkSize(session))
	logRuntimeNode(session, "relation.m2m.pairs.start", "relation", meta.Name, "keys", len(sourceKeys), "chunks", len(chunks))
	for index, chunk := range chunks {
		logRuntimeNode(session, "relation.m2m.pairs.chunk", "relation", meta.Name, "index", index, "size", len(chunk))
		bound, err := buildManyToManyPairsBoundQuery(session, rt, meta, chunk)
		if err != nil {
			logRuntimeNode(session, "relation.m2m.pairs.error", "stage", "build_bound", "relation", meta.Name, "error", err)
			return nil, err
		}
		scanned, err := queryManyToManyPairChunk(ctx, session, bound, sourceType, targetType)
		if err != nil {
			logRuntimeNode(session, "relation.m2m.pairs.error", "stage", "query_rows", "relation", meta.Name, "index", index, "error", err)
			return nil, err
		}
		pairs.Add(scanned...)
	}
	logRuntimeNode(session, "relation.m2m.pairs.done", "relation", meta.Name, "pairs", pairs.Len())
	return pairs.Values(), nil
}

func queryManyToManyPairChunk(ctx context.Context, session Session, bound BoundQuery, sourceType, targetType reflect.Type) (_ []relationKeyPair, err error) {
	rows, err := session.QueryBoundContext(ctx, bound)
	if err != nil {
		return nil, wrapDBError("query many-to-many rows", err)
	}
	defer func() {
		err = errors.Join(err, closeRows(rows))
	}()

	return scanRelationPairs(rows, sourceType, targetType)
}

func buildManyToManyPairsBoundQuery(session Session, rt *relationRuntime, meta RelationMeta, sourceKeys []any) (BoundQuery, error) {
	dialectName := session.Dialect().Name()
	cacheKey := fmt.Sprintf("m2m:%s:%s:%s:%s:%d", dialectName, meta.ThroughTable, meta.ThroughLocalColumn, meta.ThroughTargetColumn, len(sourceKeys))
	cachedSQL, ok, err := relationCachedQuery(rt, cacheKey)
	if err != nil {
		return BoundQuery{}, err
	}
	if ok {
		logRuntimeNode(session, "relation.m2m.bound.cache_hit", "relation", meta.Name, "through", meta.ThroughTable, "keys", len(sourceKeys))
		return BoundQuery{SQL: cachedSQL, Args: slices.Clone(sourceKeys)}, nil
	}
	logRuntimeNode(session, "relation.m2m.bound.cache_miss", "relation", meta.Name, "through", meta.ThroughTable, "keys", len(sourceKeys))

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
		logRuntimeNode(session, "relation.m2m.bound.error", "relation", meta.Name, "error", err)
		return BoundQuery{}, err
	}
	rt.queryCache.Set(cacheKey, bound.SQL)
	return bound, nil
}

func uniqueRelationKeysFromPairs(rt *relationRuntime, pairs []relationKeyPair, useSource bool) []any {
	keys := collectionx.NewListWithCapacity[any](len(pairs))
	seen, err := relationSeenSet(rt)
	if err != nil {
		return nil
	}
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
	counts, err := relationCountsMap(rt)
	if err != nil {
		return make(map[any][]E)
	}
	defer func() {
		counts.Clear()
		rt.countsMapPool.Put(counts)
	}()
	for _, pair := range pairs {
		if _, ok := indexed[pair.target]; ok {
			value, _ := counts.Get(pair.source)
			counts.Set(pair.source, value+1)
		}
	}
	grouped := groupedValuesFromCounts[E](counts)
	for _, pair := range pairs {
		target, ok := indexed[pair.target]
		if !ok {
			continue
		}
		grouped[pair.source] = append(grouped[pair.source], target)
	}
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
