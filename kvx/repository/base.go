package repository

import (
	"context"
	"errors"
	"fmt"

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

func (b repositoryBase[T]) metadata(entity *T) (*mapping.EntityMetadata, error) {
	metadata, err := b.tagParser.ParseType(entity)
	return wrapRepositoryResult(metadata, err, "parse entity metadata")
}

func (b repositoryBase[T]) metadataForType() (*mapping.EntityMetadata, error) {
	var zero T
	metadata, err := b.tagParser.ParseType(&zero)
	return wrapRepositoryResult(metadata, err, "parse repository metadata")
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
		return nil, ErrFieldNotFound
	}
	return b.indexer.GetEntityIDsByField(ctx, fieldTag.IndexNameOrDefault(), fieldValue)
}

func (b repositoryBase[T]) hydrateEntityID(entity *T, metadata *mapping.EntityMetadata, key string) error {
	return wrapRepositoryError(metadata.SetEntityID(entity, extractIDFromKey(key)), "hydrate entity ID")
}

func (b repositoryBase[T]) scanAllKeys(ctx context.Context, kv kvx.KV) (collectionx.List[string], error) {
	seen := set.NewSet[string]()
	cursor := uint64(0)

	for {
		keys, next, err := kv.Scan(ctx, b.keyFromID("*"), cursor, scanBatchSize)
		if err != nil {
			return nil, wrapRepositoryError(err, "scan repository keys")
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
		return nil, fmt.Errorf("collect present map: %w", err)
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
		return nil, fmt.Errorf("collect present list: %w", err)
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
		return nil, fmt.Errorf("load field id groups: %w", err)
	}
	return groups, nil
}

func runAll[T any](items []T, fn func(T) error) error {
	_, err := lo.ReduceErr(items, func(_ struct{}, item T, _ int) (struct{}, error) {
		return struct{}{}, fn(item)
	}, struct{}{})
	if err != nil {
		return fmt.Errorf("run all items: %w", err)
	}
	return nil
}
