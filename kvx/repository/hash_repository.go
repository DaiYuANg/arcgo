package repository

import (
	"context"
	"time"

	"github.com/DaiYuANg/archgo/kvx"
	"github.com/DaiYuANg/archgo/kvx/mapping"
)

// HashRepository provides repository operations for hash-based entities.
type HashRepository[T any] struct {
	client     kvx.Hash
	kv         kvx.KV
	clientFull kvx.Client
	keyBuilder *mapping.KeyBuilder
	tagParser  *mapping.TagParser
	codec      *mapping.HashCodec
	indexer    *Indexer[T]
}

// NewHashRepository creates a new HashRepository.
func NewHashRepository[T any](client kvx.Hash, kv kvx.KV, keyPrefix string) *HashRepository[T] {
	return &HashRepository[T]{
		client:     client,
		kv:         kv,
		keyBuilder: mapping.NewKeyBuilder(keyPrefix),
		tagParser:  mapping.NewTagParser(),
		codec:      mapping.NewHashCodec(nil),
		indexer:    NewIndexer[T](kv, keyPrefix),
	}
}

// NewHashRepositoryWithClient creates a new HashRepository with full client (for pipeline support).
func NewHashRepositoryWithClient[T any](client kvx.Client, keyPrefix string) *HashRepository[T] {
	return &HashRepository[T]{
		client:     client,
		kv:         client,
		clientFull: client,
		keyBuilder: mapping.NewKeyBuilder(keyPrefix),
		tagParser:  mapping.NewTagParser(),
		codec:      mapping.NewHashCodec(nil),
		indexer:    NewIndexer[T](client, keyPrefix),
	}
}

// NewHashRepositoryWithCodec creates a new HashRepository with custom codec.
func NewHashRepositoryWithCodec[T any](client kvx.Hash, kv kvx.KV, keyPrefix string, codec *mapping.HashCodec) *HashRepository[T] {
	return &HashRepository[T]{
		client:     client,
		kv:         kv,
		keyBuilder: mapping.NewKeyBuilder(keyPrefix),
		tagParser:  mapping.NewTagParser(),
		codec:      codec,
		indexer:    NewIndexer[T](kv, keyPrefix),
	}
}

// Save saves an entity.
func (r *HashRepository[T]) Save(ctx context.Context, entity *T) error {
	return r.SaveWithExpiration(ctx, entity, 0)
}

// SaveWithExpiration saves an entity with expiration.
func (r *HashRepository[T]) SaveWithExpiration(ctx context.Context, entity *T, expiration time.Duration) error {
	metadata, err := r.tagParser.ParseType(entity)
	if err != nil {
		return err
	}

	key, err := r.keyBuilder.Build(entity, metadata)
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

	// Update secondary indexes
	if len(metadata.IndexFields) > 0 {
		if err := r.indexer.IndexEntity(ctx, entity, metadata, key); err != nil {
			return err
		}
	}

	return nil
}

// SaveBatch saves multiple entities in batch.
func (r *HashRepository[T]) SaveBatch(ctx context.Context, entities []*T) error {
	return r.SaveBatchWithExpiration(ctx, entities, 0)
}

// SaveBatchWithExpiration saves multiple entities with expiration.
func (r *HashRepository[T]) SaveBatchWithExpiration(ctx context.Context, entities []*T, expiration time.Duration) error {
	if len(entities) == 0 {
		return nil
	}

	// Use pipeline if full client is available
	if r.clientFull != nil {
		pipe := r.clientFull.Pipeline()
		defer pipe.Close()

		for _, entity := range entities {
			metadata, err := r.tagParser.ParseType(entity)
			if err != nil {
				return err
			}

			key, err := r.keyBuilder.Build(entity, metadata)
			if err != nil {
				return err
			}

			hashData, err := r.codec.Encode(entity, metadata)
			if err != nil {
				return err
			}

			// Queue HSet command
			pipe.Enqueue("HSET", append([][]byte{[]byte(key)}, encodeHashData(hashData)...)...)

			if expiration > 0 {
				pipe.Enqueue("EXPIRE", []byte(key), []byte(expiration.String()))
			}

			// Update secondary indexes
			if len(metadata.IndexFields) > 0 {
				if err := r.indexer.IndexEntity(ctx, entity, metadata, key); err != nil {
					return err
				}
			}
		}

		_, err := pipe.Exec(ctx)
		return err
	}

	// Fallback to sequential save
	for _, entity := range entities {
		if err := r.SaveWithExpiration(ctx, entity, expiration); err != nil {
			return err
		}
	}
	return nil
}

// FindByID finds an entity by ID.
func (r *HashRepository[T]) FindByID(ctx context.Context, id string) (*T, error) {
	key := r.keyBuilder.BuildWithID(id)
	return r.findByKey(ctx, key)
}

// FindByIDs finds multiple entities by IDs.
func (r *HashRepository[T]) FindByIDs(ctx context.Context, ids []string) (map[string]*T, error) {
	if len(ids) == 0 {
		return make(map[string]*T), nil
	}

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = r.keyBuilder.BuildWithID(id)
	}

	// Use MGet for batch retrieval if available, otherwise fallback to parallel Get
	results := make(map[string]*T, len(ids))

	for i, id := range ids {
		entity, err := r.findByKey(ctx, keys[i])
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

// Exists checks if an entity exists by ID.
func (r *HashRepository[T]) Exists(ctx context.Context, id string) (bool, error) {
	key := r.keyBuilder.BuildWithID(id)
	return r.kv.Exists(ctx, key)
}

// ExistsBatch checks if multiple entities exist.
func (r *HashRepository[T]) ExistsBatch(ctx context.Context, ids []string) (map[string]bool, error) {
	if len(ids) == 0 {
		return make(map[string]bool), nil
	}

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = r.keyBuilder.BuildWithID(id)
	}

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

// Delete deletes an entity by ID.
func (r *HashRepository[T]) Delete(ctx context.Context, id string) error {
	key := r.keyBuilder.BuildWithID(id)

	// Get entity first to remove indexes
	entity, err := r.findByKey(ctx, key)
	if err != nil {
		if err == ErrNotFound {
			return nil
		}
		return err
	}

	metadata, err := r.tagParser.ParseType(entity)
	if err != nil {
		return err
	}

	// Remove from secondary indexes
	if len(metadata.IndexFields) > 0 {
		if err := r.indexer.RemoveEntityFromIndexes(ctx, entity, metadata); err != nil {
			return err
		}
	}

	return r.kv.Delete(ctx, key)
}

// DeleteBatch deletes multiple entities by IDs.
func (r *HashRepository[T]) DeleteBatch(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = r.keyBuilder.BuildWithID(id)
	}

	// Remove from indexes first
	for _, id := range ids {
		if err := r.Delete(ctx, id); err != nil {
			return err
		}
	}

	return nil
}

// FindAll finds all entities of this type using SCAN.
func (r *HashRepository[T]) FindAll(ctx context.Context) ([]*T, error) {
	pattern := r.keyBuilder.BuildWithID("*")

	var allKeys []string
	var cursor uint64

	for {
		keys, nextCursor, err := r.kv.Scan(ctx, pattern, cursor, 100)
		if err != nil {
			return nil, err
		}
		allKeys = append(allKeys, keys...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	results := make([]*T, 0, len(allKeys))
	for _, key := range allKeys {
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

// Count returns the number of entities using SCAN.
func (r *HashRepository[T]) Count(ctx context.Context) (int64, error) {
	pattern := r.keyBuilder.BuildWithID("*")

	var count int64
	var cursor uint64

	for {
		keys, nextCursor, err := r.kv.Scan(ctx, pattern, cursor, 100)
		if err != nil {
			return 0, err
		}
		count += int64(len(keys))
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return count, nil
}

// FindByField finds entities by a field value using secondary index.
func (r *HashRepository[T]) FindByField(ctx context.Context, fieldName string, fieldValue string) ([]*T, error) {
	// Use secondary index
	entityIDs, err := r.indexer.GetEntityIDsByField(ctx, fieldName, fieldValue)
	if err != nil {
		return nil, err
	}

	if len(entityIDs) == 0 {
		return []*T{}, nil
	}

	results := make([]*T, 0, len(entityIDs))
	for _, id := range entityIDs {
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

// FindByFields finds entities by multiple field values (AND condition).
func (r *HashRepository[T]) FindByFields(ctx context.Context, fields map[string]string) ([]*T, error) {
	if len(fields) == 0 {
		return r.FindAll(ctx)
	}

	// Get entity IDs for each field
	var intersection []string
	first := true

	for fieldName, fieldValue := range fields {
		entityIDs, err := r.indexer.GetEntityIDsByField(ctx, fieldName, fieldValue)
		if err != nil {
			return nil, err
		}

		if first {
			intersection = entityIDs
			first = false
		} else {
			// Intersect
			intersection = stringSliceIntersection(intersection, entityIDs)
		}

		if len(intersection) == 0 {
			return []*T{}, nil
		}
	}

	results := make([]*T, 0, len(intersection))
	for _, id := range intersection {
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

// UpdateField updates a single field of an entity.
func (r *HashRepository[T]) UpdateField(ctx context.Context, id string, fieldName string, value interface{}) error {
	key := r.keyBuilder.BuildWithID(id)

	// Get current entity for index update
	entity, err := r.findByKey(ctx, key)
	if err != nil {
		return err
	}

	metadata, err := r.tagParser.ParseType(entity)
	if err != nil {
		return err
	}

	// Check if field is indexed
	fieldTag, exists := metadata.Fields[fieldName]
	if !exists {
		return ErrFieldNotFound
	}

	// Update index if needed
	if fieldTag.Index {
		if err := r.indexer.UpdateFieldIndex(ctx, entity, metadata, fieldName, key); err != nil {
			return err
		}
	}

	// Update the field in hash - use JSON serialization for the value
	data, err := r.codec.EncodeSingleValue(value)
	if err != nil {
		return err
	}

	hashData := map[string][]byte{fieldTag.Name: data}
	return r.client.HSet(ctx, key, hashData)
}

// IncrementField increments a numeric field.
func (r *HashRepository[T]) IncrementField(ctx context.Context, id string, fieldName string, increment int64) (int64, error) {
	key := r.keyBuilder.BuildWithID(id)

	metadata, err := r.tagParser.ParseType(new(T))
	if err != nil {
		return 0, err
	}

	fieldTag, exists := metadata.Fields[fieldName]
	if !exists {
		return 0, ErrFieldNotFound
	}

	return r.client.HIncrBy(ctx, key, fieldTag.Name, increment)
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
	metadata, err := r.tagParser.ParseType(&entity)
	if err != nil {
		return nil, err
	}

	if err := r.codec.Decode(hashData, &entity, metadata); err != nil {
		return nil, err
	}

	return &entity, nil
}

// Helper functions
func encodeHashData(data map[string][]byte) [][]byte {
	result := make([][]byte, 0, len(data)*2)
	for k, v := range data {
		result = append(result, []byte(k), v)
	}
	return result
}

func stringSliceIntersection(a, b []string) []string {
	set := make(map[string]bool)
	for _, s := range a {
		set[s] = true
	}

	result := make([]string, 0)
	for _, s := range b {
		if set[s] {
			result = append(result, s)
		}
	}
	return result
}

// Errors
var (
	ErrNotFound              = &repositoryError{"not found"}
	ErrOperationNotSupported = &repositoryError{"operation not supported"}
	ErrFieldNotFound         = &repositoryError{"field not found"}
)

type repositoryError struct {
	msg string
}

func (e *repositoryError) Error() string {
	return "kvx: " + e.msg
}
