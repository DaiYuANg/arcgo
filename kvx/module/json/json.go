// Package json provides JSON document operations.
package json

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/samber/lo"
)

// JSON provides high-level JSON document operations.
type JSON struct {
	client kvx.JSON
}

// NewJSON creates a new JSON instance.
func NewJSON(client kvx.JSON) *JSON {
	return &JSON{client: client}
}

// Document represents a JSON document with metadata.
type Document struct {
	Key        string
	Path       string
	Data       []byte
	Expiration time.Duration
}

// Set sets a JSON document at the specified key.
func (j *JSON) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	data, err := marshalJSONValue("marshal JSON document", value)
	if err != nil {
		return fmt.Errorf("marshal JSON array values: %w", err)
	}
	return j.setDocumentData(ctx, key, data, expiration)
}

// SetPath sets a JSON value at a specific path.
func (j *JSON) SetPath(ctx context.Context, key, path string, value any) error {
	data, err := marshalJSONValue("marshal JSON path value", value)
	if err != nil {
		return fmt.Errorf("marshal JSON array values: %w", err)
	}
	return j.setPathData(ctx, key, path, data)
}

// Get gets a JSON document by key.
func (j *JSON) Get(ctx context.Context, key string, dest any) error {
	data, err := j.getDocumentData(ctx, key)
	if err != nil {
		return fmt.Errorf("marshal JSON array values: %w", err)
	}
	if len(data) == 0 {
		return fmt.Errorf("document not found: %s", key)
	}
	return unmarshalJSONValue(data, dest, fmt.Sprintf("unmarshal JSON document %q", key))
}

// GetPath gets a JSON value at a specific path.
func (j *JSON) GetPath(ctx context.Context, key, path string, dest any) error {
	data, err := j.getPathData(ctx, key, path)
	if err != nil {
		return fmt.Errorf("marshal JSON array values: %w", err)
	}
	if len(data) == 0 {
		return fmt.Errorf("path not found: %s.%s", key, path)
	}
	return unmarshalJSONValue(data, dest, fmt.Sprintf("unmarshal JSON path %q from %q", path, key))
}

// Delete deletes a JSON document or a path within it.
func (j *JSON) Delete(ctx context.Context, key string, paths ...string) error {
	if len(paths) == 0 {
		return j.deletePath(ctx, key, "$")
	}

	err := lo.Reduce(paths, func(result error, path string, _ int) error {
		if result != nil {
			return result
		}
		return j.deletePath(ctx, key, path)
	}, error(nil))
	if err != nil {
		return fmt.Errorf("delete JSON paths from %s: %w", key, err)
	}
	return nil
}

// Exists checks if a JSON document exists.
func (j *JSON) Exists(ctx context.Context, key string) (bool, error) {
	data, err := j.getDocumentData(ctx, key)
	if err != nil {
		if errors.Is(err, kvx.ErrNil) {
			return false, nil
		}
		return false, err
	}
	return len(data) > 0, nil
}

// Type gets the type of a JSON value at a path.
func (j *JSON) Type(ctx context.Context, key, path string) (string, error) {
	// This would require FT.TYPE or similar command
	// For now, we'll try to get the value and infer the type
	data, err := j.getPathData(ctx, key, path)
	if err != nil {
		return "", err
	}

	var v any
	if err := unmarshalJSONValue(data, &v, fmt.Sprintf("unmarshal JSON path %q from %q", path, key)); err != nil {
		return "", err
	}

	switch v.(type) {
	case map[string]any:
		return "object", nil
	case []any:
		return "array", nil
	case string:
		return "string", nil
	case float64:
		return "number", nil
	case bool:
		return "boolean", nil
	case nil:
		return "null", nil
	default:
		return "unknown", nil
	}
}

// Length gets the length of an array or object at a path.
func (j *JSON) Length(ctx context.Context, key, path string) (int, error) {
	// Get the value and calculate length
	data, err := j.getPathData(ctx, key, path)
	if err != nil {
		return 0, err
	}

	var v any
	if err := unmarshalJSONValue(data, &v, fmt.Sprintf("unmarshal JSON path %q from %q", path, key)); err != nil {
		return 0, err
	}

	switch val := v.(type) {
	case map[string]any:
		return len(val), nil
	case []any:
		return len(val), nil
	case string:
		return len(val), nil
	default:
		return 0, fmt.Errorf("value at path %s does not have a length", path)
	}
}

// ArrayAppend appends values to an array at a path.
func (j *JSON) ArrayAppend(_ context.Context, _, _ string, values ...any) error {
	_, err := lo.ReduceErr(values, func(_ struct{}, value any, _ int) (struct{}, error) {
		_, err := marshalJSONValue("marshal JSON array value", value)
		return struct{}{}, err
	}, struct{}{})
	if err != nil {
		return fmt.Errorf("marshal JSON array values: %w", err)
	}
	return errors.New("ArrayAppend requires adapter support for JSON.ARRAPPEND")
}

// ArrayIndex gets the index of a value in an array.
func (j *JSON) ArrayIndex(ctx context.Context, key, path string, value any) (int, error) {
	data, err := j.getPathData(ctx, key, path)
	if err != nil {
		return -1, err
	}

	var arr []any
	if decodeErr := unmarshalJSONValue(data, &arr, fmt.Sprintf("unmarshal JSON array %q from %q", path, key)); decodeErr != nil {
		return -1, decodeErr
	}

	valueData, err := marshalJSONValue("marshal JSON array lookup value", value)
	if err != nil {
		return -1, err
	}

	for i, item := range arr {
		itemData, marshalErr := marshalJSONValue("marshal JSON array item", item)
		if marshalErr != nil {
			return -1, marshalErr
		}
		if bytes.Equal(itemData, valueData) {
			return i, nil
		}
	}

	return -1, errors.New("value not found in array")
}

// ArrayPop removes and returns the last element of an array.
func (j *JSON) ArrayPop(ctx context.Context, key, path string) (any, error) {
	data, err := j.getPathData(ctx, key, path)
	if err != nil {
		return nil, err
	}

	var arr []any
	if decodeErr := unmarshalJSONValue(data, &arr, fmt.Sprintf("unmarshal JSON array %q from %q", path, key)); decodeErr != nil {
		return nil, decodeErr
	}

	if len(arr) == 0 {
		return nil, errors.New("array is empty")
	}

	last := arr[len(arr)-1]
	arr = arr[:len(arr)-1]

	// Set the modified array back
	newData, err := marshalJSONValue("marshal JSON array", arr)
	if err != nil {
		return nil, err
	}
	if err := j.setPathData(ctx, key, path, newData); err != nil {
		return nil, err
	}

	return last, nil
}

// ObjectKeys gets the keys of an object at a path.
func (j *JSON) ObjectKeys(ctx context.Context, key, path string) ([]string, error) {
	data, err := j.getPathData(ctx, key, path)
	if err != nil {
		return nil, err
	}

	var obj map[string]any
	if err := unmarshalJSONValue(data, &obj, fmt.Sprintf("unmarshal JSON object %q from %q", path, key)); err != nil {
		return nil, err
	}

	return lo.Keys(obj), nil
}

// ObjectMerge merges multiple objects into the target object.
func (j *JSON) ObjectMerge(ctx context.Context, key, path string, objects ...map[string]any) error {
	// Get current object
	data, err := j.getPathData(ctx, key, path)
	if err != nil {
		return fmt.Errorf("marshal JSON array values: %w", err)
	}

	var target map[string]any
	if len(data) > 0 {
		if decodeErr := unmarshalJSONValue(data, &target, fmt.Sprintf("unmarshal JSON object %q from %q", path, key)); decodeErr != nil {
			return decodeErr
		}
	} else {
		target = make(map[string]any)
	}

	// Merge all objects
	lo.ForEach(objects, func(obj map[string]any, _ int) {
		maps.Copy(target, obj)
	})

	// Set back
	newData, err := marshalJSONValue("marshal JSON object", target)
	if err != nil {
		return fmt.Errorf("marshal JSON array values: %w", err)
	}
	return j.setPathData(ctx, key, path, newData)
}

// MultiGet gets multiple JSON documents by keys.
func (j *JSON) MultiGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	return lo.Reduce(keys, func(results map[string][]byte, key string, _ int) map[string][]byte {
		data, err := j.getDocumentData(ctx, key)
		if err == nil {
			results[key] = data
		}
		return results
	}, make(map[string][]byte, len(keys))), nil
}
