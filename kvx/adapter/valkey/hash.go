package valkey

import (
	"context"
	"strconv"

	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/samber/lo"
)

// HGet gets a field from a hash.
func (a *Adapter) HGet(ctx context.Context, key, field string) ([]byte, error) {
	resp := a.client.Do(ctx, a.client.B().Hget().Key(key).Field(field).Build())

	return bytesFromResult("get hash field", resp)
}

// HMGet gets multiple fields from a hash.
func (a *Adapter) HMGet(ctx context.Context, key string, fields []string) (map[string][]byte, error) {
	result := make(map[string][]byte, len(fields))
	for _, field := range fields {
		value, err := a.HGet(ctx, key, field)
		if err != nil {
			if kvx.IsNil(err) {
				continue
			}
			return nil, err
		}
		result[field] = value
	}
	return result, nil
}

// HSet sets fields in a hash.
func (a *Adapter) HSet(ctx context.Context, key string, values map[string][]byte) error {
	return wrapValkeyError("set hash fields", a.client.Do(ctx, newHSetCommand(a.client, key, values)).Error())
}

// HMSet sets multiple fields in a hash.
func (a *Adapter) HMSet(ctx context.Context, key string, values map[string][]byte) error {
	return a.HSet(ctx, key, values)
}

// HGetAll gets all fields and values from a hash.
func (a *Adapter) HGetAll(ctx context.Context, key string) (map[string][]byte, error) {
	resp := a.client.Do(ctx, a.client.B().Hgetall().Key(key).Build())
	m, err := stringMapFromResult("get all hash fields", resp)
	if err != nil {
		return nil, err
	}

	return convertStringMapToBytes(m), nil
}

// HDel deletes fields from a hash.
func (a *Adapter) HDel(ctx context.Context, key string, fields ...string) error {
	return wrapValkeyError("delete hash fields", a.client.Do(ctx, a.client.B().Hdel().Key(key).Field(fields...).Build()).Error())
}

// HExists checks if a field exists in a hash.
func (a *Adapter) HExists(ctx context.Context, key, field string) (bool, error) {
	resp := a.client.Do(ctx, a.client.B().Hexists().Key(key).Field(field).Build())

	return boolFromResult("check hash field existence", resp)
}

// HKeys gets all field names in a hash.
func (a *Adapter) HKeys(ctx context.Context, key string) ([]string, error) {
	resp := a.client.Do(ctx, a.client.B().Hkeys().Key(key).Build())

	return stringSliceFromResult("list hash fields", resp)
}

// HVals gets all values in a hash.
func (a *Adapter) HVals(ctx context.Context, key string) ([][]byte, error) {
	resp := a.client.Do(ctx, a.client.B().Hvals().Key(key).Build())
	strs, err := stringSliceFromResult("list hash values", resp)
	if err != nil {
		return nil, err
	}
	return lo.Map(strs, func(value string, _ int) []byte {
		return []byte(value)
	}), nil
}

// HLen gets the number of fields in a hash.
func (a *Adapter) HLen(ctx context.Context, key string) (int64, error) {
	resp := a.client.Do(ctx, a.client.B().Hlen().Key(key).Build())

	return int64FromResult("get hash length", resp)
}

// HIncrBy increments a field by the given value.
func (a *Adapter) HIncrBy(ctx context.Context, key, field string, increment int64) (int64, error) {
	resp := a.client.Do(ctx, a.client.B().Arbitrary("HINCRBY").Args(key, field, strconv.FormatInt(increment, 10)).Build())

	return int64FromResult("increment hash field", resp)
}
