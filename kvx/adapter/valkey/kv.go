package valkey

import (
	"context"
	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/valkey-io/valkey-go"
	"time"
)

// ============== KV Interface ==============

// Get retrieves the value for the given key.
func (a *Adapter) Get(ctx context.Context, key string) ([]byte, error) {
	resp := a.client.Do(ctx, a.client.B().Get().Key(key).Build())
	if resp.Error() != nil {
		if valkey.IsValkeyNil(resp.Error()) {
			return nil, kvx.ErrNil
		}
		return nil, resp.Error()
	}
	return resp.AsBytes()
}

// MGet retrieves multiple values for the given keys.
func (a *Adapter) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	result := make(map[string][]byte, len(keys))
	for _, key := range keys {
		value, err := a.Get(ctx, key)
		if err != nil {
			if kvx.IsNil(err) {
				continue
			}
			return nil, err
		}
		result[key] = value
	}
	return result, nil
}

// Set sets the value for the given key.
func (a *Adapter) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	if expiration > 0 {
		return a.client.Do(ctx, a.client.B().Set().Key(key).Value(valkey.BinaryString(value)).Px(expiration).Build()).Error()
	}
	return a.client.Do(ctx, a.client.B().Set().Key(key).Value(valkey.BinaryString(value)).Build()).Error()
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
	return a.client.Do(ctx, a.client.B().Del().Key(key).Build()).Error()
}

// DeleteMulti deletes multiple keys.
func (a *Adapter) DeleteMulti(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	return a.client.Do(ctx, a.client.B().Arbitrary("DEL").Args(keys...).Build()).Error()
}

// Exists checks if the key exists.
func (a *Adapter) Exists(ctx context.Context, key string) (bool, error) {
	resp := a.client.Do(ctx, a.client.B().Exists().Key(key).Build())
	if resp.Error() != nil {
		return false, resp.Error()
	}
	n, err := resp.AsInt64()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// ExistsMulti checks if multiple keys exist.
func (a *Adapter) ExistsMulti(ctx context.Context, keys []string) (map[string]bool, error) {
	results := make(map[string]bool, len(keys))
	for _, key := range keys {
		exists, err := a.Exists(ctx, key)
		if err != nil {
			return nil, err
		}
		results[key] = exists
	}
	return results, nil
}

// Expire sets the expiration for the given key.
func (a *Adapter) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return a.client.Do(ctx, a.client.B().Expire().Key(key).Seconds(int64(expiration.Seconds())).Build()).Error()
}

// TTL gets the TTL for the given key.
func (a *Adapter) TTL(ctx context.Context, key string) (time.Duration, error) {
	resp := a.client.Do(ctx, a.client.B().Ttl().Key(key).Build())
	if resp.Error() != nil {
		return 0, resp.Error()
	}
	seconds, err := resp.AsInt64()
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

	start := int(cursor)
	if count <= 0 {
		count = int64(len(keys) - start)
	}
	end := start + int(count)
	if end >= len(keys) {
		return keys[start:], 0, nil
	}
	return keys[start:end], uint64(end), nil
}

// Keys returns all keys matching the pattern.
func (a *Adapter) Keys(ctx context.Context, pattern string) ([]string, error) {
	if pattern == "" {
		pattern = "*"
	}
	resp := a.client.Do(ctx, a.client.B().Arbitrary("KEYS").Args(pattern).Build())
	if resp.Error() != nil {
		return nil, resp.Error()
	}
	return resp.AsStrSlice()
}
