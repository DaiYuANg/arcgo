package valkey

import (
	"context"
	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/valkey-io/valkey-go"
	"strconv"
)

// ============== Hash Interface ==============

// HGet gets a field from a hash.
func (a *Adapter) HGet(ctx context.Context, key string, field string) ([]byte, error) {
	resp := a.client.Do(ctx, a.client.B().Hget().Key(key).Field(field).Build())
	if resp.Error() != nil {
		if valkey.IsValkeyNil(resp.Error()) {
			return nil, kvx.ErrNil
		}
		return nil, resp.Error()
	}
	return resp.AsBytes()
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
	// Build the command with FieldValue chain
	cmd := a.client.B().Hset().Key(key).FieldValue()
	for k, v := range values {
		cmd = cmd.FieldValue(k, valkey.BinaryString(v))
	}
	return a.client.Do(ctx, cmd.Build()).Error()
}

// HMSet sets multiple fields in a hash.
func (a *Adapter) HMSet(ctx context.Context, key string, values map[string][]byte) error {
	return a.HSet(ctx, key, values)
}

// HGetAll gets all fields and values from a hash.
func (a *Adapter) HGetAll(ctx context.Context, key string) (map[string][]byte, error) {
	resp := a.client.Do(ctx, a.client.B().Hgetall().Key(key).Build())
	if resp.Error() != nil {
		return nil, resp.Error()
	}

	m, err := resp.AsStrMap()
	if err != nil {
		return nil, err
	}

	result := make(map[string][]byte, len(m))
	for k, v := range m {
		result[k] = []byte(v)
	}
	return result, nil
}

// HDel deletes fields from a hash.
func (a *Adapter) HDel(ctx context.Context, key string, fields ...string) error {
	return a.client.Do(ctx, a.client.B().Hdel().Key(key).Field(fields...).Build()).Error()
}

// HExists checks if a field exists in a hash.
func (a *Adapter) HExists(ctx context.Context, key string, field string) (bool, error) {
	resp := a.client.Do(ctx, a.client.B().Hexists().Key(key).Field(field).Build())
	if resp.Error() != nil {
		return false, resp.Error()
	}
	return resp.AsBool()
}

// HKeys gets all field names in a hash.
func (a *Adapter) HKeys(ctx context.Context, key string) ([]string, error) {
	resp := a.client.Do(ctx, a.client.B().Hkeys().Key(key).Build())
	if resp.Error() != nil {
		return nil, resp.Error()
	}
	return resp.AsStrSlice()
}

// HVals gets all values in a hash.
func (a *Adapter) HVals(ctx context.Context, key string) ([][]byte, error) {
	resp := a.client.Do(ctx, a.client.B().Hvals().Key(key).Build())
	if resp.Error() != nil {
		return nil, resp.Error()
	}
	strs, err := resp.AsStrSlice()
	if err != nil {
		return nil, err
	}
	result := make([][]byte, len(strs))
	for i, v := range strs {
		result[i] = []byte(v)
	}
	return result, nil
}

// HLen gets the number of fields in a hash.
func (a *Adapter) HLen(ctx context.Context, key string) (int64, error) {
	resp := a.client.Do(ctx, a.client.B().Hlen().Key(key).Build())
	if resp.Error() != nil {
		return 0, resp.Error()
	}
	return resp.AsInt64()
}

// HIncrBy increments a field by the given value.
func (a *Adapter) HIncrBy(ctx context.Context, key string, field string, increment int64) (int64, error) {
	resp := a.client.Do(ctx, a.client.B().Arbitrary("HINCRBY").Args(key, field, strconv.FormatInt(increment, 10)).Build())
	if resp.Error() != nil {
		return 0, resp.Error()
	}
	return resp.AsInt64()
}
