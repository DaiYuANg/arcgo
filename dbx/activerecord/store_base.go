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
	return &Model[E, S]{store: s, entity: entity}
}

func (s *Store[E, S]) FindByID(ctx context.Context, id any) (*Model[E, S], error) {
	entity, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &Model[E, S]{store: s, entity: &entity, key: s.keyOf(&entity)}, nil
}

func (s *Store[E, S]) FindByIDOption(ctx context.Context, id any) (mo.Option[*Model[E, S]], error) {
	entity, err := s.repository.GetByIDOption(ctx, id)
	if err != nil {
		return mo.None[*Model[E, S]](), err
	}
	item, ok := entity.Get()
	if !ok {
		return mo.None[*Model[E, S]](), nil
	}
	e := item
	return mo.Some(&Model[E, S]{store: s, entity: &e, key: s.keyOf(&e)}), nil
}

func (s *Store[E, S]) FindByKey(ctx context.Context, key repository.Key) (*Model[E, S], error) {
	entity, err := s.repository.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	return &Model[E, S]{store: s, entity: &entity, key: key}, nil
}

func (s *Store[E, S]) FindByKeyOption(ctx context.Context, key repository.Key) (mo.Option[*Model[E, S]], error) {
	entity, err := s.repository.GetByKeyOption(ctx, key)
	if err != nil {
		return mo.None[*Model[E, S]](), err
	}
	item, ok := entity.Get()
	if !ok {
		return mo.None[*Model[E, S]](), nil
	}
	e := item
	return mo.Some(&Model[E, S]{store: s, entity: &e, key: s.keyOf(&e)}), nil
}

func (s *Store[E, S]) List(ctx context.Context, specs ...repository.Spec) ([]*Model[E, S], error) {
	items, err := s.repository.ListSpec(ctx, specs...)
	if err != nil {
		return nil, err
	}
	models := lo.Map(items, func(item E, _ int) *Model[E, S] {
		entity := item
		return &Model[E, S]{store: s, entity: &entity, key: s.keyOf(&entity)}
	})
	return models, nil
}

