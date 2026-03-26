package activerecord

import (
	"context"

	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/DaiYuANg/arcgo/dbx/repository"
	"github.com/samber/lo"
	"github.com/samber/mo"
)

type Store[E any, S dbx.SchemaSource[E]] struct {
	repository *repository.Base[E, S]
}

func New[E any, S dbx.SchemaSource[E]](db *dbx.DB, schema S) *Store[E, S] {
	return &Store[E, S]{repository: repository.New[E](db, schema)}
}

func (s *Store[E, S]) Repository() *repository.Base[E, S] {
	return s.repository
}

func (s *Store[E, S]) Wrap(entity *E) *Model[E, S] {
	return s.newModel(entity)
}

func (s *Store[E, S]) FindByID(ctx context.Context, id any) (*Model[E, S], error) {
	entity, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.newKeyedModel(&entity, s.keyOf(&entity)), nil
}

func (s *Store[E, S]) FindByIDOption(ctx context.Context, id any) (mo.Option[*Model[E, S]], error) {
	entity, err := s.repository.GetByIDOption(ctx, id)
	return s.wrapOption(entity, err)
}

func (s *Store[E, S]) FindByKey(ctx context.Context, key repository.Key) (*Model[E, S], error) {
	entity, err := s.repository.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	return s.newKeyedModel(&entity, key), nil
}

func (s *Store[E, S]) FindByKeyOption(ctx context.Context, key repository.Key) (mo.Option[*Model[E, S]], error) {
	entity, err := s.repository.GetByKeyOption(ctx, key)
	if err != nil {
		return mo.None[*Model[E, S]](), err
	}
	return mapOption(entity, func(item E) *Model[E, S] {
		entity := item
		return s.newKeyedModel(&entity, key)
	}), nil
}

func (s *Store[E, S]) List(ctx context.Context, specs ...repository.Spec) ([]*Model[E, S], error) {
	items, err := s.repository.ListSpec(ctx, specs...)
	if err != nil {
		return nil, err
	}
	return lo.Map(items, func(item E, _ int) *Model[E, S] {
		entity := item
		return s.newKeyedModel(&entity, s.keyOf(&entity))
	}), nil
}

func (s *Store[E, S]) newModel(entity *E) *Model[E, S] {
	return &Model[E, S]{store: s, entity: entity}
}

func (s *Store[E, S]) newKeyedModel(entity *E, key repository.Key) *Model[E, S] {
	return &Model[E, S]{store: s, entity: entity, key: cloneKey(key)}
}

func (s *Store[E, S]) wrapOption(entity mo.Option[E], err error) (mo.Option[*Model[E, S]], error) {
	if err != nil {
		return mo.None[*Model[E, S]](), err
	}
	return mapOption(entity, func(item E) *Model[E, S] {
		entity := item
		return s.newKeyedModel(&entity, s.keyOf(&entity))
	}), nil
}

func mapOption[T any, R any](value mo.Option[T], mapper func(T) R) mo.Option[R] {
	item, ok := value.Get()
	if !ok {
		return mo.None[R]()
	}
	return mo.Some(mapper(item))
}
