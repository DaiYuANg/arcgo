package repository

import (
	"context"

	"github.com/DaiYuANg/arcgo/kvx"
)

// FindByID loads an entity by its logical ID.
func (r *JSONRepository[T]) FindByID(ctx context.Context, id string) (*T, error) {
	return r.findByKey(ctx, r.base.keyFromID(id))
}

// FindByIDs loads all entities that exist for the provided logical IDs.
func (r *JSONRepository[T]) FindByIDs(ctx context.Context, ids []string) (map[string]*T, error) {
	return collectPresentMap(ids, func(id string) (*T, error) {
		return r.FindByID(ctx, id)
	})
}

// Exists reports whether an entity exists for the provided logical ID.
func (r *JSONRepository[T]) Exists(ctx context.Context, id string) (bool, error) {
	exists, err := r.kv.Exists(ctx, r.base.keyFromID(id))
	return wrapRepositoryResult(exists, err, "check JSON entity existence")
}

// ExistsBatch reports entity existence for each provided logical ID.
func (r *JSONRepository[T]) ExistsBatch(ctx context.Context, ids []string) (map[string]bool, error) {
	keys := r.base.keysFromIDs(ids)
	existsMap, err := r.kv.ExistsMulti(ctx, keys)
	if err != nil {
		return nil, wrapRepositoryError(err, "check JSON entity existence in batch")
	}

	return mapExistsResults(ids, keys, existsMap), nil
}

// FindByField loads all entities whose indexed field matches the provided value.
func (r *JSONRepository[T]) FindByField(ctx context.Context, fieldName, fieldValue string) ([]*T, error) {
	ids, err := r.base.idsByField(ctx, fieldName, fieldValue)
	if err != nil {
		return nil, err
	}

	return r.findManyByIDs(ctx, ids)
}

// FindByFields loads all entities that match every provided indexed field value.
func (r *JSONRepository[T]) FindByFields(ctx context.Context, fields map[string]string) ([]*T, error) {
	if len(fields) == 0 {
		return r.FindAll(ctx)
	}

	idGroups, err := loadFieldIDGroups(fields, func(fieldName, fieldValue string) ([]string, error) {
		return r.base.idsByField(ctx, fieldName, fieldValue)
	})
	if err != nil {
		return nil, err
	}

	intersection := intersectStringSlices(idGroups...)
	if len(intersection) == 0 {
		return []*T{}, nil
	}

	return r.findManyByIDs(ctx, intersection)
}

// FindAll loads all entities stored under the repository key prefix.
func (r *JSONRepository[T]) FindAll(ctx context.Context) ([]*T, error) {
	keys, err := r.base.scanAllKeys(ctx, r.kv)
	if err != nil {
		return nil, err
	}

	return collectPresentSlice(keys, func(key string) (*T, error) {
		return r.findByKey(ctx, key)
	})
}

// Count returns the number of entities stored under the repository key prefix.
func (r *JSONRepository[T]) Count(ctx context.Context) (int64, error) {
	keys, err := r.base.scanAllKeys(ctx, r.kv)
	if err != nil {
		return 0, err
	}

	return int64(len(keys)), nil
}

func (r *JSONRepository[T]) findByKey(ctx context.Context, key string) (*T, error) {
	r.logDebug("kvx json find_by_key started", "key", key)

	data, err := r.client.JSONGet(ctx, key, "$")
	if err != nil {
		if kvx.IsNil(err) {
			r.logDebug("kvx json find_by_key not found", "key", key)
			return nil, ErrNotFound
		}

		wrapped := wrapRepositoryError(err, "read JSON entity")
		r.logError("kvx json find_by_key failed", "stage", "json_get", "key", key, "error", wrapped)
		return nil, wrapped
	}
	if len(data) == 0 {
		r.logDebug("kvx json find_by_key not found", "key", key)
		return nil, ErrNotFound
	}

	var entity T
	if unmarshalErr := r.serializer.Unmarshal(data, &entity); unmarshalErr != nil {
		wrapped := wrapRepositoryError(unmarshalErr, "unmarshal JSON entity")
		r.logError("kvx json find_by_key failed", "stage", "unmarshal", "key", key, "error", wrapped)
		return nil, wrapped
	}

	metadata, err := r.base.metadataForType()
	if err != nil {
		r.logError("kvx json find_by_key failed", "stage", "metadata", "key", key, "error", err)
		return nil, err
	}
	if err := r.base.hydrateEntityID(&entity, metadata, key); err != nil {
		r.logError("kvx json find_by_key failed", "stage", "hydrate_id", "key", key, "error", err)
		return nil, err
	}

	r.logDebug("kvx json find_by_key completed", "key", key)
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
