package repository

import (
	"context"
	"time"

	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/DaiYuANg/arcgo/kvx/mapping"
	"github.com/samber/mo"
)

// HashRepository provides repository operations for hash-based entities.
type HashRepository[T any] struct {
	base     repositoryBase[T]
	client   kvx.Hash
	kv       kvx.KV
	pipeline mo.Option[pipelineProvider]
	codec    *mapping.HashCodec
}

// NewHashRepository creates a new HashRepository.
func NewHashRepository[T any](client kvx.Hash, kv kvx.KV, keyPrefix string, options ...HashRepositoryOption[T]) *HashRepository[T] {
	cfg := defaultHashConfig[T](kv, keyPrefix)
	applyHashOptions(&cfg, options...)

	repo := &HashRepository[T]{
		base: repositoryBase[T]{
			keyBuilder: cfg.keyBuilder,
			tagParser:  cfg.tagParser,
			indexer:    cfg.indexer,
		},
		client:   client,
		kv:       kv,
		pipeline: cfg.pipeline,
		codec:    cfg.codec,
	}
	return repo
}

// NewHashRepositoryWithClient creates a new HashRepository with full client (for pipeline support).
func NewHashRepositoryWithClient[T any](client kvx.Client, keyPrefix string, options ...HashRepositoryOption[T]) *HashRepository[T] {
	options = append([]HashRepositoryOption[T]{WithPipeline[T](client)}, options...)
	return NewHashRepository[T](client, client, keyPrefix, options...)
}

// NewHashRepositoryWithCodec creates a new HashRepository with custom codec.
func NewHashRepositoryWithCodec[T any](client kvx.Hash, kv kvx.KV, keyPrefix string, codec *mapping.HashCodec, options ...HashRepositoryOption[T]) *HashRepository[T] {
	return NewHashRepository[T](client, kv, keyPrefix, append([]HashRepositoryOption[T]{WithHashCodec[T](codec)}, options...)...)
}

func (r *HashRepository[T]) Save(ctx context.Context, entity *T) error {
	return r.SaveWithExpiration(ctx, entity, 0)
}

func (r *HashRepository[T]) SaveWithExpiration(ctx context.Context, entity *T, expiration time.Duration) error {
	metadata, err := r.base.metadata(entity)
	if err != nil {
		return err
	}
	key, err := r.base.keyBuilder.Build(entity, metadata)
	if err != nil {
		return err
	}
	hashData, err := r.codec.Encode(entity, metadata)
	if err != nil {
		return err
	}
	if err := r.client.HSet(ctx, key, hashData); err != nil {
		return err
	}
	if expiration > 0 {
		if err := r.kv.Expire(ctx, key, expiration); err != nil {
			return err
		}
	}
	if len(metadata.IndexFields) > 0 {
		if err := r.base.indexer.IndexEntity(ctx, entity, metadata, key); err != nil {
			return err
		}
	}
	return nil
}

func (r *HashRepository[T]) SaveBatch(ctx context.Context, entities []*T) error {
	return r.SaveBatchWithExpiration(ctx, entities, 0)
}

func (r *HashRepository[T]) SaveBatchWithExpiration(ctx context.Context, entities []*T, expiration time.Duration) error {
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
			hashData, err := r.codec.Encode(entity, metadata)
			if err != nil {
				return err
			}
			pipe.Enqueue("HSET", append([][]byte{[]byte(key)}, encodeHashData(hashData)...)...)
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

func (r *HashRepository[T]) FindByID(ctx context.Context, id string) (*T, error) {
	return r.findByKey(ctx, r.base.keyFromID(id))
}

func (r *HashRepository[T]) FindByIDs(ctx context.Context, ids []string) (map[string]*T, error) {
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

func (r *HashRepository[T]) Exists(ctx context.Context, id string) (bool, error) {
	return r.kv.Exists(ctx, r.base.keyFromID(id))
}

func (r *HashRepository[T]) ExistsBatch(ctx context.Context, ids []string) (map[string]bool, error) {
	keys := r.base.keysFromIDs(ids)
	existsMap, err := r.kv.ExistsMulti(ctx, keys)
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(ids))
	for i, id := range ids {
		result[id] = existsMap[keys[i]]
	}
	return result, nil
}

func (r *HashRepository[T]) Delete(ctx context.Context, id string) error {
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
	fields, err := r.client.HKeys(ctx, key)
	if err != nil {
		return err
	}
	if len(fields) > 0 {
		if err := r.client.HDel(ctx, key, fields...); err != nil {
			return err
		}
	}
	return r.kv.Delete(ctx, key)
}

func (r *HashRepository[T]) DeleteBatch(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if err := r.Delete(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *HashRepository[T]) FindAll(ctx context.Context) ([]*T, error) {
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

func (r *HashRepository[T]) Count(ctx context.Context) (int64, error) {
	keys, err := r.base.scanAllKeys(ctx, r.kv)
	if err != nil {
		return 0, err
	}
	return int64(len(keys)), nil
}

func (r *HashRepository[T]) FindByField(ctx context.Context, fieldName string, fieldValue string) ([]*T, error) {
	entityIDs, err := r.base.idsByField(ctx, fieldName, fieldValue)
	if err != nil {
		return nil, err
	}
	return r.findManyByIDs(ctx, entityIDs)
}

func (r *HashRepository[T]) FindByFields(ctx context.Context, fields map[string]string) ([]*T, error) {
	if len(fields) == 0 {
		return r.FindAll(ctx)
	}
	idGroups := make([][]string, 0, len(fields))
	for fieldName, fieldValue := range fields {
		entityIDs, err := r.base.idsByField(ctx, fieldName, fieldValue)
		if err != nil {
			return nil, err
		}
		idGroups = append(idGroups, entityIDs)
	}
	intersection := intersectStringSlices(idGroups...)
	if len(intersection) == 0 {
		return []*T{}, nil
	}
	return r.findManyByIDs(ctx, intersection)
}

func (r *HashRepository[T]) UpdateField(ctx context.Context, id string, fieldName string, value interface{}) error {
	key := r.base.keyFromID(id)
	entity, err := r.findByKey(ctx, key)
	if err != nil {
		return err
	}
	metadata, err := r.base.metadata(entity)
	if err != nil {
		return err
	}
	resolvedField, fieldTag, exists := metadata.ResolveField(fieldName)
	if !exists {
		return ErrFieldNotFound
	}
	data, err := r.codec.EncodeSingleValue(value)
	if err != nil {
		return err
	}
	if fieldTag.Index {
		if err := r.base.indexer.UpdateFieldIndex(ctx, entity, metadata, resolvedField, key); err != nil {
			return err
		}
	}
	return r.client.HSet(ctx, key, map[string][]byte{fieldTag.StorageName(): data})
}

func (r *HashRepository[T]) IncrementField(ctx context.Context, id string, fieldName string, increment int64) (int64, error) {
	metadata, err := r.base.metadataForType()
	if err != nil {
		return 0, err
	}
	_, fieldTag, exists := metadata.ResolveField(fieldName)
	if !exists {
		return 0, ErrFieldNotFound
	}
	return r.client.HIncrBy(ctx, r.base.keyFromID(id), fieldTag.StorageName(), increment)
}

func (r *HashRepository[T]) findByKey(ctx context.Context, key string) (*T, error) {
	hashData, err := r.client.HGetAll(ctx, key)
	if err != nil {
		return nil, err
	}
	if len(hashData) == 0 {
		return nil, ErrNotFound
	}
	var entity T
	metadata, err := r.base.metadataForType()
	if err != nil {
		return nil, err
	}
	if err := r.codec.Decode(hashData, &entity, metadata); err != nil {
		return nil, err
	}
	if err := r.base.hydrateEntityID(&entity, metadata, key); err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *HashRepository[T]) findManyByIDs(ctx context.Context, ids []string) ([]*T, error) {
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

func encodeHashData(data map[string][]byte) [][]byte {
	result := make([][]byte, 0, len(data)*2)
	for k, v := range data {
		result = append(result, []byte(k), v)
	}
	return result
}

var (
	ErrNotFound              = &repositoryError{"not found"}
	ErrOperationNotSupported = &repositoryError{"operation not supported"}
	ErrFieldNotFound         = &repositoryError{"field not found"}
)

type repositoryError struct{ msg string }

func (e *repositoryError) Error() string { return "kvx: " + e.msg }
