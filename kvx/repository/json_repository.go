package repository

import (
	"context"
	"errors"
	"time"

	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/DaiYuANg/arcgo/kvx/mapping"
	"github.com/samber/mo"
)

// JSONRepository provides repository operations for JSON-based entities.
type JSONRepository[T any] struct {
	base       repositoryBase[T]
	client     kvx.JSON
	kv         kvx.KV
	pipeline   mo.Option[pipelineProvider]
	script     mo.Option[kvx.Script]
	serializer mapping.Serializer
}

func NewJSONRepository[T any](client kvx.JSON, kv kvx.KV, keyPrefix string, options ...JSONRepositoryOption[T]) *JSONRepository[T] {
	cfg := defaultJSONConfig[T](kv, keyPrefix)
	applyJSONOptions(&cfg, options...)
	repo := &JSONRepository[T]{
		base:       repositoryBase[T]{keyBuilder: cfg.keyBuilder, tagParser: cfg.tagParser, indexer: cfg.indexer},
		client:     client,
		kv:         kv,
		pipeline:   cfg.pipeline,
		script:     cfg.script,
		serializer: cfg.serializer,
	}
	return repo
}

func NewJSONRepositoryWithClient[T any](client kvx.Client, keyPrefix string, options ...JSONRepositoryOption[T]) *JSONRepository[T] {
	options = append([]JSONRepositoryOption[T]{WithPipeline[T](client), WithScript[T](client)}, options...)
	return NewJSONRepository[T](client, client, keyPrefix, options...)
}

func (r *JSONRepository[T]) Save(ctx context.Context, entity *T) error {
	return r.SaveWithExpiration(ctx, entity, 0)
}

func (r *JSONRepository[T]) SaveWithExpiration(ctx context.Context, entity *T, expiration time.Duration) error {
	metadata, err := r.base.metadata(entity)
	if err != nil {
		return err
	}
	key, err := r.base.keyBuilder.Build(entity, metadata)
	if err != nil {
		return err
	}
	data, err := r.serializer.Marshal(entity)
	if err != nil {
		return err
	}
	previous, err := r.findByKey(ctx, key)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	if errors.Is(err, ErrNotFound) {
		previous = nil
	}

	oldEntries, newEntries, err := r.base.indexer.ReplaceEntityIndexEntries(ctx, previous, entity, metadata, key)
	if err != nil {
		return err
	}

	if script, ok := r.script.Get(); ok {
		return execJSONUpsertScript(ctx, script, key, data, expiration, oldEntries, newEntries)
	}

	if err := r.client.JSONSet(ctx, key, "$", data, expiration); err != nil {
		return err
	}
	if err := r.base.indexer.ApplyIndexDiff(ctx, oldEntries, newEntries); err != nil {
		return err
	}
	return nil
}

func (r *JSONRepository[T]) SaveBatch(ctx context.Context, entities []*T) error {
	return r.SaveBatchWithExpiration(ctx, entities, 0)
}

func (r *JSONRepository[T]) SaveBatchWithExpiration(ctx context.Context, entities []*T, expiration time.Duration) error {
	if len(entities) == 0 {
		return nil
	}
	for _, entity := range entities {
		if err := r.SaveWithExpiration(ctx, entity, expiration); err != nil {
			return err
		}
	}
	return nil
}

func (r *JSONRepository[T]) FindByID(ctx context.Context, id string) (*T, error) {
	return r.findByKey(ctx, r.base.keyFromID(id))
}

func (r *JSONRepository[T]) FindByIDs(ctx context.Context, ids []string) (map[string]*T, error) {
	return collectPresentMap(ids, func(id string) (*T, error) {
		return r.FindByID(ctx, id)
	})
}

func (r *JSONRepository[T]) Exists(ctx context.Context, id string) (bool, error) {
	return r.kv.Exists(ctx, r.base.keyFromID(id))
}

func (r *JSONRepository[T]) ExistsBatch(ctx context.Context, ids []string) (map[string]bool, error) {
	keys := r.base.keysFromIDs(ids)
	existsMap, err := r.kv.ExistsMulti(ctx, keys)
	if err != nil {
		return nil, err
	}
	return mapExistsResults(ids, keys, existsMap), nil
}

func (r *JSONRepository[T]) Delete(ctx context.Context, id string) error {
	key := r.base.keyFromID(id)
	entity, err := r.findByKey(ctx, key)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	}
	metadata, err := r.base.metadata(entity)
	if err != nil {
		return err
	}
	oldEntries, err := r.base.indexer.EntityIndexEntries(entity, metadata, key)
	if err != nil {
		return err
	}
	if script, ok := r.script.Get(); ok {
		return execDeleteScript(ctx, script, key, oldEntries)
	}
	if err := r.client.JSONDelete(ctx, key, "$"); err != nil {
		return err
	}
	return r.base.indexer.ApplyIndexDiff(ctx, oldEntries, nil)
}

func (r *JSONRepository[T]) DeleteBatch(ctx context.Context, ids []string) error {
	return runAll(ids, func(id string) error {
		return r.Delete(ctx, id)
	})
}

func (r *JSONRepository[T]) UpdateField(ctx context.Context, id string, fieldPath string, value interface{}) error {
	key := r.base.keyFromID(id)
	entity, err := r.findByKey(ctx, key)
	if err != nil {
		return err
	}
	metadata, err := r.base.metadata(entity)
	if err != nil {
		return err
	}
	fieldName := extractFieldNameFromPath(fieldPath)
	resolvedField, fieldTag, exists := metadata.ResolveField(fieldName)
	data, err := r.serializer.Marshal(value)
	if err != nil {
		return err
	}
	oldEntries := []string(nil)
	newEntries := []string(nil)
	if exists && fieldTag.Index {
		oldEntries, newEntries, err = r.base.indexer.ReplaceFieldIndexEntries(metadata, resolvedField, key, entity, value)
		if err != nil {
			return err
		}
	}
	if script, ok := r.script.Get(); ok {
		return execJSONFieldUpdateScript(ctx, script, key, fieldPath, data, oldEntries, newEntries)
	}
	if err := r.client.JSONSetField(ctx, key, fieldPath, data); err != nil {
		return err
	}
	return r.base.indexer.ApplyIndexDiff(ctx, oldEntries, newEntries)
}

func (r *JSONRepository[T]) FindByField(ctx context.Context, fieldName string, fieldValue string) ([]*T, error) {
	ids, err := r.base.idsByField(ctx, fieldName, fieldValue)
	if err != nil {
		return nil, err
	}
	return r.findManyByIDs(ctx, ids)
}

func (r *JSONRepository[T]) FindByFields(ctx context.Context, fields map[string]string) ([]*T, error) {
	if len(fields) == 0 {
		return r.FindAll(ctx)
	}
	idGroups := make([][]string, 0, len(fields))
	for fieldName, fieldValue := range fields {
		ids, err := r.base.idsByField(ctx, fieldName, fieldValue)
		if err != nil {
			return nil, err
		}
		idGroups = append(idGroups, ids)
	}
	intersection := intersectStringSlices(idGroups...)
	if len(intersection) == 0 {
		return []*T{}, nil
	}
	return r.findManyByIDs(ctx, intersection)
}

func (r *JSONRepository[T]) FindAll(ctx context.Context) ([]*T, error) {
	keys, err := r.base.scanAllKeys(ctx, r.kv)
	if err != nil {
		return nil, err
	}
	return collectPresentSlice(keys, func(key string) (*T, error) {
		return r.findByKey(ctx, key)
	})
}

func (r *JSONRepository[T]) Count(ctx context.Context) (int64, error) {
	keys, err := r.base.scanAllKeys(ctx, r.kv)
	if err != nil {
		return 0, err
	}
	return int64(len(keys)), nil
}

func (r *JSONRepository[T]) findByKey(ctx context.Context, key string) (*T, error) {
	data, err := r.client.JSONGet(ctx, key, "$")
	if err != nil {
		if kvx.IsNil(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if len(data) == 0 {
		return nil, ErrNotFound
	}
	var entity T
	if err := r.serializer.Unmarshal(data, &entity); err != nil {
		return nil, err
	}
	metadata, err := r.base.metadataForType()
	if err != nil {
		return nil, err
	}
	if err := r.base.hydrateEntityID(&entity, metadata, key); err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *JSONRepository[T]) findManyByIDs(ctx context.Context, ids []string) ([]*T, error) {
	return collectPresentSlice(ids, func(id string) (*T, error) {
		return r.FindByID(ctx, id)
	})
}

func extractFieldNameFromPath(path string) string {
	if len(path) > 2 && path[:2] == "$." {
		return path[2:]
	}
	return path
}
