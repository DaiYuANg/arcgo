package dbx

import (
	"context"
	"errors"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/mo"
)

type relationSourceState struct {
	rt     *relationRuntime
	keys   collectionx.List[any]
	lookup []relationLookupValue
}

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
	const logPrefix = "relation.load.many_to_many"
	logRuntimeNode(session, logPrefix+".start", "sources", len(sources))
	if proceed, err := startRelationLoad(session, sources, sourceSchema, sourceMapper, targetSchema, targetMapper, assign != nil, logPrefix); err != nil || !proceed {
		return err
	}
	meta := relation.Meta()
	state, err := prepareRelationSourceState(session, sources, sourceSchema, sourceMapper, meta, logPrefix)
	if err != nil {
		return err
	}
	if state.keys.Len() == 0 {
		assignEmptyRelations(sources, assign)
		logRelationLoadDone(session, logPrefix, "reason", "no_source_keys")
		return nil
	}
	grouped, targetCount, hasPairs, err := loadManyToManyGroupedTargets(ctx, session, state.rt, sourceSchema.schemaRef(), meta, targetSchema, targetMapper, state.keys, logPrefix)
	if err != nil {
		return err
	}
	if !hasPairs {
		assignEmptyRelations(sources, assign)
		logRelationLoadDone(session, logPrefix, "reason", "no_pairs")
		return nil
	}
	for index := range sources {
		key := state.lookup[index]
		assign(index, &sources[index], grouped.Get(key.key))
	}
	logRelationLoadDone(session, logPrefix, "sources", len(sources), "targets", targetCount)
	return nil
}

func loadSingleRelation[S any, T any](ctx context.Context, session Session, sources []S, sourceSchema SchemaSource[S], sourceMapper Mapper[S], meta RelationMeta, targetSchema SchemaSource[T], targetMapper Mapper[T], assign func(int, *S, mo.Option[T])) error {
	const logPrefix = "relation.load.single"
	if proceed, err := startRelationLoad(session, sources, sourceSchema, sourceMapper, targetSchema, targetMapper, assign != nil, logPrefix); err != nil || !proceed {
		return err
	}
	state, err := prepareRelationSourceState(session, sources, sourceSchema, sourceMapper, meta, logPrefix)
	if err != nil {
		return err
	}
	if state.keys.Len() == 0 {
		assignMissingSingleRelations(sources, assign)
		logRelationLoadDone(session, logPrefix, "reason", "no_source_keys")
		return nil
	}
	targetsByKey, targetCount, err := loadSingleRelationTargets(ctx, session, state.rt, meta, targetSchema, targetMapper, state.keys, logPrefix)
	if err != nil {
		return err
	}
	assignLoadedSingleRelations(sources, state.lookup, targetsByKey, assign)
	logRelationLoadDone(session, logPrefix, "sources", len(sources), "targets", targetCount)
	return nil
}

func loadMultiRelation[S any, T any](ctx context.Context, session Session, sources []S, sourceSchema SchemaSource[S], sourceMapper Mapper[S], meta RelationMeta, targetSchema SchemaSource[T], targetMapper Mapper[T], assign func(int, *S, []T)) error {
	if err := validateRelationLoadInputs(session, sourceSchema, sourceMapper, targetSchema, targetMapper); err != nil {
		logRuntimeNode(session, "relation.load.multi.error", "stage", "validate_inputs", "error", err)
		return err
	}
	if assign == nil {
		return errors.New("dbx: relation loader requires assign callback")
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
	if sourceKeys.Len() == 0 {
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
		assign(index, &sources[index], grouped.Get(key.key))
	}
	logRuntimeNode(session, "relation.load.multi.done", "sources", len(sources), "targets", targets.Len())
	return nil
}

func validateRelationLoadInputs[S any, T any](session Session, sourceSchema SchemaSource[S], sourceMapper Mapper[S], targetSchema SchemaSource[T], targetMapper Mapper[T]) error {
	switch {
	case session == nil:
		return ErrNilDB
	case sourceSchema == nil:
		return errors.New("dbx: source schema is nil")
	case targetSchema == nil:
		return errors.New("dbx: target schema is nil")
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

func startRelationLoad[S any, T any](session Session, sources []S, sourceSchema SchemaSource[S], sourceMapper Mapper[S], targetSchema SchemaSource[T], targetMapper Mapper[T], assignProvided bool, logPrefix string) (bool, error) {
	if err := validateRelationLoadInputs(session, sourceSchema, sourceMapper, targetSchema, targetMapper); err != nil {
		logRelationLoadError(session, logPrefix, "validate_inputs", err)
		return false, err
	}
	if !assignProvided {
		err := errors.New("dbx: relation loader requires assign callback")
		logRelationLoadError(session, logPrefix, "assign_callback", err)
		return false, err
	}
	if len(sources) == 0 {
		logRelationLoadDone(session, logPrefix, "reason", "empty_sources")
		return false, nil
	}
	return true, nil
}

func prepareRelationSourceState[E any](session Session, sources []E, sourceSchema SchemaSource[E], sourceMapper Mapper[E], meta RelationMeta, logPrefix string) (relationSourceState, error) {
	rt := getRelationRuntime(session)
	keys, lookup, err := collectSourceRelationKeys(rt, sources, sourceMapper, sourceSchema.schemaRef(), meta)
	if err != nil {
		logRelationLoadError(session, logPrefix, "collect_source_keys", err)
		return relationSourceState{}, err
	}
	return relationSourceState{rt: rt, keys: keys, lookup: lookup}, nil
}

func loadManyToManyGroupedTargets[T any](ctx context.Context, session Session, rt *relationRuntime, sourceSchema schemaDefinition, meta RelationMeta, targetSchema SchemaSource[T], targetMapper Mapper[T], sourceKeys collectionx.List[any], logPrefix string) (collectionx.MultiMap[any, T], int, bool, error) {
	targetColumn, err := relationTargetColumnForSchema(targetSchema, meta)
	if err != nil {
		logRelationLoadError(session, logPrefix, "resolve_target_column", err)
		return nil, 0, false, err
	}
	pairs, err := queryManyToManyPairs(ctx, session, rt, meta, sourceKeys, relationKeyTypeForMeta(sourceSchema, meta.LocalColumn), targetColumn.GoType)
	if err != nil {
		logRelationLoadError(session, logPrefix, "query_pairs", err)
		return nil, 0, false, err
	}
	if pairs.Len() == 0 {
		return collectionx.NewMultiMap[any, T](), 0, false, nil
	}
	targetKeys := uniqueRelationKeysFromPairs(rt, pairs, false)
	targets, err := queryRelationTargets(ctx, session, rt, targetSchema, targetMapper, targetColumn, targetKeys)
	if err != nil {
		logRelationLoadError(session, logPrefix, "query_targets", err)
		return nil, 0, false, err
	}
	targetsByKey, err := indexRelationTargets(targets, targetMapper, targetColumn.Name, "", false)
	if err != nil {
		logRelationLoadError(session, logPrefix, "index_targets", err)
		return nil, 0, false, err
	}
	return groupManyToManyTargets(rt, pairs, targetsByKey), targets.Len(), true, nil
}

func loadSingleRelationTargets[T any](ctx context.Context, session Session, rt *relationRuntime, meta RelationMeta, targetSchema SchemaSource[T], targetMapper Mapper[T], sourceKeys collectionx.List[any], logPrefix string) (map[any]T, int, error) {
	targetColumn, err := relationTargetColumnForSchema(targetSchema, meta)
	if err != nil {
		logRelationLoadError(session, logPrefix, "resolve_target_column", err)
		return nil, 0, err
	}
	targets, err := queryRelationTargets(ctx, session, rt, targetSchema, targetMapper, targetColumn, sourceKeys)
	if err != nil {
		logRelationLoadError(session, logPrefix, "query_targets", err)
		return nil, 0, err
	}
	targetsByKey, err := indexRelationTargets(targets, targetMapper, targetColumn.Name, meta.Name, meta.Kind == RelationHasOne)
	if err != nil {
		logRelationLoadError(session, logPrefix, "index_targets", err)
		return nil, 0, err
	}
	return targetsByKey, targets.Len(), nil
}

func assignMissingSingleRelations[S any, T any](sources []S, assign func(int, *S, mo.Option[T])) {
	for index := range sources {
		assign(index, &sources[index], mo.None[T]())
	}
}

func assignLoadedSingleRelations[S any, T any](sources []S, lookup []relationLookupValue, targetsByKey map[any]T, assign func(int, *S, mo.Option[T])) {
	for index := range sources {
		target, ok := relationTargetByLookup(lookup[index], targetsByKey)
		if !ok {
			assign(index, &sources[index], mo.None[T]())
			continue
		}
		assign(index, &sources[index], mo.Some(target))
	}
}

func relationTargetByLookup[T any](lookup relationLookupValue, targetsByKey map[any]T) (T, bool) {
	if !lookup.present {
		var zero T
		return zero, false
	}
	target, ok := targetsByKey[lookup.key]
	if !ok {
		var zero T
		return zero, false
	}
	return target, true
}

func logRelationLoadError(session Session, logPrefix, stage string, err error) {
	logRuntimeNode(session, logPrefix+".error", "stage", stage, "error", err)
}

func logRelationLoadDone(session Session, logPrefix string, attrs ...any) {
	logRuntimeNode(session, logPrefix+".done", attrs...)
}
