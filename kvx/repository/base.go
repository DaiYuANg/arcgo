package repository

import (
	"context"

	"github.com/DaiYuANg/arcgo/collectionx/set"
	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/DaiYuANg/arcgo/kvx/mapping"
	"github.com/samber/lo"
)

const scanBatchSize int64 = 256

type repositoryBase[T any] struct {
	keyBuilder *mapping.KeyBuilder
	tagParser  *mapping.TagParser
	indexer    *Indexer[T]
}

func (b repositoryBase[T]) metadata(entity *T) (*mapping.EntityMetadata, error) {
	return b.tagParser.ParseType(entity)
}

func (b repositoryBase[T]) metadataForType() (*mapping.EntityMetadata, error) {
	var zero T
	return b.tagParser.ParseType(&zero)
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
	return metadata.SetEntityID(entity, extractIDFromKey(key))
}

func (b repositoryBase[T]) scanAllKeys(ctx context.Context, kv kvx.KV) ([]string, error) {
	seen := set.NewSet[string]()
	cursor := uint64(0)

	for {
		keys, next, err := kv.Scan(ctx, b.keyFromID("*"), cursor, scanBatchSize)
		if err != nil {
			return nil, err
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

func stringSliceIntersection(a, b []string) []string {
	return intersectStringSlices(a, b)
}
