package repository

import (
	"context"
)

// FindByID loads an entity by its logical ID.
func (r *HashRepository[T]) FindByID(ctx context.Context, id string) (*T, error) {
	return r.findByKey(ctx, r.base.keyFromID(id))
}

// FindByIDs loads all entities that exist for the provided logical IDs.
func (r *HashRepository[T]) FindByIDs(ctx context.Context, ids []string) (map[string]*T, error) {
	return collectPresentMap(ids, func(id string) (*T, error) {
		return r.FindByID(ctx, id)
	})
}

// Exists reports whether an entity exists for the provided logical ID.
func (r *HashRepository[T]) Exists(ctx context.Context, id string) (bool, error) {
	exists, err := r.kv.Exists(ctx, r.base.keyFromID(id))
	return wrapRepositoryResult(exists, err, "check hash entity existence")
}

// ExistsBatch reports entity existence for each provided logical ID.
func (r *HashRepository[T]) ExistsBatch(ctx context.Context, ids []string) (map[string]bool, error) {
	keys := r.base.keysFromIDs(ids)
	existsMap, err := r.kv.ExistsMulti(ctx, keys)
	if err != nil {
		return nil, wrapRepositoryError(err, "check hash entity existence in batch")
	}

	return mapExistsResults(ids, keys, existsMap), nil
}

// FindAll loads all entities stored under the repository key prefix.
func (r *HashRepository[T]) FindAll(ctx context.Context) ([]*T, error) {
	keys, err := r.base.scanAllKeys(ctx, r.kv)
	if err != nil {
		return nil, err
	}

	return collectPresentSlice(keys, func(key string) (*T, error) {
		return r.findByKey(ctx, key)
	})
}

// Count returns the number of entities stored under the repository key prefix.
func (r *HashRepository[T]) Count(ctx context.Context) (int64, error) {
	keys, err := r.base.scanAllKeys(ctx, r.kv)
	if err != nil {
		return 0, err
	}

	return int64(len(keys)), nil
}

// FindByField loads all entities whose indexed field matches the provided value.
func (r *HashRepository[T]) FindByField(ctx context.Context, fieldName, fieldValue string) ([]*T, error) {
	entityIDs, err := r.base.idsByField(ctx, fieldName, fieldValue)
	if err != nil {
		return nil, err
	}

	return r.findManyByIDs(ctx, entityIDs)
}

// FindByFields loads all entities that match every provided indexed field value.
func (r *HashRepository[T]) FindByFields(ctx context.Context, fields map[string]string) ([]*T, error) {
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

func (r *HashRepository[T]) findByKey(ctx context.Context, key string) (*T, error) {
	r.logDebug("kvx hash find_by_key started", "key", key)

	hashData, err := r.client.HGetAll(ctx, key)
	if err != nil {
		wrapped := wrapRepositoryError(err, "read hash entity data")
		r.logError("kvx hash find_by_key failed", "stage", "hgetall", "key", key, "error", wrapped)
		return nil, wrapped
	}
	if len(hashData) == 0 {
		r.logDebug("kvx hash find_by_key not found", "key", key)
		return nil, ErrNotFound
	}

	var entity T
	metadata, err := r.base.metadataForType()
	if err != nil {
		r.logError("kvx hash find_by_key failed", "stage", "metadata", "key", key, "error", err)
		return nil, err
	}
	if err := r.codec.Decode(hashData, &entity, metadata); err != nil {
		wrapped := wrapRepositoryError(err, "decode hash entity")
		r.logError("kvx hash find_by_key failed", "stage", "decode", "key", key, "error", wrapped)
		return nil, wrapped
	}
	if err := r.base.hydrateEntityID(&entity, metadata, key); err != nil {
		r.logError("kvx hash find_by_key failed", "stage", "hydrate_id", "key", key, "error", err)
		return nil, err
	}

	r.logDebug("kvx hash find_by_key completed", "key", key)
	return &entity, nil
}

func (r *HashRepository[T]) findManyByIDs(ctx context.Context, ids []string) ([]*T, error) {
	return collectPresentSlice(ids, func(id string) (*T, error) {
		return r.FindByID(ctx, id)
	})
}
