package shared

import (
	"context"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/kvx"
)

// HashBackend is an in-memory hash backend used by the kvx examples.
type HashBackend struct {
	hashes     map[string]map[string][]byte
	keys       map[string][]byte
	expiration map[string]time.Duration
}

// NewHashBackend creates an in-memory hash backend for local examples.
func NewHashBackend() *HashBackend {
	return &HashBackend{
		hashes:     make(map[string]map[string][]byte),
		keys:       make(map[string][]byte),
		expiration: make(map[string]time.Duration),
	}
}

// Get returns the raw value stored at key.
func (b *HashBackend) Get(_ context.Context, key string) ([]byte, error) {
	value, ok := b.keys[key]
	if !ok {
		return nil, kvx.ErrNil
	}

	return value, nil
}

// MGet returns the values available for the provided keys.
func (b *HashBackend) MGet(_ context.Context, keys []string) (map[string][]byte, error) {
	result := make(map[string][]byte, len(keys))
	for _, key := range keys {
		if value, ok := b.keys[key]; ok {
			result[key] = value
		}
	}

	return result, nil
}

// Set stores the raw value at key.
func (b *HashBackend) Set(_ context.Context, key string, value []byte, expiration time.Duration) error {
	b.keys[key] = value
	if expiration > 0 {
		b.expiration[key] = expiration
	}

	return nil
}

// MSet stores the provided values.
func (b *HashBackend) MSet(ctx context.Context, values map[string][]byte, expiration time.Duration) error {
	for key, value := range values {
		if err := b.Set(ctx, key, value, expiration); err != nil {
			return err
		}
	}

	return nil
}

// Delete removes any value stored at key.
func (b *HashBackend) Delete(_ context.Context, key string) error {
	delete(b.keys, key)
	delete(b.hashes, key)
	delete(b.expiration, key)
	return nil
}

// DeleteMulti removes all provided keys.
func (b *HashBackend) DeleteMulti(ctx context.Context, keys []string) error {
	for _, key := range keys {
		if err := b.Delete(ctx, key); err != nil {
			return err
		}
	}

	return nil
}

// Exists reports whether key exists.
func (b *HashBackend) Exists(_ context.Context, key string) (bool, error) {
	_, ok := b.keys[key]
	return ok, nil
}

// ExistsMulti reports existence for each provided key.
func (b *HashBackend) ExistsMulti(_ context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool, len(keys))
	for _, key := range keys {
		_, ok := b.keys[key]
		result[key] = ok
	}

	return result, nil
}

// Expire stores the configured expiration duration for key.
func (b *HashBackend) Expire(_ context.Context, key string, expiration time.Duration) error {
	b.expiration[key] = expiration
	return nil
}

// TTL returns the configured expiration duration for key.
func (b *HashBackend) TTL(_ context.Context, key string) (time.Duration, error) {
	return b.expiration[key], nil
}

// Scan returns keys matching the provided glob-style prefix pattern.
func (b *HashBackend) Scan(_ context.Context, pattern string, _ uint64, _ int64) (collectionx.List[string], uint64, error) {
	keys := collectionx.NewListWithCapacity[string](len(b.keys))
	for key := range b.keys {
		if matchesPattern(key, pattern) {
			keys.Add(key)
		}
	}

	return keys, 0, nil
}

// Keys returns keys matching pattern.
func (b *HashBackend) Keys(ctx context.Context, pattern string) (collectionx.List[string], error) {
	keys, _, err := b.Scan(ctx, pattern, 0, 0)
	return keys, err
}
