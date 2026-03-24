package activerecord

import (
	"context"
	"errors"

	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/DaiYuANg/arcgo/dbx/repository"
)

type Model[E any, S dbx.SchemaSource[E]] struct {
	store  *Store[E, S]
	entity *E
	key    repository.Key
}

func (m *Model[E, S]) Entity() *E {
	return m.entity
}

func (m *Model[E, S]) Key() repository.Key {
	if m.key == nil && m.entity != nil {
		m.key = m.store.keyOf(m.entity)
	}
	return cloneKey(m.key)
}

func (m *Model[E, S]) Save(ctx context.Context) error {
	if m == nil || m.store == nil || m.store.repository == nil {
		return dbx.ErrNilDB
	}
	if m.entity == nil {
		return &repository.ValidationError{Message: "entity is nil"}
	}
	key := m.Key()
	if len(key) == 0 || hasZeroKeyValue(key) {
		if err := m.store.repository.Create(ctx, m.entity); err != nil {
			return err
		}
		m.key = m.store.keyOf(m.entity)
		return nil
	}
	assignments, err := m.store.repository.Mapper().UpdateAssignments(m.store.repository.Schema(), m.entity)
	if err != nil {
		return err
	}
	if len(assignments) == 0 {
		return nil
	}
	_, err = m.store.repository.UpdateByKey(ctx, key, assignments...)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return m.store.repository.Create(ctx, m.entity)
		}
		return err
	}
	return nil
}

func (m *Model[E, S]) Reload(ctx context.Context) error {
	if m == nil || m.store == nil || m.store.repository == nil {
		return dbx.ErrNilDB
	}
	if m.entity == nil {
		return &repository.ValidationError{Message: "entity is nil"}
	}
	key := m.Key()
	if len(key) == 0 {
		return &repository.ValidationError{Message: "entity key is empty"}
	}
	latest, err := m.store.repository.GetByKey(ctx, key)
	if err != nil {
		return err
	}
	*m.entity = latest
	m.key = m.store.keyOf(m.entity)
	return nil
}

func (m *Model[E, S]) Delete(ctx context.Context) error {
	if m == nil || m.store == nil || m.store.repository == nil {
		return dbx.ErrNilDB
	}
	key := m.Key()
	if len(key) == 0 {
		return &repository.ValidationError{Message: "entity key is empty"}
	}
	_, err := m.store.repository.DeleteByKey(ctx, key)
	return err
}

