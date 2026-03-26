package repository

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx/set"
	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/DaiYuANg/arcgo/kvx/mapping"
	"github.com/samber/lo"
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
	v := reflect.Indirect(reflect.ValueOf(entity))
	for _, fieldName := range metadata.IndexFields {
		fieldTag, ok := metadata.Fields[fieldName]
		if !ok {
			continue
		}
		fieldVal := v.FieldByName(fieldName)
		if !fieldVal.IsValid() {
			continue
		}
		fieldValue := formatIndexValue(fieldVal)
		if fieldValue == "" {
			continue
		}
		indexKey := i.buildIndexKey(fieldTag.IndexNameOrDefault(), fieldValue)
		if err := i.addToIndex(ctx, indexKey, extractIDFromKey(entityKey)); err != nil {
			return fmt.Errorf("failed to index field %s: %w", fieldName, err)
		}
	}
	return nil
}

// RemoveEntityFromIndexes removes an entity from all secondary indexes.
func (i *Indexer[T]) RemoveEntityFromIndexes(ctx context.Context, entity *T, metadata *mapping.EntityMetadata) error {
	v := reflect.Indirect(reflect.ValueOf(entity))
	entityKey, err := i.keyBuilder.Build(entity, metadata)
	if err != nil {
		return err
	}
	entityID := extractIDFromKey(entityKey)

	for _, fieldName := range metadata.IndexFields {
		fieldTag, ok := metadata.Fields[fieldName]
		if !ok {
			continue
		}
		fieldVal := v.FieldByName(fieldName)
		if !fieldVal.IsValid() {
			continue
		}
		fieldValue := formatIndexValue(fieldVal)
		if fieldValue == "" {
			continue
		}
		indexKey := i.buildIndexKey(fieldTag.IndexNameOrDefault(), fieldValue)
		if err := i.removeFromIndex(ctx, indexKey, entityID); err != nil {
			return fmt.Errorf("failed to remove index for field %s: %w", fieldName, err)
		}
	}
	return nil
}

// EntityIndexEntries returns all concrete index entry keys for an entity.
func (i *Indexer[T]) EntityIndexEntries(entity *T, metadata *mapping.EntityMetadata, entityKey string) ([]string, error) {
	if entity == nil {
		return nil, nil
	}
	v := reflect.Indirect(reflect.ValueOf(entity))
	entityID := extractIDFromKey(entityKey)
	results := make([]string, 0, len(metadata.IndexFields))
	for _, fieldName := range metadata.IndexFields {
		fieldTag, ok := metadata.Fields[fieldName]
		if !ok {
			continue
		}
		fieldVal := v.FieldByName(fieldName)
		if !fieldVal.IsValid() {
			continue
		}
		if entry := i.indexEntryKey(fieldTag.IndexNameOrDefault(), formatIndexValue(fieldVal), entityID); entry != "" {
			results = append(results, entry)
		}
	}
	return results, nil
}

// ReplaceEntityIndexEntries calculates the index diff for replacing one entity with another.
func (i *Indexer[T]) ReplaceEntityIndexEntries(ctx context.Context, oldEntity *T, newEntity *T, metadata *mapping.EntityMetadata, entityKey string) ([]string, []string, error) {
	_ = ctx
	oldEntries, err := i.EntityIndexEntries(oldEntity, metadata, entityKey)
	if err != nil {
		return nil, nil, err
	}
	newEntries, err := i.EntityIndexEntries(newEntity, metadata, entityKey)
	if err != nil {
		return nil, nil, err
	}
	return diffIndexEntries(oldEntries, newEntries), diffIndexEntries(newEntries, oldEntries), nil
}

// ReplaceFieldIndexEntries calculates the index diff for updating one indexed field.
func (i *Indexer[T]) ReplaceFieldIndexEntries(metadata *mapping.EntityMetadata, fieldName string, entityKey string, entity *T, newValue interface{}) ([]string, []string, error) {
	resolvedField, fieldTag, exists := metadata.ResolveField(fieldName)
	if !exists || !fieldTag.Index || entity == nil {
		return nil, nil, nil
	}
	v := reflect.Indirect(reflect.ValueOf(entity))
	fieldVal := v.FieldByName(resolvedField)
	if !fieldVal.IsValid() {
		return nil, nil, nil
	}

	entityID := extractIDFromKey(entityKey)
	oldEntry := i.indexEntryKey(fieldTag.IndexNameOrDefault(), formatIndexValue(fieldVal), entityID)
	newEntry := i.indexEntryKey(fieldTag.IndexNameOrDefault(), formatIndexValue(reflect.ValueOf(newValue)), entityID)
	if oldEntry == newEntry {
		return nil, nil, nil
	}

	oldEntries := make([]string, 0, 1)
	newEntries := make([]string, 0, 1)
	if oldEntry != "" {
		oldEntries = append(oldEntries, oldEntry)
	}
	if newEntry != "" {
		newEntries = append(newEntries, newEntry)
	}
	return oldEntries, newEntries, nil
}

// ApplyIndexDiff removes stale index entries and writes the new ones.
func (i *Indexer[T]) ApplyIndexDiff(ctx context.Context, removeEntries []string, addEntries []string) error {
	for _, entry := range removeEntries {
		if err := i.kv.Delete(ctx, entry); err != nil {
			return err
		}
	}
	for _, entry := range addEntries {
		if err := i.kv.Set(ctx, entry, []byte("1"), 0); err != nil {
			return err
		}
	}
	return nil
}

// GetEntityIDsByField returns entity IDs that have the specified field value.
func (i *Indexer[T]) GetEntityIDsByField(ctx context.Context, fieldName string, fieldValue string) ([]string, error) {
	return i.getIndexMembers(ctx, i.buildIndexKey(fieldName, fieldValue))
}

func (i *Indexer[T]) buildIndexKey(fieldName, fieldValue string) string {
	prefix := strings.TrimSuffix(i.keyBuilder.BuildWithID(""), ":")
	if prefix == "" {
		return fmt.Sprintf("idx:%s:%s", fieldName, fieldValue)
	}
	return fmt.Sprintf("%s:idx:%s:%s", prefix, fieldName, fieldValue)
}

func (i *Indexer[T]) indexEntryKey(fieldName, fieldValue string, entityID string) string {
	if fieldValue == "" || entityID == "" {
		return ""
	}
	return fmt.Sprintf("%s:%s", i.buildIndexKey(fieldName, fieldValue), entityID)
}

func (i *Indexer[T]) addToIndex(ctx context.Context, indexKey string, entityID string) error {
	return i.kv.Set(ctx, fmt.Sprintf("%s:%s", indexKey, entityID), []byte("1"), 0)
}

func (i *Indexer[T]) removeFromIndex(ctx context.Context, indexKey string, entityID string) error {
	return i.kv.Delete(ctx, fmt.Sprintf("%s:%s", indexKey, entityID))
}

func (i *Indexer[T]) getIndexMembers(ctx context.Context, indexKey string) ([]string, error) {
	keys, err := i.kv.Keys(ctx, indexKey+":*")
	if err != nil {
		return nil, err
	}
	results := make([]string, 0, len(keys))
	prefixLen := len(indexKey) + 1
	for _, key := range keys {
		if len(key) > prefixLen {
			results = append(results, key[prefixLen:])
		}
	}
	return results, nil
}

func formatIndexValue(v reflect.Value) string {
	if !v.IsValid() {
		return ""
	}
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
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

func diffIndexEntries(left []string, right []string) []string {
	if len(left) == 0 {
		return nil
	}
	rightSet := set.NewSet[string](right...)
	return lo.Filter(left, func(entry string, _ int) bool {
		return !rightSet.Contains(entry)
	})
}

func extractIDFromKey(key string) string {
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == ':' {
			return key[i+1:]
		}
	}
	return key
}
