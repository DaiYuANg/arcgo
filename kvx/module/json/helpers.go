package json

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

func (j *JSON) getDocumentData(ctx context.Context, key string) ([]byte, error) {
	data, err := j.client.JSONGet(ctx, key, "$")
	if err != nil {
		return nil, fmt.Errorf("get JSON document %q: %w", key, err)
	}
	return data, nil
}

func (j *JSON) getPathData(ctx context.Context, key, path string) ([]byte, error) {
	data, err := j.client.JSONGetField(ctx, key, path)
	if err != nil {
		return nil, fmt.Errorf("get JSON path %q from %q: %w", path, key, err)
	}
	return data, nil
}

func (j *JSON) setDocumentData(ctx context.Context, key string, data []byte, expiration time.Duration) error {
	if err := j.client.JSONSet(ctx, key, "$", data, expiration); err != nil {
		return fmt.Errorf("set JSON document %q: %w", key, err)
	}
	return nil
}

func (j *JSON) setPathData(ctx context.Context, key, path string, data []byte) error {
	if err := j.client.JSONSetField(ctx, key, path, data); err != nil {
		return fmt.Errorf("set JSON path %q for %q: %w", path, key, err)
	}
	return nil
}

func (j *JSON) deletePath(ctx context.Context, key, path string) error {
	if err := j.client.JSONDelete(ctx, key, path); err != nil {
		return fmt.Errorf("delete JSON path %q from %q: %w", path, key, err)
	}
	return nil
}

func marshalJSONValue(op string, value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return data, nil
}

func unmarshalJSONValue(data []byte, dest any, op string) error {
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}
