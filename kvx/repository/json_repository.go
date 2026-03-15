package repository

import (
	"context"
	"time"

	"github.com/DaiYuANg/archgo/kvx"
	"github.com/DaiYuANg/archgo/kvx/mapping"
)

// JSONRepository provides repository operations for JSON-based entities.
type JSONRepository[T any] struct {
	client     kvx.JSON
	kv         kvx.KV
	clientFull kvx.Client
	keyBuilder *mapping.KeyBuilder
	tagParser  *mapping.TagParser
	serializer mapping.Serializer
	indexer    *Indexer[T]
}

// NewJSONRepository creates a new JSONRepository.
func NewJSONRepository[T any](client kvx.JSON, kv kvx.KV, keyPrefix string) *JSONRepository[T] {
	return &JSONRepository[T]{
		client:     client,
		kv:         kv,
		keyBuilder: mapping.NewKeyBuilder(keyPrefix),
		tagParser:  mapping.NewTagParser(),
		serializer: mapping.NewJSONSerializer(),
		indexer:    NewIndexer[T](kv, keyPrefix),
	}
}

// NewJSONRepositoryWithClient creates a new JSONRepository with full client (for pipeline support).
func NewJSONRepositoryWithClient[T any](client kvx.Client, keyPrefix string) *JSONRepository[T] {
	return &JSONRepository[T]{
		client:     client,
		kv:         client,
		clientFull: client,
		keyBuilder: mapping.NewKeyBuilder(keyPrefix),
		tagParser:  mapping.NewTagParser(),
		serializer: mapping.NewJSONSerializer(),
		indexer:    NewIndexer[T](client, keyPrefix),
	}
}

// Save saves an entity as JSON.
func (r *JSONRepository[T]) Save(ctx context.Context, entity *T) error {
	return r.SaveWithExpiration(ctx, entity, 0)
}

// SaveWithExpiration saves an entity as JSON with expiration.
func (r *JSONRepository[T]) SaveWithExpiration(ctx context.Context, entity *T, expiration time.Duration) error {
	metadata, err := r.tagParser.ParseType(entity)
	if err != nil {
		return err
	}

	key, err := r.keyBuilder.Build(entity, metadata)
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

	// Update secondary indexes
	if len(metadata.IndexFields) > 0 {
		if err := r.indexer.IndexEntity(ctx, entity, metadata, key); err != nil {
			return err
		}
	}

	return nil
}

// SaveBatch saves multiple entities in batch.
func (r *JSONRepository[T]) SaveBatch(ctx context.Context, entities []*T) error {
	return r.SaveBatchWithExpiration(ctx, entities, 0)
}

// SaveBatchWithExpiration saves multiple entities with expiration.
func (r *JSONRepository[T]) SaveBatchWithExpiration(ctx context.Context, entities []*T, expiration time.Duration) error {
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

			data, err := r.serializer.Marshal(entity)
			if err != nil {
				return err
			}

			// Queue JSON.SET command
			pipe.Enqueue("JSON.SET", []byte(key), []byte("$"), data)

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
func (r *JSONRepository[T]) FindByID(ctx context.Context, id string) (*T, error) {
	key := r.keyBuilder.BuildWithID(id)
	return r.findByKey(ctx, key)
}

// FindByIDs finds multiple entities by IDs.
func (r *JSONRepository[T]) FindByIDs(ctx context.Context, ids []string) (map[string]*T, error) {
	if len(ids) == 0 {
		return make(map[string]*T), nil
	}

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

// Exists checks if an entity exists by ID.
func (r *JSONRepository[T]) Exists(ctx context.Context, id string) (bool, error) {
	key := r.keyBuilder.BuildWithID(id)
	data, err := r.client.JSONGet(ctx, key, "$")
	if err != nil {
		return false, err
	}
	return len(data) > 0, nil
}

// ExistsBatch checks if multiple entities exist.
func (r *JSONRepository[T]) ExistsBatch(ctx context.Context, ids []string) (map[string]bool, error) {
	if len(ids) == 0 {
		return make(map[string]bool), nil
	}

	results := make(map[string]bool, len(ids))
	for _, id := range ids {
		exists, err := r.Exists(ctx, id)
		if err != nil {
			return nil, err
		}
		results[id] = exists
	}

	return results, nil
}

// Delete deletes an entity by ID.
func (r *JSONRepository[T]) Delete(ctx context.Context, id string) error {
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

	return r.client.JSONDelete(ctx, key, "$")
}

// DeleteBatch deletes multiple entities by IDs.
func (r *JSONRepository[T]) DeleteBatch(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	for _, id := range ids {
		if err := r.Delete(ctx, id); err != nil {
			return err
		}
	}

	return nil
}

// UpdateField updates a single field in the entity.
func (r *JSONRepository[T]) UpdateField(ctx context.Context, id string, fieldPath string, value interface{}) error {
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

	// Update index if needed
	fieldName := extractFieldNameFromPath(fieldPath)
	if fieldTag, exists := metadata.Fields[fieldName]; exists && fieldTag.Index {
		if err := r.indexer.UpdateFieldIndex(ctx, entity, metadata, fieldName, key); err != nil {
			return err
		}
	}

	data, err := r.serializer.Marshal(value)
	if err != nil {
		return err
	}

	return r.client.JSONSetField(ctx, key, fieldPath, data)
}

// FindByField finds entities by a field value using secondary index.
func (r *JSONRepository[T]) FindByField(ctx context.Context, fieldName string, fieldValue string) ([]*T, error) {
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
func (r *JSONRepository[T]) FindByFields(ctx context.Context, fields map[string]string) ([]*T, error) {
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

// FindAll finds all entities of this type using SCAN.
func (r *JSONRepository[T]) FindAll(ctx context.Context) ([]*T, error) {
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
func (r *JSONRepository[T]) Count(ctx context.Context) (int64, error) {
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

	return &entity, nil
}

// extractFieldNameFromPath extracts field name from JSON path.
func extractFieldNameFromPath(path string) string {
	// Remove leading $.
	if len(path) > 2 && path[:2] == "$." {
		return path[2:]
	}
	return path
}
