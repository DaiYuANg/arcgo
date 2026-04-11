// Package json provides JSON document operations.
//
//revive:disable:file-length-limit JSON module operations are kept together as one cohesive API surface.
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
	"github.com/samber/oops"
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
	data, err := marshalJSONValue("set", value)
	if err != nil {
		return oops.In("kvx/module/json").
			With("op", "set", "key", key, "path", "$", "expiration", expiration).
			Wrapf(err, "marshal json document")
	}
	return j.setDocumentData(ctx, key, data, expiration)
}

// SetPath sets a JSON value at a specific path.
func (j *JSON) SetPath(ctx context.Context, key, path string, value any) error {
	data, err := marshalJSONValue("set_path", value)
	if err != nil {
		return oops.In("kvx/module/json").
			With("op", "set_path", "key", key, "path", path).
			Wrapf(err, "marshal json path value")
	}
	return j.setPathData(ctx, key, path, data)
}

// Get gets a JSON document by key.
func (j *JSON) Get(ctx context.Context, key string, dest any) error {
	data, err := j.getDocumentData(ctx, key)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return oops.In("kvx/module/json").
			With("op", "get", "key", key, "path", "$").
			Wrapf(kvx.ErrNil, "document not found")
	}
	if err := unmarshalJSONValue(data, dest, "get"); err != nil {
		return oops.In("kvx/module/json").
			With("op", "get", "key", key, "path", "$").
			Wrapf(err, "unmarshal json document")
	}
	return nil
}

// GetPath gets a JSON value at a specific path.
func (j *JSON) GetPath(ctx context.Context, key, path string, dest any) error {
	data, err := j.getPathData(ctx, key, path)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return oops.In("kvx/module/json").
			With("op", "get_path", "key", key, "path", path).
			Wrapf(kvx.ErrNil, "path not found")
	}
	if err := unmarshalJSONValue(data, dest, "get_path"); err != nil {
		return oops.In("kvx/module/json").
			With("op", "get_path", "key", key, "path", path).
			Wrapf(err, "unmarshal json path")
	}
	return nil
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
		return oops.In("kvx/module/json").
			With("op", "delete", "key", key, "path_count", len(paths)).
			Wrapf(err, "delete json paths")
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
	if err := unmarshalJSONValue(data, &v, "type"); err != nil {
		return "", oops.In("kvx/module/json").
			With("op", "type", "key", key, "path", path).
			Wrapf(err, "unmarshal json path")
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
	if err := unmarshalJSONValue(data, &v, "length"); err != nil {
		return 0, oops.In("kvx/module/json").
			With("op", "length", "key", key, "path", path).
			Wrapf(err, "unmarshal json path")
	}

	switch val := v.(type) {
	case map[string]any:
		return len(val), nil
	case []any:
		return len(val), nil
	case string:
		return len(val), nil
	default:
		return 0, oops.In("kvx/module/json").
			With("op", "length", "key", key, "path", path, "value_type", fmt.Sprintf("%T", v)).
			Errorf("value at path %s does not have a length", path)
	}
}

// ArrayAppend appends values to an array at a path.
func (j *JSON) ArrayAppend(_ context.Context, key, path string, values ...any) error {
	_, err := lo.ReduceErr(values, func(_ struct{}, value any, _ int) (struct{}, error) {
		_, err := marshalJSONValue("array_append", value)
		return struct{}{}, err
	}, struct{}{})
	if err != nil {
		return oops.In("kvx/module/json").
			With("op", "array_append", "key", key, "path", path, "value_count", len(values)).
			Wrapf(err, "marshal json array values")
	}
	return oops.In("kvx/module/json").
		With("op", "array_append", "key", key, "path", path, "value_count", len(values)).
		Wrapf(kvx.ErrUnsupportedOption, "JSON.ARRAPPEND requires adapter support")
}

// ArrayIndex gets the index of a value in an array.
func (j *JSON) ArrayIndex(ctx context.Context, key, path string, value any) (int, error) {
	data, err := j.getPathData(ctx, key, path)
	if err != nil {
		return -1, err
	}

	var arr []any
	if decodeErr := unmarshalJSONValue(data, &arr, "array_index"); decodeErr != nil {
		return -1, oops.In("kvx/module/json").
			With("op", "array_index", "key", key, "path", path).
			Wrapf(decodeErr, "unmarshal json array")
	}

	valueData, err := marshalJSONValue("array_index", value)
	if err != nil {
		return -1, oops.In("kvx/module/json").
			With("op", "array_index", "key", key, "path", path).
			Wrapf(err, "marshal lookup value")
	}

	for i, item := range arr {
		itemData, marshalErr := marshalJSONValue("array_index", item)
		if marshalErr != nil {
			return -1, oops.In("kvx/module/json").
				With("op", "array_index", "key", key, "path", path, "index", i).
				Wrapf(marshalErr, "marshal array item")
		}
		if bytes.Equal(itemData, valueData) {
			return i, nil
		}
	}

	return -1, oops.In("kvx/module/json").
		With("op", "array_index", "key", key, "path", path).
		New("value not found in array")
}

// ArrayPop removes and returns the last element of an array.
func (j *JSON) ArrayPop(ctx context.Context, key, path string) (any, error) {
	data, err := j.getPathData(ctx, key, path)
	if err != nil {
		return nil, err
	}

	var arr []any
	if decodeErr := unmarshalJSONValue(data, &arr, "array_pop"); decodeErr != nil {
		return nil, oops.In("kvx/module/json").
			With("op", "array_pop", "key", key, "path", path).
			Wrapf(decodeErr, "unmarshal json array")
	}

	if len(arr) == 0 {
		return nil, oops.In("kvx/module/json").
			With("op", "array_pop", "key", key, "path", path).
			New("array is empty")
	}

	last := arr[len(arr)-1]
	arr = arr[:len(arr)-1]

	// Set the modified array back
	newData, err := marshalJSONValue("array_pop", arr)
	if err != nil {
		return nil, oops.In("kvx/module/json").
			With("op", "array_pop", "key", key, "path", path).
			Wrapf(err, "marshal json array")
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
	if err := unmarshalJSONValue(data, &obj, "object_keys"); err != nil {
		return nil, oops.In("kvx/module/json").
			With("op", "object_keys", "key", key, "path", path).
			Wrapf(err, "unmarshal json object")
	}

	return lo.Keys(obj), nil
}

// ObjectMerge merges multiple objects into the target object.
func (j *JSON) ObjectMerge(ctx context.Context, key, path string, objects ...map[string]any) error {
	// Get current object
	data, err := j.getPathData(ctx, key, path)
	if err != nil {
		return err
	}

	var target map[string]any
	if len(data) > 0 {
		if decodeErr := unmarshalJSONValue(data, &target, "object_merge"); decodeErr != nil {
			return oops.In("kvx/module/json").
				With("op", "object_merge", "key", key, "path", path).
				Wrapf(decodeErr, "unmarshal json object")
		}
	} else {
		target = make(map[string]any)
	}

	// Merge all objects
	lo.ForEach(objects, func(obj map[string]any, _ int) {
		maps.Copy(target, obj)
	})

	// Set back
	newData, err := marshalJSONValue("object_merge", target)
	if err != nil {
		return oops.In("kvx/module/json").
			With("op", "object_merge", "key", key, "path", path, "object_count", len(objects)).
			Wrapf(err, "marshal json object")
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
