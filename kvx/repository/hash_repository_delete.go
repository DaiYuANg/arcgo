package repository

import "context"

type hashDeleteState = repositoryDeleteState

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
	return prepareRepositoryDelete(
		ctx,
		r.base,
		id,
		r.findByKey,
		r.logDebug,
		"kvx hash delete started",
		"load_hash_entity_for_delete",
		"load hash entity for delete",
		"collect_hash_index_entries_for_delete",
		"collect hash index entries for delete",
	)
}

func (r *HashRepository[T]) persistHashDelete(ctx context.Context, state hashDeleteState) error {
	if script, ok := r.script.Get(); ok {
		return execDeleteScript(ctx, script, state.key, state.removeEntries)
	}

	fields, err := r.client.HKeys(ctx, state.key)
	if err != nil {
		return wrapRepositoryError(err, "list hash fields for delete", "op", "list_hash_fields_for_delete", "key", state.key)
	}
	if !fields.IsEmpty() {
		if err := r.client.HDel(ctx, state.key, fields.Values()...); err != nil {
			return wrapRepositoryError(err, "delete hash fields", "op", "delete_hash_fields", "key", state.key, "field_count", fields.Len())
		}
	}
	if err := r.kv.Delete(ctx, state.key); err != nil {
		return wrapRepositoryError(err, "delete hash entity key", "op", "delete_hash_entity_key", "key", state.key)
	}

	return r.base.indexer.ApplyIndexDiff(ctx, state.removeEntries, nil)
}
