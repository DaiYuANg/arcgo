package repository

import (
	"context"
	"time"

	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/DaiYuANg/arcgo/kvx/mapping"
	"github.com/samber/mo"
)

// JSONRepository provides repository operations for JSON-based entities.
type JSONRepository[T any] struct {
	base       repositoryBase[T]
	client     kvx.JSON
	kv         kvx.KV
	pipeline   mo.Option[pipelineProvider]
	serializer mapping.Serializer
}

func NewJSONRepository[T any](client kvx.JSON, kv kvx.KV, keyPrefix string, options ...JSONRepositoryOption[T]) *JSONRepository[T] {
	cfg := defaultJSONConfig[T](kv, keyPrefix)
	applyJSONOptions(&cfg, options...)
	repo := &JSONRepository[T]{
		base:       repositoryBase[T]{keyBuilder: cfg.keyBuilder, tagParser: cfg.tagParser, indexer: cfg.indexer},
		client:     client,
		kv:         kv,
		pipeline:   cfg.pipeline,
		serializer: cfg.serializer,
	}
	return repo
}

func NewJSONRepositoryWithClient[T any](client kvx.Client, keyPrefix string, options ...JSONRepositoryOption[T]) *JSONRepository[T] {
	options = append([]JSONRepositoryOption[T]{WithPipeline[T](client)}, options...)
	return NewJSONRepository[T](client, client, keyPrefix, options...)
}

func (r *JSONRepository[T]) Save(ctx context.Context, entity *T) error {
	return r.SaveWithExpiration(ctx, entity, 0)
}

func (r *JSONRepository[T]) SaveWithExpiration(ctx context.Context, entity *T, expiration time.Duration) error {
	metadata, err := r.base.metadata(entity)
	if err != nil {
		return err
	}
	key, err := r.base.keyBuilder.Build(entity, metadata)
	if err != nil {
		return err
	}
	data, err := r.serializer.Marshal(entity)
	if err != nil {
		return err
	}
	if err := r.client.JSONSet(ctx, key, "$", data, expiration); err != nil {
		return err
	}
	if len(metadata.IndexFields) > 0 {
		if err := r.base.indexer.IndexEntity(ctx, entity, metadata, key); err != nil {
			return err
		}
	}
	return nil
}

func (r *JSONRepository[T]) SaveBatch(ctx context.Context, entities []*T) error {
	return r.SaveBatchWithExpiration(ctx, entities, 0)
}

func (r *JSONRepository[T]) SaveBatchWithExpiration(ctx context.Context, entities []*T, expiration time.Duration) error {
	if len(entities) == 0 {
		return nil
	}
	if provider, ok := r.pipeline.Get(); ok {
		pipe := provider.Pipeline()
		defer pipe.Close()
		for _, entity := range entities {
			metadata, err := r.base.metadata(entity)
			if err != nil {
				return err
			}
			key, err := r.base.keyBuilder.Build(entity, metadata)
			if err != nil {
				return err
			}
			data, err := r.serializer.Marshal(entity)
			if err != nil {
				return err
			}
			pipe.Enqueue("JSON.SET", []byte(key), []byte("$"), data)
			enqueueExpire(pipe, key, expiration)
			if len(metadata.IndexFields) > 0 {
				if err := r.base.indexer.IndexEntity(ctx, entity, metadata, key); err != nil {
					return err
				}
			}
		}
		_, err := pipe.Exec(ctx)
		return err
	}
	for _, entity := range entities {
		if err := r.SaveWithExpiration(ctx, entity, expiration); err != nil {
			return err
		}
	}
	return nil
}

func (r *JSONRepository[T]) FindByID(ctx context.Context, id string) (*T, error) {
	return r.findByKey(ctx, r.base.keyFromID(id))
}

func (r *JSONRepository[T]) FindByIDs(ctx context.Context, ids []string) (map[string]*T, error) {
	results := make(map[string]*T, len(ids))
	for _, id := range ids {
		entity, err := r.FindByID(ctx, id)
		if err != nil {
			if err == ErrNotFound {
				continue
			}
			return nil, err
		}
		results[id] = entity
	}
	return results, nil
}

func (r *JSONRepository[T]) Exists(ctx context.Context, id string) (bool, error) {
	return r.kv.Exists(ctx, r.base.keyFromID(id))
}

func (r *JSONRepository[T]) ExistsBatch(ctx context.Context, ids []string) (map[string]bool, error) {
	keys := r.base.keysFromIDs(ids)
	existsMap, err := r.kv.ExistsMulti(ctx, keys)
	if err != nil {
		return nil, err
	}
	results := make(map[string]bool, len(ids))
	for i, id := range ids {
		results[id] = existsMap[keys[i]]
	}
	return results, nil
}

func (r *JSONRepository[T]) Delete(ctx context.Context, id string) error {
	key := r.base.keyFromID(id)
	entity, err := r.findByKey(ctx, key)
	if err != nil {
		if err == ErrNotFound {
			return nil
		}
		return err
	}
	metadata, err := r.base.metadata(entity)
	if err != nil {
		return err
	}
	if len(metadata.IndexFields) > 0 {
		if err := r.base.indexer.RemoveEntityFromIndexes(ctx, entity, metadata); err != nil {
			return err
		}
	}
	return r.client.JSONDelete(ctx, key, "$")
}

func (r *JSONRepository[T]) DeleteBatch(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if err := r.Delete(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *JSONRepository[T]) UpdateField(ctx context.Context, id string, fieldPath string, value interface{}) error {
	key := r.base.keyFromID(id)
	entity, err := r.findByKey(ctx, key)
	if err != nil {
		return err
	}
	metadata, err := r.base.metadata(entity)
	if err != nil {
		return err
	}
	fieldName := extractFieldNameFromPath(fieldPath)
	resolvedField, fieldTag, exists := metadata.ResolveField(fieldName)
	if exists && fieldTag.Index {
		if err := r.base.indexer.UpdateFieldIndex(ctx, entity, metadata, resolvedField, key); err != nil {
			return err
		}
	}
	data, err := r.serializer.Marshal(value)
	if err != nil {
		return err
	}
	return r.client.JSONSetField(ctx, key, fieldPath, data)
}

func (r *JSONRepository[T]) FindByField(ctx context.Context, fieldName string, fieldValue string) ([]*T, error) {
	ids, err := r.base.idsByField(ctx, fieldName, fieldValue)
	if err != nil {
		return nil, err
	}
	return r.findManyByIDs(ctx, ids)
}

func (r *JSONRepository[T]) FindByFields(ctx context.Context, fields map[string]string) ([]*T, error) {
	if len(fields) == 0 {
		return r.FindAll(ctx)
	}
	idGroups := make([][]string, 0, len(fields))
	for fieldName, fieldValue := range fields {
		ids, err := r.base.idsByField(ctx, fieldName, fieldValue)
		if err != nil {
			return nil, err
		}
		idGroups = append(idGroups, ids)
	}
	intersection := intersectStringSlices(idGroups...)
	if len(intersection) == 0 {
		return []*T{}, nil
	}
	return r.findManyByIDs(ctx, intersection)
}

func (r *JSONRepository[T]) FindAll(ctx context.Context) ([]*T, error) {
	keys, err := r.base.scanAllKeys(ctx, r.kv)
	if err != nil {
		return nil, err
	}
	results := make([]*T, 0, len(keys))
	for _, key := range keys {
		entity, err := r.findByKey(ctx, key)
		if err != nil {
			if err == ErrNotFound {
				continue
			}
			return nil, err
		}
		results = append(results, entity)
	}
	return results, nil
}

func (r *JSONRepository[T]) Count(ctx context.Context) (int64, error) {
	keys, err := r.base.scanAllKeys(ctx, r.kv)
	if err != nil {
		return 0, err
	}
	return int64(len(keys)), nil
}

func (r *JSONRepository[T]) findByKey(ctx context.Context, key string) (*T, error) {
	data, err := r.client.JSONGet(ctx, key, "$")
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, ErrNotFound
	}
	var entity T
	if err := r.serializer.Unmarshal(data, &entity); err != nil {
		return nil, err
	}
	metadata, err := r.base.metadataForType()
	if err != nil {
		return nil, err
	}
	if err := r.base.hydrateEntityID(&entity, metadata, key); err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *JSONRepository[T]) findManyByIDs(ctx context.Context, ids []string) ([]*T, error) {
	results := make([]*T, 0, len(ids))
	for _, id := range ids {
		entity, err := r.FindByID(ctx, id)
		if err != nil {
			if err == ErrNotFound {
				continue
			}
			return nil, err
		}
		results = append(results, entity)
	}
	return results, nil
}

func extractFieldNameFromPath(path string) string {
	if len(path) > 2 && path[:2] == "$." {
		return path[2:]
	}
	return path
}
