package shared

import (
	"context"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/kvx"
)

// JSONBackend is an in-memory JSON backend used by the kvx examples.
type JSONBackend struct {
	data       map[string][]byte
	expiration map[string]time.Duration
}

// NewJSONBackend creates an in-memory JSON backend for local examples.
func NewJSONBackend() *JSONBackend {
	return &JSONBackend{
		data:       make(map[string][]byte),
		expiration: make(map[string]time.Duration),
	}
}

// Get returns the raw JSON document stored at key.
func (b *JSONBackend) Get(_ context.Context, key string) ([]byte, error) {
	value, ok := b.data[key]
	if !ok {
		return nil, kvx.ErrNil
	}

	return value, nil
}

// MGet returns the raw JSON documents available for the provided keys.
func (b *JSONBackend) MGet(_ context.Context, keys []string) (map[string][]byte, error) {
	result := make(map[string][]byte, len(keys))
	for _, key := range keys {
		if value, ok := b.data[key]; ok {
			result[key] = value
		}
	}

	return result, nil
}

// Set stores the raw JSON document at key.
func (b *JSONBackend) Set(_ context.Context, key string, value []byte, expiration time.Duration) error {
	b.data[key] = value
	if expiration > 0 {
		b.expiration[key] = expiration
	}

	return nil
}

// MSet stores the provided raw JSON documents.
func (b *JSONBackend) MSet(ctx context.Context, values map[string][]byte, expiration time.Duration) error {
	for key, value := range values {
		if err := b.Set(ctx, key, value, expiration); err != nil {
			return err
		}
	}

	return nil
}

// Delete removes the JSON document stored at key.
func (b *JSONBackend) Delete(_ context.Context, key string) error {
	delete(b.data, key)
	delete(b.expiration, key)
	return nil
}

// DeleteMulti removes all provided JSON documents.
func (b *JSONBackend) DeleteMulti(ctx context.Context, keys []string) error {
	for _, key := range keys {
		if err := b.Delete(ctx, key); err != nil {
			return err
		}
	}

	return nil
}

// Exists reports whether key exists.
func (b *JSONBackend) Exists(_ context.Context, key string) (bool, error) {
	_, ok := b.data[key]
	return ok, nil
}

// ExistsMulti reports existence for each provided key.
func (b *JSONBackend) ExistsMulti(_ context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool, len(keys))
	for _, key := range keys {
		_, ok := b.data[key]
		result[key] = ok
	}

	return result, nil
}

// Expire stores the configured expiration duration for key.
func (b *JSONBackend) Expire(_ context.Context, key string, expiration time.Duration) error {
	b.expiration[key] = expiration
	return nil
}

// TTL returns the configured expiration duration for key.
func (b *JSONBackend) TTL(_ context.Context, key string) (time.Duration, error) {
	return b.expiration[key], nil
}

// Scan returns keys matching the provided glob-style prefix pattern.
func (b *JSONBackend) Scan(_ context.Context, pattern string, _ uint64, _ int64) (collectionx.List[string], uint64, error) {
	keys := collectionx.NewListWithCapacity[string](len(b.data))
	for key := range b.data {
		if matchesPattern(key, pattern) {
			keys.Add(key)
		}
	}

	return keys, 0, nil
}

// Keys returns keys matching pattern.
func (b *JSONBackend) Keys(ctx context.Context, pattern string) (collectionx.List[string], error) {
	keys, _, err := b.Scan(ctx, pattern, 0, 0)
	return keys, err
}

// JSONSet stores the whole JSON document at key.
func (b *JSONBackend) JSONSet(ctx context.Context, key, _ string, value []byte, expiration time.Duration) error {
	return b.Set(ctx, key, value, expiration)
}

// JSONGet returns the whole JSON document stored at key.
func (b *JSONBackend) JSONGet(_ context.Context, key, _ string) ([]byte, error) {
	value, ok := b.data[key]
	if !ok {
		return nil, nil
	}

	return value, nil
}

// JSONDelete removes the JSON document stored at key.
func (b *JSONBackend) JSONDelete(ctx context.Context, key, _ string) error {
	return b.Delete(ctx, key)
}
