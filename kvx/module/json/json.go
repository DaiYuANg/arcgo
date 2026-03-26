// Package json provides JSON document operations.
package json

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
func (j *JSON) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return j.client.JSONSet(ctx, key, "$", data, expiration)
}

// SetPath sets a JSON value at a specific path.
func (j *JSON) SetPath(ctx context.Context, key, path string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return j.client.JSONSetField(ctx, key, path, data)
}

// Get gets a JSON document by key.
func (j *JSON) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := j.client.JSONGet(ctx, key, "$")
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("document not found: %s", key)
	}
	return json.Unmarshal(data, dest)
}

// GetPath gets a JSON value at a specific path.
func (j *JSON) GetPath(ctx context.Context, key, path string, dest interface{}) error {
	data, err := j.client.JSONGetField(ctx, key, path)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("path not found: %s.%s", key, path)
	}
	return json.Unmarshal(data, dest)
}

// Delete deletes a JSON document or a path within it.
func (j *JSON) Delete(ctx context.Context, key string, paths ...string) error {
	if len(paths) == 0 {
		return j.client.JSONDelete(ctx, key, "$")
	}
	for _, path := range paths {
		if err := j.client.JSONDelete(ctx, key, path); err != nil {
			return err
		}
	}
	return nil
}

// Exists checks if a JSON document exists.
func (j *JSON) Exists(ctx context.Context, key string) (bool, error) {
	data, err := j.client.JSONGet(ctx, key, "$")
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
	data, err := j.client.JSONGetField(ctx, key, path)
	if err != nil {
		return "", err
	}

	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return "", err
	}

	switch v.(type) {
	case map[string]interface{}:
		return "object", nil
	case []interface{}:
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
	data, err := j.client.JSONGetField(ctx, key, path)
	if err != nil {
		return 0, err
	}

	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return 0, err
	}

	switch val := v.(type) {
	case map[string]interface{}:
		return len(val), nil
	case []interface{}:
		return len(val), nil
	case string:
		return len(val), nil
	default:
		return 0, fmt.Errorf("value at path %s does not have a length", path)
	}
}

// ArrayAppend appends values to an array at a path.
func (j *JSON) ArrayAppend(ctx context.Context, key, path string, values ...interface{}) error {
	// Build the JSON.ARRAPPEND command
	// This is a simplified version - full implementation would need adapter support
	for _, value := range values {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		// Try to append by getting current array, appending, and setting back
		// This is not atomic - full implementation should use JSON.ARRAPPEND
		_ = data
	}
	return fmt.Errorf("ArrayAppend requires adapter support for JSON.ARRAPPEND")
}

// ArrayIndex gets the index of a value in an array.
func (j *JSON) ArrayIndex(ctx context.Context, key, path string, value interface{}) (int, error) {
	data, err := j.client.JSONGetField(ctx, key, path)
	if err != nil {
		return -1, err
	}

	var arr []interface{}
	if err := json.Unmarshal(data, &arr); err != nil {
		return -1, err
	}

	valueData, err := json.Marshal(value)
	if err != nil {
		return -1, err
	}

	for i, item := range arr {
		itemData, _ := json.Marshal(item)
		if string(itemData) == string(valueData) {
			return i, nil
		}
	}

	return -1, fmt.Errorf("value not found in array")
}

// ArrayPop removes and returns the last element of an array.
func (j *JSON) ArrayPop(ctx context.Context, key, path string) (interface{}, error) {
	data, err := j.client.JSONGetField(ctx, key, path)
	if err != nil {
		return nil, err
	}

	var arr []interface{}
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, err
	}

	if len(arr) == 0 {
		return nil, fmt.Errorf("array is empty")
	}

	last := arr[len(arr)-1]
	arr = arr[:len(arr)-1]

	// Set the modified array back
	newData, _ := json.Marshal(arr)
	if err := j.client.JSONSetField(ctx, key, path, newData); err != nil {
		return nil, err
	}

	return last, nil
}

// ObjectKeys gets the keys of an object at a path.
func (j *JSON) ObjectKeys(ctx context.Context, key, path string) ([]string, error) {
	data, err := j.client.JSONGetField(ctx, key, path)
	if err != nil {
		return nil, err
	}

	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}

	return lo.Keys(obj), nil
}

// ObjectMerge merges multiple objects into the target object.
func (j *JSON) ObjectMerge(ctx context.Context, key, path string, objects ...map[string]interface{}) error {
	// Get current object
	data, err := j.client.JSONGetField(ctx, key, path)
	if err != nil {
		return err
	}

	var target map[string]interface{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &target); err != nil {
			return err
		}
	} else {
		target = make(map[string]interface{})
	}

	// Merge all objects
	for _, obj := range objects {
		for k, v := range obj {
			target[k] = v
		}
	}

	// Set back
	newData, _ := json.Marshal(target)
	return j.client.JSONSetField(ctx, key, path, newData)
}

// MultiGet gets multiple JSON documents by keys.
func (j *JSON) MultiGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	results := make(map[string][]byte, len(keys))
	for _, key := range keys {
		data, err := j.client.JSONGet(ctx, key, "$")
		if err != nil {
			continue
		}
		results[key] = data
	}
	return results, nil
}

// DocumentRepository provides repository-style access to JSON documents.
type DocumentRepository[T any] struct {
	json      *JSON
	keyPrefix string
}

// NewDocumentRepository creates a new DocumentRepository.
func NewDocumentRepository[T any](client kvx.JSON, keyPrefix string) *DocumentRepository[T] {
	return &DocumentRepository[T]{
		json:      NewJSON(client),
		keyPrefix: keyPrefix,
	}
}

// buildKey builds the full key for a document.
func (r *DocumentRepository[T]) buildKey(id string) string {
	if r.keyPrefix == "" {
		return id
	}
	return fmt.Sprintf("%s:%s", r.keyPrefix, id)
}

// Save saves a document.
func (r *DocumentRepository[T]) Save(ctx context.Context, id string, doc *T, expiration time.Duration) error {
	key := r.buildKey(id)
	return r.json.Set(ctx, key, doc, expiration)
}

// FindByID finds a document by ID.
func (r *DocumentRepository[T]) FindByID(ctx context.Context, id string) (*T, error) {
	key := r.buildKey(id)
	var doc T
	if err := r.json.Get(ctx, key, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// Delete deletes a document.
func (r *DocumentRepository[T]) Delete(ctx context.Context, id string) error {
	key := r.buildKey(id)
	return r.json.Delete(ctx, key)
}

// Exists checks if a document exists.
func (r *DocumentRepository[T]) Exists(ctx context.Context, id string) (bool, error) {
	key := r.buildKey(id)
	return r.json.Exists(ctx, key)
}

// UpdatePath updates a specific path in a document.
func (r *DocumentRepository[T]) UpdatePath(ctx context.Context, id, path string, value interface{}) error {
	key := r.buildKey(id)
	return r.json.SetPath(ctx, key, path, value)
}

// GetPath gets a specific path from a document.
func (r *DocumentRepository[T]) GetPath(ctx context.Context, id, path string, dest interface{}) error {
	key := r.buildKey(id)
	return r.json.GetPath(ctx, key, path, dest)
}
