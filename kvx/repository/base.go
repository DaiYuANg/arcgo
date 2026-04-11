package repository

import (
	"context"
	"errors"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/collectionx/set"
	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/DaiYuANg/arcgo/kvx/mapping"
	"github.com/samber/lo"
	"github.com/samber/mo"
)

const scanBatchSize int64 = 256

type repositoryBase[T any] struct {
	keyBuilder *mapping.KeyBuilder
	tagParser  *mapping.TagParser
	indexer    *Indexer[T]
}

type repositoryDeleteState struct {
	key           string
	removeEntries []string
}

func prepareRepositoryDelete[T any](
	ctx context.Context,
	base repositoryBase[T],
	id string,
	findByKey func(context.Context, string) (*T, error),
	logDebug func(string, ...any),
	startMessage string,
	loadOp string,
	loadMessage string,
	collectOp string,
	collectMessage string,
) (repositoryDeleteState, bool, error) {
	key := base.keyFromID(id)
	logDebug(startMessage, "key", key)

	entity, err := findByKey(ctx, key)
	if errors.Is(err, ErrNotFound) {
		return repositoryDeleteState{key: key}, false, nil
	}
	if err != nil {
		return repositoryDeleteState{key: key}, false, wrapRepositoryError(err, loadMessage, "op", loadOp, "id", id, "key", key)
	}

	metadata, err := base.metadata(entity)
	if err != nil {
		return repositoryDeleteState{key: key}, false, err
	}
	removeEntries, err := base.indexer.EntityIndexEntries(entity, metadata, key)
	if err != nil {
		return repositoryDeleteState{key: key}, false, wrapRepositoryError(err, collectMessage, "op", collectOp, "id", id, "key", key)
	}

	return repositoryDeleteState{key: key, removeEntries: removeEntries}, true, nil
}

func (b repositoryBase[T]) metadata(entity *T) (*mapping.EntityMetadata, error) {
	metadata, err := b.tagParser.ParseType(entity)
	return wrapRepositoryResult(metadata, err, "parse entity metadata", "op", "parse_entity_metadata")
}

func (b repositoryBase[T]) metadataForType() (*mapping.EntityMetadata, error) {
	var zero T
	metadata, err := b.tagParser.ParseType(&zero)
	return wrapRepositoryResult(metadata, err, "parse repository metadata", "op", "parse_repository_metadata")
}

func (b repositoryBase[T]) keyFromID(id string) string {
	return b.keyBuilder.BuildWithID(id)
}

func (b repositoryBase[T]) keysFromIDs(ids []string) []string {
	return lo.Map(ids, func(id string, _ int) string {
		return b.keyFromID(id)
	})
}

func (b repositoryBase[T]) idsByField(ctx context.Context, fieldName, fieldValue string) ([]string, error) {
	metadata, err := b.metadataForType()
	if err != nil {
		return nil, err
	}
	_, fieldTag, ok := metadata.ResolveField(fieldName)
	if !ok {
		return nil, wrapRepositoryError(ErrFieldNotFound, "resolve field metadata", "op", "resolve_field_metadata", "field_name", fieldName, "field_value", fieldValue)
	}
	return b.indexer.GetEntityIDsByField(ctx, fieldTag.IndexNameOrDefault(), fieldValue)
}

func (b repositoryBase[T]) hydrateEntityID(entity *T, metadata *mapping.EntityMetadata, key string) error {
	return wrapRepositoryError(metadata.SetEntityID(entity, extractIDFromKey(key)), "hydrate entity ID", "op", "hydrate_entity_id", "key", key, "entity_id", extractIDFromKey(key))
}

func (b repositoryBase[T]) scanAllKeys(ctx context.Context, kv kvx.KV) (collectionx.List[string], error) {
	seen := set.NewSet[string]()
	cursor := uint64(0)

	for {
		keys, next, err := kv.Scan(ctx, b.keyFromID("*"), cursor, scanBatchSize)
		if err != nil {
			return nil, wrapRepositoryError(err, "scan repository keys", "op", "scan_repository_keys", "pattern", b.keyFromID("*"), "cursor", cursor, "batch_size", scanBatchSize)
		}

		seen.Add(keys.Values()...)
		if next == 0 {
			return collectionx.NewListWithCapacity(seen.Len(), seen.Values()...), nil
		}
		cursor = next
	}
}

func intersectStringSlices(groups ...[]string) []string {
	if len(groups) == 0 {
		return nil
	}

	intersection := lo.Reduce(groups[1:], func(result *set.Set[string], group []string, _ int) *set.Set[string] {
		if result.IsEmpty() {
			return set.NewSet[string]()
		}
		return result.Intersect(set.NewSet[string](group...))
	}, set.NewSet[string](groups[0]...))

	return intersection.Values()
}

func collectPresentMap[K comparable, T any](items []K, load func(K) (*T, error)) (map[K]*T, error) {
	results, err := lo.ReduceErr(items, func(results map[K]*T, item K, _ int) (map[K]*T, error) {
		entityOpt, err := loadPresent(load(item))
		if err != nil {
			return nil, err
		}
		entityOpt.ForEach(func(entity *T) {
			results[item] = entity
		})
		return results, nil
	}, make(map[K]*T, len(items)))
	if err != nil {
		return nil, wrapRepositoryError(err, "collect present map", "op", "collect_present_map", "item_count", len(items))
	}
	return results, nil
}

func collectPresentList[K any, T any](items []K, load func(K) (*T, error)) (collectionx.List[*T], error) {
	results, err := lo.ReduceErr(items, func(results collectionx.List[*T], item K, _ int) (collectionx.List[*T], error) {
		entityOpt, err := loadPresent(load(item))
		if err != nil {
			return nil, err
		}
		entityOpt.ForEach(func(entity *T) {
			results.Add(entity)
		})
		return results, nil
	}, collectionx.NewListWithCapacity[*T](len(items)))
	if err != nil {
		return nil, wrapRepositoryError(err, "collect present list", "op", "collect_present_list", "item_count", len(items))
	}
	return results, nil
}

func loadPresent[T any](entity *T, err error) (mo.Option[*T], error) {
	if err == nil {
		return mo.Some(entity), nil
	}
	if errors.Is(err, ErrNotFound) {
		return mo.None[*T](), nil
	}
	return mo.None[*T](), err
}

func mapExistsResults(ids, keys []string, existsMap map[string]bool) map[string]bool {
	return lo.Reduce(ids, func(results map[string]bool, id string, index int) map[string]bool {
		results[id] = existsMap[keys[index]]
		return results
	}, make(map[string]bool, len(ids)))
}

func loadFieldIDGroups(fields map[string]string, load func(fieldName, fieldValue string) ([]string, error)) ([][]string, error) {
	groups, err := lo.ReduceErr(lo.Entries(fields), func(result [][]string, entry lo.Entry[string, string], _ int) ([][]string, error) {
		ids, err := load(entry.Key, entry.Value)
		if err != nil {
			return nil, err
		}
		return lo.Concat(result, [][]string{ids}), nil
	}, make([][]string, 0, len(fields)))
	if err != nil {
		return nil, wrapRepositoryError(err, "load field id groups", "op", "load_field_id_groups", "field_count", len(fields))
	}
	return groups, nil
}

func runAll[T any](items []T, fn func(T) error) error {
	_, err := lo.ReduceErr(items, func(_ struct{}, item T, _ int) (struct{}, error) {
		return struct{}{}, fn(item)
	}, struct{}{})
	if err != nil {
		return wrapRepositoryError(err, "run all items", "op", "run_all_items", "item_count", len(items))
	}
	return nil
}
