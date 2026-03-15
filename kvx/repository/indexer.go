package repository

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/DaiYuANg/archgo/kvx"
	"github.com/DaiYuANg/archgo/kvx/mapping"
)

// Indexer manages secondary indexes for entities.
type Indexer[T any] struct {
	kv         kvx.KV
	keyBuilder *mapping.KeyBuilder
}

// NewIndexer creates a new Indexer.
func NewIndexer[T any](kv kvx.KV, keyPrefix string) *Indexer[T] {
	return &Indexer[T]{
		kv:         kv,
		keyBuilder: mapping.NewKeyBuilder(keyPrefix),
	}
}

// IndexEntity adds an entity to secondary indexes.
func (i *Indexer[T]) IndexEntity(ctx context.Context, entity *T, metadata *mapping.EntityMetadata, entityKey string) error {
	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	for _, fieldName := range metadata.IndexFields {
		fieldTag := metadata.Fields[fieldName]
		fieldVal := v.FieldByName(fieldName)
		if !fieldVal.IsValid() {
			continue
		}

		fieldValue := formatIndexValue(fieldVal)
		if fieldValue == "" {
			continue
		}

		indexKey := i.buildIndexKey(fieldTag.Name, fieldValue)

		// Add entity ID to the index set
		entityID := extractIDFromKey(entityKey)
		if err := i.addToIndex(ctx, indexKey, entityID); err != nil {
			return fmt.Errorf("failed to index field %s: %w", fieldName, err)
		}
	}

	return nil
}

// RemoveEntityFromIndexes removes an entity from all secondary indexes.
func (i *Indexer[T]) RemoveEntityFromIndexes(ctx context.Context, entity *T, metadata *mapping.EntityMetadata) error {
	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	for _, fieldName := range metadata.IndexFields {
		fieldTag := metadata.Fields[fieldName]
		fieldVal := v.FieldByName(fieldName)
		if !fieldVal.IsValid() {
			continue
		}

		fieldValue := formatIndexValue(fieldVal)
		if fieldValue == "" {
			continue
		}

		indexKey := i.buildIndexKey(fieldTag.Name, fieldValue)

		// Get entity ID
		entityKey, _ := i.keyBuilder.Build(entity, metadata)
		entityID := extractIDFromKey(entityKey)

		if err := i.removeFromIndex(ctx, indexKey, entityID); err != nil {
			return fmt.Errorf("failed to remove index for field %s: %w", fieldName, err)
		}
	}

	return nil
}

// UpdateFieldIndex updates the index when a field value changes.
func (i *Indexer[T]) UpdateFieldIndex(ctx context.Context, entity *T, metadata *mapping.EntityMetadata, fieldName string, entityKey string) error {
	fieldTag, exists := metadata.Fields[fieldName]
	if !exists || !fieldTag.Index {
		return nil
	}

	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	fieldVal := v.FieldByName(fieldName)
	if !fieldVal.IsValid() {
		return nil
	}

	fieldValue := formatIndexValue(fieldVal)
	if fieldValue == "" {
		return nil
	}

	indexKey := i.buildIndexKey(fieldTag.Name, fieldValue)
	entityID := extractIDFromKey(entityKey)

	return i.addToIndex(ctx, indexKey, entityID)
}

// GetEntityIDsByField returns entity IDs that have the specified field value.
func (i *Indexer[T]) GetEntityIDsByField(ctx context.Context, fieldName string, fieldValue string) ([]string, error) {
	indexKey := i.buildIndexKey(fieldName, fieldValue)
	return i.getIndexMembers(ctx, indexKey)
}

// buildIndexKey builds the key for a secondary index.
func (i *Indexer[T]) buildIndexKey(fieldName, fieldValue string) string {
	prefix := i.keyBuilder.BuildWithID("")
	if prefix == "" {
		return fmt.Sprintf("idx:%s:%s", fieldName, fieldValue)
	}
	// Remove trailing colon if present
	prefix = strings.TrimSuffix(prefix, ":")
	return fmt.Sprintf("%s:idx:%s:%s", prefix, fieldName, fieldValue)
}

// addToIndex adds an entity ID to an index.
func (i *Indexer[T]) addToIndex(ctx context.Context, indexKey string, entityID string) error {
	// Use a set to store entity IDs for uniqueness
	// We use a simple approach with hash field
	return i.kv.Set(ctx, fmt.Sprintf("%s:%s", indexKey, entityID), []byte("1"), 0)
}

// removeFromIndex removes an entity ID from an index.
func (i *Indexer[T]) removeFromIndex(ctx context.Context, indexKey string, entityID string) error {
	return i.kv.Delete(ctx, fmt.Sprintf("%s:%s", indexKey, entityID))
}

// getIndexMembers retrieves all entity IDs from an index.
func (i *Indexer[T]) getIndexMembers(ctx context.Context, indexKey string) ([]string, error) {
	pattern := indexKey + ":*"
	keys, err := i.kv.Keys(ctx, pattern)
	if err != nil {
		return nil, err
	}

	// Extract entity IDs from keys
	results := make([]string, 0, len(keys))
	prefixLen := len(indexKey) + 1
	for _, key := range keys {
		if len(key) > prefixLen {
			results = append(results, key[prefixLen:])
		}
	}

	return results, nil
}

// formatIndexValue formats a reflect.Value for indexing.
func formatIndexValue(v reflect.Value) string {
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", v.Uint())
	case reflect.Bool:
		if v.Bool() {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

// extractIDFromKey extracts the ID from an entity key.
func extractIDFromKey(key string) string {
	// Find the last colon
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == ':' {
			return key[i+1:]
		}
	}
	return key
}
