package repository

import (
	"context"
	"errors"

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

func (b repositoryBase[T]) scanAllKeys(ctx context.Context, kv kvx.KV) ([]string, error) {
	seen := set.NewSet[string]()
	cursor := uint64(0)

	for {
		keys, next, err := kv.Scan(ctx, b.keyFromID("*"), cursor, scanBatchSize)
		if err != nil {
			return nil, wrapRepositoryError(err, "scan repository keys")
		}

		seen.Add(keys...)
		if next == 0 {
			return seen.Values(), nil
		}
		cursor = next
	}
}

func intersectStringSlices(groups ...[]string) []string {
	if len(groups) == 0 {
		return nil
	}

	intersection := set.NewSet[string](groups[0]...)
	for _, group := range groups[1:] {
		if intersection.IsEmpty() {
			return nil
		}
		intersection = intersection.Intersect(set.NewSet[string](group...))
	}

	return intersection.Values()
}

func collectPresentMap[K comparable, T any](items []K, load func(K) (*T, error)) (map[K]*T, error) {
	results := make(map[K]*T, len(items))
	for _, item := range items {
		entityOpt, err := loadPresent(load(item))
		if err != nil {
			return nil, err
		}
		entityOpt.ForEach(func(entity *T) {
			results[item] = entity
		})
	}
	return results, nil
}

func collectPresentSlice[K any, T any](items []K, load func(K) (*T, error)) ([]*T, error) {
	results := make([]*T, 0, len(items))
	for _, item := range items {
		entityOpt, err := loadPresent(load(item))
		if err != nil {
			return nil, err
		}
		entityOpt.ForEach(func(entity *T) {
			results = append(results, entity)
		})
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
	results := make(map[string]bool, len(ids))
	for i, id := range ids {
		results[id] = existsMap[keys[i]]
	}
	return results
}

func runAll[T any](items []T, fn func(T) error) error {
	for _, item := range items {
		if err := fn(item); err != nil {
			return err
		}
	}
	return nil
}
