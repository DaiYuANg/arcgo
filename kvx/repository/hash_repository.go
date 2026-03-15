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
	keyBuilder *mapping.KeyBuilder
	tagParser  *mapping.TagParser
	codec      *mapping.HashCodec
}

// NewHashRepository creates a new HashRepository.
func NewHashRepository[T any](client kvx.Hash, keyPrefix string) *HashRepository[T] {
	return &HashRepository[T]{
		client:     client,
		keyBuilder: mapping.NewKeyBuilder(keyPrefix),
		tagParser:  mapping.NewTagParser(),
		codec:      mapping.NewHashCodec(nil),
	}
}

// NewHashRepositoryWithCodec creates a new HashRepository with custom codec.
func NewHashRepositoryWithCodec[T any](client kvx.Hash, keyPrefix string, codec *mapping.HashCodec) *HashRepository[T] {
	return &HashRepository[T]{
		client:     client,
		keyBuilder: mapping.NewKeyBuilder(keyPrefix),
		tagParser:  mapping.NewTagParser(),
		codec:      codec,
	}
}

// Save saves an entity.
func (r *HashRepository[T]) Save(ctx context.Context, entity *T) error {
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

	return r.client.HSet(ctx, key, hashData)
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

	// Note: Need KV interface for Expire, can be added if client supports it
	return nil
}

// FindByID finds an entity by ID.
func (r *HashRepository[T]) FindByID(ctx context.Context, id string) (*T, error) {
	key := r.keyBuilder.BuildWithID(id)
	return r.findByKey(ctx, key)
}

// Exists checks if an entity exists by ID.
func (r *HashRepository[T]) Exists(ctx context.Context, id string) (bool, error) {
	key := r.keyBuilder.BuildWithID(id)
	return r.client.HExists(ctx, key, "_") // Check if hash exists
}

// Delete deletes an entity by ID.
func (r *HashRepository[T]) Delete(ctx context.Context, id string) error {
	key := r.keyBuilder.BuildWithID(id)

	// Get all fields first
	hashData, err := r.client.HGetAll(ctx, key)
	if err != nil {
		return err
	}

	if len(hashData) == 0 {
		return nil // Already deleted
	}

	// Delete all fields
	fields := make([]string, 0, len(hashData))
	for field := range hashData {
		fields = append(fields, field)
	}

	return r.client.HDel(ctx, key, fields...)
}

// FindAll finds all entities of this type.
func (r *HashRepository[T]) FindAll(ctx context.Context) ([]*T, error) {
	// Note: This requires KEYS or SCAN which should be added to the interface
	// For now, this is a placeholder that returns an error
	return nil, ErrOperationNotSupported
}

// Count returns the number of entities.
func (r *HashRepository[T]) Count(ctx context.Context) (int64, error) {
	// Note: This requires KEYS or SCAN which should be added to the interface
	return 0, ErrOperationNotSupported
}

// FindByField finds entities by a field value (requires secondary index).
func (r *HashRepository[T]) FindByField(ctx context.Context, fieldName string, fieldValue string) ([]*T, error) {
	// This requires secondary index support
	// Placeholder for now
	return nil, ErrOperationNotSupported
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

// Errors
var (
	ErrNotFound              = &repositoryError{"not found"}
	ErrOperationNotSupported = &repositoryError{"operation not supported"}
)

type repositoryError struct {
	msg string
}

func (e *repositoryError) Error() string {
	return "kvx: " + e.msg
}
