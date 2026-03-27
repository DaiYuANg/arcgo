package valkey

import (
	"context"
	"time"

	"github.com/valkey-io/valkey-go"
)

// JSONSet sets a JSON value at key.
func (a *Adapter) JSONSet(ctx context.Context, key, path string, value []byte, expiration time.Duration) error {
	err := wrapValkeyError("set json value", a.client.Do(ctx, a.client.B().JsonSet().Key(key).Path(path).Value(valkey.BinaryString(value)).Build()).Error())
	if err != nil {
		return err
	}

	if expiration > 0 {
		return wrapValkeyError("expire json key", a.client.Do(ctx, a.client.B().Expire().Key(key).Seconds(int64(expiration.Seconds())).Build()).Error())
	}

	return nil
}

// JSONGet gets a JSON value at key.
func (a *Adapter) JSONGet(ctx context.Context, key, path string) ([]byte, error) {
	resp := a.client.Do(ctx, a.client.B().JsonGet().Key(key).Path(path).Build())

	return bytesFromResult("get json value", resp)
}

// JSONSetField sets a field in a JSON document.
func (a *Adapter) JSONSetField(ctx context.Context, key, path string, value []byte) error {
	return wrapValkeyError("set json field", a.client.Do(ctx, a.client.B().JsonSet().Key(key).Path(path).Value(valkey.BinaryString(value)).Build()).Error())
}

// JSONGetField gets a field from a JSON document.
func (a *Adapter) JSONGetField(ctx context.Context, key, path string) ([]byte, error) {
	return a.JSONGet(ctx, key, path)
}

// JSONDelete deletes a JSON value or field.
func (a *Adapter) JSONDelete(ctx context.Context, key, path string) error {
	return wrapValkeyError("delete json value", a.client.Do(ctx, a.client.B().JsonDel().Key(key).Path(path).Build()).Error())
}
