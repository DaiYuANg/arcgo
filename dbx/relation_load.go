package dbx

import (
	"context"
	"fmt"

	"github.com/samber/mo"
)

func LoadBelongsTo[S any, T any](ctx context.Context, session Session, sources []S, sourceSchema SchemaSource[S], sourceMapper Mapper[S], relation BelongsTo[S, T], targetSchema SchemaSource[T], targetMapper Mapper[T], assign func(int, *S, mo.Option[T])) error {
	return loadSingleRelation(ctx, session, sources, sourceSchema, sourceMapper, relation.Meta(), targetSchema, targetMapper, assign)
}

func LoadHasOne[S any, T any](ctx context.Context, session Session, sources []S, sourceSchema SchemaSource[S], sourceMapper Mapper[S], relation HasOne[S, T], targetSchema SchemaSource[T], targetMapper Mapper[T], assign func(int, *S, mo.Option[T])) error {
	return loadSingleRelation(ctx, session, sources, sourceSchema, sourceMapper, relation.Meta(), targetSchema, targetMapper, assign)
}

func LoadHasMany[S any, T any](ctx context.Context, session Session, sources []S, sourceSchema SchemaSource[S], sourceMapper Mapper[S], relation HasMany[S, T], targetSchema SchemaSource[T], targetMapper Mapper[T], assign func(int, *S, []T)) error {
	return loadMultiRelation(ctx, session, sources, sourceSchema, sourceMapper, relation.Meta(), targetSchema, targetMapper, assign)
}

func LoadManyToMany[S any, T any](ctx context.Context, session Session, sources []S, sourceSchema SchemaSource[S], sourceMapper Mapper[S], relation ManyToMany[S, T], targetSchema SchemaSource[T], targetMapper Mapper[T], assign func(int, *S, []T)) error {
	if err := validateRelationLoadInputs(session, sourceSchema, sourceMapper, targetSchema, targetMapper); err != nil {
		return err
	}
	if assign == nil {
		return fmt.Errorf("dbx: relation loader requires assign callback")
	}
	if len(sources) == 0 {
		return nil
	}

	meta := relation.Meta()
	sourceKeys, sourceLookup, err := collectSourceRelationKeys(sources, sourceMapper, sourceSchema.schemaRef(), meta)
	if err != nil {
		return err
	}
	if len(sourceKeys) == 0 {
		assignEmptyRelations(sources, assign)
		return nil
	}

	targetColumn, err := relationTargetColumnForSchema(targetSchema, meta)
	if err != nil {
		return err
	}
	pairs, err := queryManyToManyPairs(ctx, session, meta, sourceKeys, relationKeyTypeForMeta(sourceSchema.schemaRef(), meta.LocalColumn), targetColumn.GoType)
	if err != nil {
		return err
	}
	if len(pairs) == 0 {
		assignEmptyRelations(sources, assign)
		return nil
	}

	targetKeys := uniqueRelationKeysFromPairs(pairs, false)
	targets, err := queryRelationTargets(ctx, session, targetSchema, targetMapper, targetColumn, targetKeys)
	if err != nil {
		return err
	}
	targetsByKey, err := indexRelationTargets(targets, targetMapper, targetColumn.Name)
	if err != nil {
		return err
	}
	grouped := groupManyToManyTargets(pairs, targetsByKey)
	for index := range sources {
		key := sourceLookup[index]
		assign(index, &sources[index], grouped[key.key])
	}
	return nil
}

func loadSingleRelation[S any, T any](ctx context.Context, session Session, sources []S, sourceSchema SchemaSource[S], sourceMapper Mapper[S], meta RelationMeta, targetSchema SchemaSource[T], targetMapper Mapper[T], assign func(int, *S, mo.Option[T])) error {
	if err := validateRelationLoadInputs(session, sourceSchema, sourceMapper, targetSchema, targetMapper); err != nil {
		return err
	}
	if assign == nil {
		return fmt.Errorf("dbx: relation loader requires assign callback")
	}
	if len(sources) == 0 {
		return nil
	}

	sourceKeys, sourceLookup, err := collectSourceRelationKeys(sources, sourceMapper, sourceSchema.schemaRef(), meta)
	if err != nil {
		return err
	}
	if len(sourceKeys) == 0 {
		for index := range sources {
			assign(index, &sources[index], mo.None[T]())
		}
		return nil
	}

	targetColumn, err := relationTargetColumnForSchema(targetSchema, meta)
	if err != nil {
		return err
	}
	targets, err := queryRelationTargets(ctx, session, targetSchema, targetMapper, targetColumn, sourceKeys)
	if err != nil {
		return err
	}
	targetsByKey, err := indexRelationTargets(targets, targetMapper, targetColumn.Name)
	if err != nil {
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
	return nil
}

func loadMultiRelation[S any, T any](ctx context.Context, session Session, sources []S, sourceSchema SchemaSource[S], sourceMapper Mapper[S], meta RelationMeta, targetSchema SchemaSource[T], targetMapper Mapper[T], assign func(int, *S, []T)) error {
	if err := validateRelationLoadInputs(session, sourceSchema, sourceMapper, targetSchema, targetMapper); err != nil {
		return err
	}
	if assign == nil {
		return fmt.Errorf("dbx: relation loader requires assign callback")
	}
	if len(sources) == 0 {
		return nil
	}

	sourceKeys, sourceLookup, err := collectSourceRelationKeys(sources, sourceMapper, sourceSchema.schemaRef(), meta)
	if err != nil {
		return err
	}
	if len(sourceKeys) == 0 {
		assignEmptyRelations(sources, assign)
		return nil
	}

	targetColumn, err := relationTargetColumnForSchema(targetSchema, meta)
	if err != nil {
		return err
	}
	targets, err := queryRelationTargets(ctx, session, targetSchema, targetMapper, targetColumn, sourceKeys)
	if err != nil {
		return err
	}
	grouped, err := groupRelationTargets(targets, targetMapper, targetColumn.Name)
	if err != nil {
		return err
	}
	for index := range sources {
		key := sourceLookup[index]
		assign(index, &sources[index], grouped[key.key])
	}
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
