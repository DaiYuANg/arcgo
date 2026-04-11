package repository

import (
	"context"
	"errors"
	"time"
)

type jsonSaveState[T any] struct {
	key           string
	payload       []byte
	removeEntries []string
	addEntries    []string
}

type jsonFieldUpdateState struct {
	key           string
	fieldPath     string
	payload       []byte
	removeEntries []string
	addEntries    []string
}

type jsonDeleteState = repositoryDeleteState

// Save stores an entity in the JSON repository without setting a TTL.
func (r *JSONRepository[T]) Save(ctx context.Context, entity *T) error {
	return r.SaveWithExpiration(ctx, entity, 0)
}

// SaveWithExpiration stores an entity in the JSON repository and optionally sets a TTL.
func (r *JSONRepository[T]) SaveWithExpiration(ctx context.Context, entity *T, expiration time.Duration) error {
	r.logDebug("kvx json save started", "expiration_ms", expiration.Milliseconds())

	state, err := r.prepareJSONSave(ctx, entity)
	if err != nil {
		r.logError("kvx json save failed", "error", err)
		return err
	}
	if err := r.persistJSONSave(ctx, state, expiration); err != nil {
		r.logError("kvx json save failed", "key", state.key, "error", err)
		return err
	}

	r.logDebug("kvx json save completed", "key", state.key, "indexed", len(state.addEntries))
	return nil
}

// SaveBatch stores a batch of entities in the JSON repository without setting a TTL.
func (r *JSONRepository[T]) SaveBatch(ctx context.Context, entities []*T) error {
	return r.SaveBatchWithExpiration(ctx, entities, 0)
}

// SaveBatchWithExpiration stores a batch of entities in the JSON repository and optionally sets a TTL.
func (r *JSONRepository[T]) SaveBatchWithExpiration(ctx context.Context, entities []*T, expiration time.Duration) error {
	if len(entities) == 0 {
		return nil
	}

	return runAll(entities, func(entity *T) error {
		return r.SaveWithExpiration(ctx, entity, expiration)
	})
}

// Delete removes an entity and all of its index entries.
func (r *JSONRepository[T]) Delete(ctx context.Context, id string) error {
	state, found, err := r.prepareJSONDelete(ctx, id)
	if err != nil {
		r.logError("kvx json delete failed", "error", err)
		return err
	}
	if !found {
		r.logDebug("kvx json delete skipped", "key", state.key, "reason", "not_found")
		return nil
	}
	if err := r.persistJSONDelete(ctx, state); err != nil {
		r.logError("kvx json delete failed", "key", state.key, "error", err)
		return err
	}

	r.logDebug("kvx json delete completed", "key", state.key)
	return nil
}

// DeleteBatch removes a batch of entities and their index entries.
func (r *JSONRepository[T]) DeleteBatch(ctx context.Context, ids []string) error {
	return runAll(ids, func(id string) error {
		return r.Delete(ctx, id)
	})
}

// UpdateField updates a JSON field path and refreshes any related index entries.
func (r *JSONRepository[T]) UpdateField(ctx context.Context, id, fieldPath string, value any) error {
	state, err := r.prepareJSONFieldUpdate(ctx, id, fieldPath, value)
	if err != nil {
		return err
	}

	if script, ok := r.script.Get(); ok {
		return execJSONFieldUpdateScript(
			ctx,
			script,
			state.key,
			state.fieldPath,
			state.payload,
			state.removeEntries,
			state.addEntries,
		)
	}
	if err := r.client.JSONSetField(ctx, state.key, state.fieldPath, state.payload); err != nil {
		return wrapRepositoryError(err, "write JSON field value", "op", "write_json_field_value", "key", state.key, "field_path", state.fieldPath)
	}

	return r.base.indexer.ApplyIndexDiff(ctx, state.removeEntries, state.addEntries)
}

func (r *JSONRepository[T]) prepareJSONSave(ctx context.Context, entity *T) (jsonSaveState[T], error) {
	metadata, err := r.base.metadata(entity)
	if err != nil {
		return jsonSaveState[T]{}, err
	}

	key, err := r.base.keyBuilder.Build(entity, metadata)
	if err != nil {
		return jsonSaveState[T]{}, wrapRepositoryError(err, "build JSON entity key", "op", "build_json_entity_key")
	}

	payload, err := r.serializer.Marshal(entity)
	if err != nil {
		return jsonSaveState[T]{}, wrapRepositoryError(err, "marshal JSON entity", "op", "marshal_json_entity", "key", key)
	}

	previous, _, err := r.loadPreviousJSONEntity(ctx, key)
	if err != nil {
		return jsonSaveState[T]{}, err
	}

	removeEntries, addEntries, err := r.base.indexer.ReplaceEntityIndexEntries(ctx, previous, entity, metadata, key)
	if err != nil {
		return jsonSaveState[T]{}, wrapRepositoryError(err, "compute JSON index diff", "op", "compute_json_index_diff", "key", key)
	}

	return jsonSaveState[T]{
		key:           key,
		payload:       payload,
		removeEntries: removeEntries,
		addEntries:    addEntries,
	}, nil
}

func (r *JSONRepository[T]) loadPreviousJSONEntity(ctx context.Context, key string) (*T, bool, error) {
	entity, err := r.findByKey(ctx, key)
	if errors.Is(err, ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, wrapRepositoryError(err, "load existing JSON entity", "op", "load_existing_json_entity", "key", key)
	}

	return entity, true, nil
}

func (r *JSONRepository[T]) persistJSONSave(ctx context.Context, state jsonSaveState[T], expiration time.Duration) error {
	if script, ok := r.script.Get(); ok {
		return execJSONUpsertScript(
			ctx,
			script,
			state.key,
			state.payload,
			expiration,
			state.removeEntries,
			state.addEntries,
		)
	}

	if err := r.client.JSONSet(ctx, state.key, "$", state.payload, expiration); err != nil {
		return wrapRepositoryError(err, "write JSON entity", "op", "write_json_entity", "key", state.key, "payload_size", len(state.payload), "expiration", expiration)
	}

	return r.base.indexer.ApplyIndexDiff(ctx, state.removeEntries, state.addEntries)
}

func (r *JSONRepository[T]) prepareJSONDelete(ctx context.Context, id string) (jsonDeleteState, bool, error) {
	return prepareRepositoryDelete(
		ctx,
		r.base,
		id,
		r.findByKey,
		r.logDebug,
		"kvx json delete started",
		"load_json_entity_for_delete",
		"load JSON entity for delete",
		"collect_json_index_entries_for_delete",
		"collect JSON index entries for delete",
	)
}

func (r *JSONRepository[T]) persistJSONDelete(ctx context.Context, state jsonDeleteState) error {
	if script, ok := r.script.Get(); ok {
		return execDeleteScript(ctx, script, state.key, state.removeEntries)
	}
	if err := r.client.JSONDelete(ctx, state.key, "$"); err != nil {
		return wrapRepositoryError(err, "delete JSON entity", "op", "delete_json_entity", "key", state.key)
	}

	return r.base.indexer.ApplyIndexDiff(ctx, state.removeEntries, nil)
}

func (r *JSONRepository[T]) prepareJSONFieldUpdate(ctx context.Context, id, fieldPath string, value any) (jsonFieldUpdateState, error) {
	key := r.base.keyFromID(id)
	entity, err := r.findByKey(ctx, key)
	if err != nil {
		return jsonFieldUpdateState{}, wrapRepositoryError(err, "load JSON entity for field update", "op", "load_json_entity_for_field_update", "id", id, "key", key, "field_path", fieldPath)
	}

	metadata, err := r.base.metadata(entity)
	if err != nil {
		return jsonFieldUpdateState{}, err
	}

	fieldName := extractFieldNameFromPath(fieldPath)
	resolvedField, fieldTag, exists := metadata.ResolveField(fieldName)

	payload, err := r.serializer.Marshal(value)
	if err != nil {
		return jsonFieldUpdateState{}, wrapRepositoryError(err, "marshal JSON field value", "op", "marshal_json_field_value", "id", id, "key", key, "field_path", fieldPath)
	}

	removeEntries := []string(nil)
	addEntries := []string(nil)
	if exists && fieldTag.Index {
		removeEntries, addEntries, err = r.base.indexer.ReplaceFieldIndexEntries(metadata, resolvedField, key, entity, value)
		if err != nil {
			return jsonFieldUpdateState{}, wrapRepositoryError(err, "compute JSON field index diff", "op", "compute_json_field_index_diff", "id", id, "key", key, "field_path", fieldPath, "field_name", fieldName)
		}
	}

	return jsonFieldUpdateState{
		key:           key,
		fieldPath:     fieldPath,
		payload:       payload,
		removeEntries: removeEntries,
		addEntries:    addEntries,
	}, nil
}
