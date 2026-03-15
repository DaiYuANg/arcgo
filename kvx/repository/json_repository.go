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
	keyBuilder *mapping.KeyBuilder
	tagParser  *mapping.TagParser
	serializer mapping.Serializer
}

// NewJSONRepository creates a new JSONRepository.
func NewJSONRepository[T any](client kvx.JSON, kv kvx.KV, keyPrefix string) *JSONRepository[T] {
	return &JSONRepository[T]{
		client:     client,
		kv:         kv,
		keyBuilder: mapping.NewKeyBuilder(keyPrefix),
		tagParser:  mapping.NewTagParser(),
		serializer: mapping.NewJSONSerializer(),
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

	return r.client.JSONSet(ctx, key, "$", data, expiration)
}

// FindByID finds an entity by ID.
func (r *JSONRepository[T]) FindByID(ctx context.Context, id string) (*T, error) {
	key := r.keyBuilder.BuildWithID(id)
	return r.findByKey(ctx, key)
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

// Delete deletes an entity by ID.
func (r *JSONRepository[T]) Delete(ctx context.Context, id string) error {
	key := r.keyBuilder.BuildWithID(id)
	return r.client.JSONDelete(ctx, key, "$")
}

// UpdateField updates a single field in the entity.
func (r *JSONRepository[T]) UpdateField(ctx context.Context, id string, fieldPath string, value interface{}) error {
	key := r.keyBuilder.BuildWithID(id)

	data, err := r.serializer.Marshal(value)
	if err != nil {
		return err
	}

	return r.client.JSONSetField(ctx, key, fieldPath, data)
}

// FindByField finds entities by a field value.
func (r *JSONRepository[T]) FindByField(ctx context.Context, fieldPath string, value interface{}) ([]*T, error) {
	// This requires RediSearch or similar secondary index support
	return nil, ErrOperationNotSupported
}

// FindAll finds all entities of this type.
func (r *JSONRepository[T]) FindAll(ctx context.Context) ([]*T, error) {
	return nil, ErrOperationNotSupported
}

// Count returns the number of entities.
func (r *JSONRepository[T]) Count(ctx context.Context) (int64, error) {
	return 0, ErrOperationNotSupported
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
