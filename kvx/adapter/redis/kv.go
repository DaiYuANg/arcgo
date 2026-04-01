package redis

import (
	"context"
	"time"

	"github.com/samber/lo"
)

// Get retrieves the value for the given key.
func (a *Adapter) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := a.client.Get(ctx, key).Result()
	val, err = wrapRedisNilResult("get value", val, err)
	if err != nil {
		return nil, err
	}

	return []byte(val), nil
}

// MGet retrieves multiple values for the given keys.
func (a *Adapter) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	vals, err := a.client.MGet(ctx, keys...).Result()
	vals, err = wrapRedisResult("get multiple values", vals, err)
	if err != nil {
		return nil, err
	}

	return lo.Reduce(vals, func(result map[string][]byte, value any, index int) map[string][]byte {
		str, ok := value.(string)
		if ok {
			result[keys[index]] = []byte(str)
		}
		return result
	}, make(map[string][]byte)), nil
}

// Set sets the value for the given key.
func (a *Adapter) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	return wrapRedisError("set value", a.client.Set(ctx, key, value, expiration).Err())
}

// MSet sets multiple key-value pairs.
func (a *Adapter) MSet(ctx context.Context, values map[string][]byte, expiration time.Duration) error {
	if err := wrapRedisError("set multiple values", a.client.MSet(ctx, convertBytesMapToAny(values)).Err()); err != nil {
		return err
	}

	if expiration > 0 {
		for key := range values {
			if err := wrapRedisError("expire key", a.client.Expire(ctx, key, expiration).Err()); err != nil {
				return err
			}
		}
	}
	return nil
}

// Delete deletes the given key.
func (a *Adapter) Delete(ctx context.Context, key string) error {
	return wrapRedisError("delete key", a.client.Del(ctx, key).Err())
}

// DeleteMulti deletes multiple keys.
func (a *Adapter) DeleteMulti(ctx context.Context, keys []string) error {
	return wrapRedisError("delete multiple keys", a.client.Del(ctx, keys...).Err())
}

// Exists checks if the key exists.
func (a *Adapter) Exists(ctx context.Context, key string) (bool, error) {
	count, err := a.client.Exists(ctx, key).Result()
	return wrapRedisResult("check key existence", count > 0, err)
}

// ExistsMulti checks if multiple keys exist.
func (a *Adapter) ExistsMulti(ctx context.Context, keys []string) (map[string]bool, error) {
	loadErr := error(nil)
	results := lo.Reduce(keys, func(acc map[string]bool, key string, _ int) map[string]bool {
		if loadErr != nil {
			return acc
		}

		exists, err := a.Exists(ctx, key)
		if err != nil {
			loadErr = err
			return acc
		}

		acc[key] = exists
		return acc
	}, make(map[string]bool, len(keys)))
	if loadErr != nil {
		return nil, loadErr
	}
	return results, nil
}

// Expire sets the expiration for the given key.
func (a *Adapter) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return wrapRedisError("expire key", a.client.Expire(ctx, key, expiration).Err())
}

// TTL gets the TTL for the given key.
func (a *Adapter) TTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := a.client.TTL(ctx, key).Result()
	return wrapRedisResult("get key ttl", ttl, err)
}

// Scan iterates over keys matching the pattern.
func (a *Adapter) Scan(ctx context.Context, pattern string, cursor uint64, count int64) ([]string, uint64, error) {
	keys, nextCursor, err := a.client.Scan(ctx, cursor, pattern, count).Result()
	if err != nil {
		return nil, 0, wrapRedisError("scan keys", err)
	}

	return keys, nextCursor, nil
}

// Keys returns all keys matching the pattern.
func (a *Adapter) Keys(ctx context.Context, pattern string) ([]string, error) {
	keys, err := a.client.Keys(ctx, pattern).Result()
	return wrapRedisResult("list keys", keys, err)
}
