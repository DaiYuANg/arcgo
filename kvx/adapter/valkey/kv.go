package valkey

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/samber/lo"
	"github.com/valkey-io/valkey-go"
)

// Get retrieves the value for the given key.
func (a *Adapter) Get(ctx context.Context, key string) ([]byte, error) {
	resp := a.client.Do(ctx, a.client.B().Get().Key(key).Build())

	return bytesFromResult("get value", resp)
}

// MGet retrieves multiple values for the given keys.
func (a *Adapter) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	loadErr := error(nil)
	result := lo.Reduce(keys, func(acc map[string][]byte, key string, _ int) map[string][]byte {
		if loadErr != nil {
			return acc
		}

		value, err := a.Get(ctx, key)
		if err != nil {
			if kvx.IsNil(err) {
				return acc
			}
			loadErr = err
			return acc
		}

		acc[key] = value
		return acc
	}, make(map[string][]byte, len(keys)))
	if loadErr != nil {
		return nil, loadErr
	}
	return result, nil
}

// Set sets the value for the given key.
func (a *Adapter) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	if expiration > 0 {
		return wrapValkeyError("set value", a.client.Do(ctx, a.client.B().Set().Key(key).Value(valkey.BinaryString(value)).Px(expiration).Build()).Error())
	}

	return wrapValkeyError("set value", a.client.Do(ctx, a.client.B().Set().Key(key).Value(valkey.BinaryString(value)).Build()).Error())
}

// MSet sets multiple key-value pairs.
func (a *Adapter) MSet(ctx context.Context, values map[string][]byte, expiration time.Duration) error {
	for key, value := range values {
		if err := a.Set(ctx, key, value, expiration); err != nil {
			return err
		}
	}
	return nil
}

// Delete deletes the given key.
func (a *Adapter) Delete(ctx context.Context, key string) error {
	return wrapValkeyError("delete key", a.client.Do(ctx, a.client.B().Del().Key(key).Build()).Error())
}

// DeleteMulti deletes multiple keys.
func (a *Adapter) DeleteMulti(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	return wrapValkeyError("delete multiple keys", a.client.Do(ctx, a.client.B().Arbitrary("DEL").Args(keys...).Build()).Error())
}

// Exists checks if the key exists.
func (a *Adapter) Exists(ctx context.Context, key string) (bool, error) {
	resp := a.client.Do(ctx, a.client.B().Exists().Key(key).Build())
	n, err := int64FromResult("check key existence", resp)
	if err != nil {
		return false, err
	}

	return n > 0, nil
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
	return wrapValkeyError("expire key", a.client.Do(ctx, a.client.B().Expire().Key(key).Seconds(int64(expiration.Seconds())).Build()).Error())
}

// TTL gets the TTL for the given key.
func (a *Adapter) TTL(ctx context.Context, key string) (time.Duration, error) {
	resp := a.client.Do(ctx, a.client.B().Ttl().Key(key).Build())
	seconds, err := int64FromResult("get key ttl", resp)
	if err != nil {
		return 0, err
	}

	return time.Duration(seconds) * time.Second, nil
}

// Scan iterates over keys matching the pattern.
func (a *Adapter) Scan(ctx context.Context, pattern string, cursor uint64, count int64) ([]string, uint64, error) {
	keys, err := a.Keys(ctx, pattern)
	if err != nil {
		return nil, 0, err
	}
	if cursor >= uint64(len(keys)) {
		return []string{}, 0, nil
	}

	if cursor > uint64(math.MaxInt) {
		return nil, 0, fmt.Errorf("valkey scan cursor exceeds int range: %d", cursor)
	}

	start := int(cursor)
	if count <= 0 {
		count = int64(len(keys) - start)
	}
	end := start + int(count)
	if end >= len(keys) {
		return keys[start:], 0, nil
	}

	window := keys[start:end]
	return window, cursor + uint64(len(window)), nil
}

// Keys returns all keys matching the pattern.
func (a *Adapter) Keys(ctx context.Context, pattern string) ([]string, error) {
	if pattern == "" {
		pattern = "*"
	}
	resp := a.client.Do(ctx, a.client.B().Arbitrary("KEYS").Args(pattern).Build())

	return stringSliceFromResult("list keys", resp)
}
