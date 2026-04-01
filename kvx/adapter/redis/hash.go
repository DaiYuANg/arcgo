package redis

import (
	"context"

	"github.com/samber/lo"
)

// HGet gets a field from a hash.
func (a *Adapter) HGet(ctx context.Context, key, field string) ([]byte, error) {
	val, err := a.client.HGet(ctx, key, field).Result()
	val, err = wrapRedisNilResult("get hash field", val, err)
	if err != nil {
		return nil, err
	}

	return []byte(val), nil
}

// HMGet gets multiple fields from a hash.
func (a *Adapter) HMGet(ctx context.Context, key string, fields []string) (map[string][]byte, error) {
	vals, err := a.client.HMGet(ctx, key, fields...).Result()
	vals, err = wrapRedisResult("get multiple hash fields", vals, err)
	if err != nil {
		return nil, err
	}

	return lo.Reduce(vals, func(result map[string][]byte, value any, index int) map[string][]byte {
		str, ok := value.(string)
		if ok {
			result[fields[index]] = []byte(str)
		}
		return result
	}, make(map[string][]byte)), nil
}

// HSet sets fields in a hash.
func (a *Adapter) HSet(ctx context.Context, key string, values map[string][]byte) error {
	return wrapRedisError("set hash fields", a.client.HSet(ctx, key, convertBytesMapToAny(values)).Err())
}

// HMSet sets multiple fields in a hash.
func (a *Adapter) HMSet(ctx context.Context, key string, values map[string][]byte) error {
	return wrapRedisError("set multiple hash fields", a.client.HMSet(ctx, key, convertBytesMapToAny(values)).Err())
}

// HGetAll gets all fields and values from a hash.
func (a *Adapter) HGetAll(ctx context.Context, key string) (map[string][]byte, error) {
	val, err := a.client.HGetAll(ctx, key).Result()
	val, err = wrapRedisResult("get all hash fields", val, err)
	if err != nil {
		return nil, err
	}

	return lo.MapValues(val, func(value string, _ string) []byte {
		return []byte(value)
	}), nil
}

// HDel deletes fields from a hash.
func (a *Adapter) HDel(ctx context.Context, key string, fields ...string) error {
	return wrapRedisError("delete hash fields", a.client.HDel(ctx, key, fields...).Err())
}

// HExists checks if a field exists in a hash.
func (a *Adapter) HExists(ctx context.Context, key, field string) (bool, error) {
	exists, err := a.client.HExists(ctx, key, field).Result()
	return wrapRedisResult("check hash field existence", exists, err)
}

// HKeys gets all field names in a hash.
func (a *Adapter) HKeys(ctx context.Context, key string) ([]string, error) {
	keys, err := a.client.HKeys(ctx, key).Result()
	return wrapRedisResult("list hash fields", keys, err)
}

// HVals gets all values in a hash.
func (a *Adapter) HVals(ctx context.Context, key string) ([][]byte, error) {
	vals, err := a.client.HVals(ctx, key).Result()
	vals, err = wrapRedisResult("list hash values", vals, err)
	if err != nil {
		return nil, err
	}

	return lo.Map(vals, func(value string, _ int) []byte {
		return []byte(value)
	}), nil
}

// HLen gets the number of fields in a hash.
func (a *Adapter) HLen(ctx context.Context, key string) (int64, error) {
	length, err := a.client.HLen(ctx, key).Result()
	return wrapRedisResult("get hash length", length, err)
}

// HIncrBy increments a field by the given value.
func (a *Adapter) HIncrBy(ctx context.Context, key, field string, increment int64) (int64, error) {
	value, err := a.client.HIncrBy(ctx, key, field, increment).Result()
	return wrapRedisResult("increment hash field", value, err)
}
