package dbx

import (
	"context"
	"fmt"

	"github.com/samber/mo"
)

func LoadBelongsTo[S any, T any](ctx context.Context, session Session, sources []S, sourceSchema SchemaSource[S], sourceMapper Mapper[S], relation BelongsTo[S, T], targetSchema SchemaSource[T], targetMapper Mapper[T], assign func(int, *S, mo.Option[T])) error {
	logRuntimeNode(session, "relation.load.belongs_to.start", "sources", len(sources))
	return loadSingleRelation(ctx, session, sources, sourceSchema, sourceMapper, relation.Meta(), targetSchema, targetMapper, assign)
}

func LoadHasOne[S any, T any](ctx context.Context, session Session, sources []S, sourceSchema SchemaSource[S], sourceMapper Mapper[S], relation HasOne[S, T], targetSchema SchemaSource[T], targetMapper Mapper[T], assign func(int, *S, mo.Option[T])) error {
	logRuntimeNode(session, "relation.load.has_one.start", "sources", len(sources))
	return loadSingleRelation(ctx, session, sources, sourceSchema, sourceMapper, relation.Meta(), targetSchema, targetMapper, assign)
}

func LoadHasMany[S any, T any](ctx context.Context, session Session, sources []S, sourceSchema SchemaSource[S], sourceMapper Mapper[S], relation HasMany[S, T], targetSchema SchemaSource[T], targetMapper Mapper[T], assign func(int, *S, []T)) error {
	logRuntimeNode(session, "relation.load.has_many.start", "sources", len(sources))
	return loadMultiRelation(ctx, session, sources, sourceSchema, sourceMapper, relation.Meta(), targetSchema, targetMapper, assign)
}

func LoadManyToMany[S any, T any](ctx context.Context, session Session, sources []S, sourceSchema SchemaSource[S], sourceMapper Mapper[S], relation ManyToMany[S, T], targetSchema SchemaSource[T], targetMapper Mapper[T], assign func(int, *S, []T)) error {
	logRuntimeNode(session, "relation.load.many_to_many.start", "sources", len(sources))
	if err := validateRelationLoadInputs(session, sourceSchema, sourceMapper, targetSchema, targetMapper); err != nil {
		logRuntimeNode(session, "relation.load.many_to_many.error", "stage", "validate_inputs", "error", err)
		return err
	}
	if assign == nil {
		return fmt.Errorf("dbx: relation loader requires assign callback")
	}
	if len(sources) == 0 {
		logRuntimeNode(session, "relation.load.many_to_many.done", "reason", "empty_sources")
		return nil
	}

	meta := relation.Meta()
	rt := getRelationRuntime(session)
	sourceKeys, sourceLookup, err := collectSourceRelationKeys(rt, sources, sourceMapper, sourceSchema.schemaRef(), meta)
	if err != nil {
		logRuntimeNode(session, "relation.load.many_to_many.error", "stage", "collect_source_keys", "error", err)
		return err
	}
	if len(sourceKeys) == 0 {
		assignEmptyRelations(sources, assign)
		logRuntimeNode(session, "relation.load.many_to_many.done", "reason", "no_source_keys")
		return nil
	}

	targetColumn, err := relationTargetColumnForSchema(targetSchema, meta)
	if err != nil {
		logRuntimeNode(session, "relation.load.many_to_many.error", "stage", "resolve_target_column", "error", err)
		return err
	}
	pairs, err := queryManyToManyPairs(ctx, session, rt, meta, sourceKeys, relationKeyTypeForMeta(sourceSchema.schemaRef(), meta.LocalColumn), targetColumn.GoType)
	if err != nil {
		logRuntimeNode(session, "relation.load.many_to_many.error", "stage", "query_pairs", "error", err)
		return err
	}
	if len(pairs) == 0 {
		assignEmptyRelations(sources, assign)
		logRuntimeNode(session, "relation.load.many_to_many.done", "reason", "no_pairs")
		return nil
	}

	targetKeys := uniqueRelationKeysFromPairs(rt, pairs, false)
	targets, err := queryRelationTargets(ctx, session, rt, targetSchema, targetMapper, targetColumn, targetKeys)
	if err != nil {
		logRuntimeNode(session, "relation.load.many_to_many.error", "stage", "query_targets", "error", err)
		return err
	}
	targetsByKey, err := indexRelationTargets(targets, targetMapper, targetColumn.Name)
	if err != nil {
		logRuntimeNode(session, "relation.load.many_to_many.error", "stage", "index_targets", "error", err)
		return err
	}
	grouped := groupManyToManyTargets(rt, pairs, targetsByKey)
	for index := range sources {
		key := sourceLookup[index]
		assign(index, &sources[index], grouped[key.key])
	}
	logRuntimeNode(session, "relation.load.many_to_many.done", "sources", len(sources), "targets", len(targets))
	return nil
}

func loadSingleRelation[S any, T any](ctx context.Context, session Session, sources []S, sourceSchema SchemaSource[S], sourceMapper Mapper[S], meta RelationMeta, targetSchema SchemaSource[T], targetMapper Mapper[T], assign func(int, *S, mo.Option[T])) error {
	if err := validateRelationLoadInputs(session, sourceSchema, sourceMapper, targetSchema, targetMapper); err != nil {
		logRuntimeNode(session, "relation.load.single.error", "stage", "validate_inputs", "error", err)
		return err
	}
	if assign == nil {
		return fmt.Errorf("dbx: relation loader requires assign callback")
	}
	if len(sources) == 0 {
		logRuntimeNode(session, "relation.load.single.done", "reason", "empty_sources")
		return nil
	}

	rt := getRelationRuntime(session)
	sourceKeys, sourceLookup, err := collectSourceRelationKeys(rt, sources, sourceMapper, sourceSchema.schemaRef(), meta)
	if err != nil {
		logRuntimeNode(session, "relation.load.single.error", "stage", "collect_source_keys", "error", err)
		return err
	}
	if len(sourceKeys) == 0 {
		for index := range sources {
			assign(index, &sources[index], mo.None[T]())
		}
		logRuntimeNode(session, "relation.load.single.done", "reason", "no_source_keys")
		return nil
	}

	targetColumn, err := relationTargetColumnForSchema(targetSchema, meta)
	if err != nil {
		logRuntimeNode(session, "relation.load.single.error", "stage", "resolve_target_column", "error", err)
		return err
	}
	targets, err := queryRelationTargets(ctx, session, rt, targetSchema, targetMapper, targetColumn, sourceKeys)
	if err != nil {
		logRuntimeNode(session, "relation.load.single.error", "stage", "query_targets", "error", err)
		return err
	}
	targetsByKey, err := indexRelationTargets(targets, targetMapper, targetColumn.Name)
	if err != nil {
		logRuntimeNode(session, "relation.load.single.error", "stage", "index_targets", "error", err)
		return err
	}
	for index := range sources {
		key := sourceLookup[index]
		if !key.present {
			assign(index, &sources[index], mo.None[T]())
			continue
		}
		target, ok := targetsByKey[key.key]
		if !ok {
			assign(index, &sources[index], mo.None[T]())
			continue
		}
		assign(index, &sources[index], mo.Some(target))
	}
	logRuntimeNode(session, "relation.load.single.done", "sources", len(sources), "targets", len(targets))
	return nil
}

func loadMultiRelation[S any, T any](ctx context.Context, session Session, sources []S, sourceSchema SchemaSource[S], sourceMapper Mapper[S], meta RelationMeta, targetSchema SchemaSource[T], targetMapper Mapper[T], assign func(int, *S, []T)) error {
	if err := validateRelationLoadInputs(session, sourceSchema, sourceMapper, targetSchema, targetMapper); err != nil {
		logRuntimeNode(session, "relation.load.multi.error", "stage", "validate_inputs", "error", err)
		return err
	}
	if assign == nil {
		return fmt.Errorf("dbx: relation loader requires assign callback")
	}
	if len(sources) == 0 {
		logRuntimeNode(session, "relation.load.multi.done", "reason", "empty_sources")
		return nil
	}

	rt := getRelationRuntime(session)
	sourceKeys, sourceLookup, err := collectSourceRelationKeys(rt, sources, sourceMapper, sourceSchema.schemaRef(), meta)
	if err != nil {
		logRuntimeNode(session, "relation.load.multi.error", "stage", "collect_source_keys", "error", err)
		return err
	}
	if len(sourceKeys) == 0 {
		assignEmptyRelations(sources, assign)
		logRuntimeNode(session, "relation.load.multi.done", "reason", "no_source_keys")
		return nil
	}

	targetColumn, err := relationTargetColumnForSchema(targetSchema, meta)
	if err != nil {
		logRuntimeNode(session, "relation.load.multi.error", "stage", "resolve_target_column", "error", err)
		return err
	}
	targets, err := queryRelationTargets(ctx, session, rt, targetSchema, targetMapper, targetColumn, sourceKeys)
	if err != nil {
		logRuntimeNode(session, "relation.load.multi.error", "stage", "query_targets", "error", err)
		return err
	}
	grouped, err := groupRelationTargets(rt, targets, targetMapper, targetColumn.Name)
	if err != nil {
		logRuntimeNode(session, "relation.load.multi.error", "stage", "group_targets", "error", err)
		return err
	}
	for index := range sources {
		key := sourceLookup[index]
		assign(index, &sources[index], grouped[key.key])
	}
	logRuntimeNode(session, "relation.load.multi.done", "sources", len(sources), "targets", len(targets))
	return nil
}

func validateRelationLoadInputs[S any, T any](session Session, sourceSchema SchemaSource[S], sourceMapper Mapper[S], targetSchema SchemaSource[T], targetMapper Mapper[T]) error {
	switch {
	case session == nil:
		return ErrNilDB
	case sourceSchema == nil:
		return fmt.Errorf("dbx: source schema is nil")
	case targetSchema == nil:
		return fmt.Errorf("dbx: target schema is nil")
	case sourceMapper.meta == nil:
		return ErrNilMapper
	case targetMapper.meta == nil:
		return ErrNilMapper
	default:
		return nil
	}
}

func assignEmptyRelations[S any, T any](sources []S, assign func(int, *S, []T)) {
	for index := range sources {
		assign(index, &sources[index], nil)
	}
}
