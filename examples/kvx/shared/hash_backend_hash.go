package shared

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strconv"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/kvx"
)

// HGet returns the field value stored in the hash at key.
func (b *HashBackend) HGet(_ context.Context, key, field string) ([]byte, error) {
	hash, ok := b.hashes[key]
	if !ok {
		return nil, kvx.ErrNil
	}

	value, ok := hash[field]
	if !ok {
		return nil, kvx.ErrNil
	}

	return value, nil
}

// HMGet returns the available field values from the hash at key.
func (b *HashBackend) HMGet(ctx context.Context, key string, fields []string) (map[string][]byte, error) {
	result := make(map[string][]byte, len(fields))
	for _, field := range fields {
		value, err := b.HGet(ctx, key, field)
		if err == nil {
			result[field] = value
		}
	}

	return result, nil
}

// HSet stores the provided field values in the hash at key.
func (b *HashBackend) HSet(_ context.Context, key string, values map[string][]byte) error {
	if _, ok := b.hashes[key]; !ok {
		b.hashes[key] = make(map[string][]byte, len(values))
	}

	maps.Copy(b.hashes[key], values)
	b.keys[key] = []byte("1")
	return nil
}

// HMSet stores the provided field values in the hash at key.
func (b *HashBackend) HMSet(ctx context.Context, key string, values map[string][]byte) error {
	return b.HSet(ctx, key, values)
}

// HGetAll returns every field value stored in the hash at key.
func (b *HashBackend) HGetAll(_ context.Context, key string) (map[string][]byte, error) {
	hash, ok := b.hashes[key]
	if !ok {
		return map[string][]byte{}, nil
	}

	result := make(map[string][]byte, len(hash))
	maps.Copy(result, hash)
	return result, nil
}

// HDel removes the provided fields from the hash at key.
func (b *HashBackend) HDel(_ context.Context, key string, fields ...string) error {
	hash, ok := b.hashes[key]
	if !ok {
		return nil
	}

	for _, field := range fields {
		delete(hash, field)
	}

	if len(hash) == 0 {
		delete(b.hashes, key)
		delete(b.keys, key)
	}

	return nil
}

// HExists reports whether field exists in the hash at key.
func (b *HashBackend) HExists(_ context.Context, key, field string) (bool, error) {
	hash, ok := b.hashes[key]
	if !ok {
		return false, nil
	}

	_, ok = hash[field]
	return ok, nil
}

// HKeys returns the field names stored in the hash at key.
func (b *HashBackend) HKeys(_ context.Context, key string) (collectionx.List[string], error) {
	hash, ok := b.hashes[key]
	if !ok {
		return collectionx.NewList[string](), nil
	}

	keys := collectionx.NewListWithCapacity[string](len(hash))
	for field := range hash {
		keys.Add(field)
	}

	return keys, nil
}

// HVals returns the field values stored in the hash at key.
func (b *HashBackend) HVals(_ context.Context, key string) (collectionx.List[[]byte], error) {
	hash, ok := b.hashes[key]
	if !ok {
		return collectionx.NewList[[]byte](), nil
	}

	values := collectionx.NewListWithCapacity[[]byte](len(hash))
	for _, value := range hash {
		values.Add(value)
	}

	return values, nil
}

// HLen returns the number of fields stored in the hash at key.
func (b *HashBackend) HLen(_ context.Context, key string) (int64, error) {
	return int64(len(b.hashes[key])), nil
}

// HIncrBy increments the integer field stored in the hash at key.
func (b *HashBackend) HIncrBy(ctx context.Context, key, field string, increment int64) (int64, error) {
	current, err := b.HGet(ctx, key, field)
	if err != nil && !errors.Is(err, kvx.ErrNil) {
		return 0, err
	}

	var value int64
	if len(current) > 0 {
		value, err = strconv.ParseInt(string(current), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse %s[%s] as int64: %w", key, field, err)
		}
	}

	value += increment
	if err := b.HSet(ctx, key, map[string][]byte{field: []byte(strconv.FormatInt(value, 10))}); err != nil {
		return 0, err
	}

	return value, nil
}
