package repository

import (
	"context"
	"errors"
)

type hashDeleteState struct {
	key           string
	removeEntries []string
}

// Delete removes an entity and all of its index entries.
func (r *HashRepository[T]) Delete(ctx context.Context, id string) error {
	state, found, err := r.prepareHashDelete(ctx, id)
	if err != nil {
		r.logError("kvx hash delete failed", "error", err)
		return err
	}
	if !found {
		r.logDebug("kvx hash delete skipped", "key", state.key, "reason", "not_found")
		return nil
	}
	if err := r.persistHashDelete(ctx, state); err != nil {
		r.logError("kvx hash delete failed", "key", state.key, "error", err)
		return err
	}

	r.logDebug("kvx hash delete completed", "key", state.key)
	return nil
}

// DeleteBatch removes a batch of entities and their index entries.
func (r *HashRepository[T]) DeleteBatch(ctx context.Context, ids []string) error {
	return runAll(ids, func(id string) error {
		return r.Delete(ctx, id)
	})
}

func (r *HashRepository[T]) prepareHashDelete(ctx context.Context, id string) (hashDeleteState, bool, error) {
	key := r.base.keyFromID(id)
	r.logDebug("kvx hash delete started", "key", key)

	entity, err := r.findByKey(ctx, key)
	if errors.Is(err, ErrNotFound) {
		return hashDeleteState{key: key}, false, nil
	}
	if err != nil {
		return hashDeleteState{key: key}, false, wrapRepositoryError(err, "load hash entity for delete")
	}

	metadata, err := r.base.metadata(entity)
	if err != nil {
		return hashDeleteState{key: key}, false, err
	}

	removeEntries, err := r.base.indexer.EntityIndexEntries(entity, metadata, key)
	if err != nil {
		return hashDeleteState{key: key}, false, wrapRepositoryError(err, "collect hash index entries for delete")
	}

	return hashDeleteState{
		key:           key,
		removeEntries: removeEntries,
	}, true, nil
}

func (r *HashRepository[T]) persistHashDelete(ctx context.Context, state hashDeleteState) error {
	if script, ok := r.script.Get(); ok {
		return execDeleteScript(ctx, script, state.key, state.removeEntries)
	}

	fields, err := r.client.HKeys(ctx, state.key)
	if err != nil {
		return wrapRepositoryError(err, "list hash fields for delete")
	}
	if !fields.IsEmpty() {
		if err := r.client.HDel(ctx, state.key, fields.Values()...); err != nil {
			return wrapRepositoryError(err, "delete hash fields")
		}
	}
	if err := r.kv.Delete(ctx, state.key); err != nil {
		return wrapRepositoryError(err, "delete hash entity key")
	}

	return r.base.indexer.ApplyIndexDiff(ctx, state.removeEntries, nil)
}
