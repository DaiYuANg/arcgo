package redis

import (
	"context"
	"time"
)

// JSONSet sets a JSON value at key.
func (a *Adapter) JSONSet(ctx context.Context, key, path string, value []byte, expiration time.Duration) error {
	err := wrapRedisError("set json value", a.client.Do(ctx, "JSON.SET", key, path, value).Err())
	if err != nil {
		return err
	}

	if expiration > 0 {
		return wrapRedisError("expire json key", a.client.Expire(ctx, key, expiration).Err())
	}

	return nil
}

// JSONGet gets a JSON value at key.
func (a *Adapter) JSONGet(ctx context.Context, key, path string) ([]byte, error) {
	val, err := a.client.Do(ctx, "JSON.GET", key, path).Result()
	val, err = wrapRedisNilResult("get json value", val, err)
	if err != nil {
		return nil, err
	}

	return valueToBytes(val), nil
}

// JSONSetField sets a field in a JSON document.
func (a *Adapter) JSONSetField(ctx context.Context, key, path string, value []byte) error {
	return wrapRedisError("set json field", a.client.Do(ctx, "JSON.SET", key, path, value).Err())
}

// JSONGetField gets a field from a JSON document.
func (a *Adapter) JSONGetField(ctx context.Context, key, path string) ([]byte, error) {
	return a.JSONGet(ctx, key, path)
}

// JSONDelete deletes a JSON value or field.
func (a *Adapter) JSONDelete(ctx context.Context, key, path string) error {
	return wrapRedisError("delete json value", a.client.Do(ctx, "JSON.DEL", key, path).Err())
}
