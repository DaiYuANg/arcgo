package set

import (
	collectionlist "github.com/DaiYuANg/arcgo/collectionx/list"
	collectionmapping "github.com/DaiYuANg/arcgo/collectionx/mapping"
)

// OrderedSet keeps insertion order of unique items.
// Zero value is ready to use.
type OrderedSet[T comparable] struct {
	order collectionlist.List[T]
	items collectionmapping.Map[T, struct{}]
	index collectionmapping.Map[T, int]
}

// NewOrderedSet creates an ordered set with optional items.
func NewOrderedSet[T comparable](items ...T) *OrderedSet[T] {
	return NewOrderedSetWithCapacity(len(items), items...)
}

// NewOrderedSetWithCapacity creates an ordered set with preallocated capacity and optional items.
func NewOrderedSetWithCapacity[T comparable](capacity int, items ...T) *OrderedSet[T] {
	if capacity < len(items) {
		capacity = len(items)
	}
	s := &OrderedSet[T]{}
	if capacity > 0 {
		s.order = *collectionlist.NewListWithCapacity[T](capacity)
		s.items = *collectionmapping.NewMapWithCapacity[T, struct{}](capacity)
		s.index = *collectionmapping.NewMapWithCapacity[T, int](capacity)
	}
	s.Add(items...)
	return s
}

// Add inserts one or more items.
func (s *OrderedSet[T]) Add(items ...T) {
	if s == nil || len(items) == 0 {
		return
	}

	for _, item := range items {
		if _, exists := s.items.Get(item); exists {
			continue
		}
		s.order.Add(item)
		s.items.Set(item, struct{}{})
		s.index.Set(item, s.order.Len()-1)
	}
}

// Remove deletes item and reports whether it existed.
func (s *OrderedSet[T]) Remove(item T) bool {
	if s == nil {
		return false
	}
	pos, ok := s.index.Get(item)
	if !ok {
		return false
	}

	s.items.Delete(item)
	s.index.Delete(item)

	_, _ = s.order.RemoveAt(pos)
	for i := pos; i < s.order.Len(); i++ {
		nextItem, _ := s.order.Get(i)
		s.index.Set(nextItem, i)
	}
	return true
}

// Contains reports whether item exists.
func (s *OrderedSet[T]) Contains(item T) bool {
	if s == nil {
		return false
	}
	_, ok := s.items.Get(item)
	return ok
}

// Len returns item count.
func (s *OrderedSet[T]) Len() int {
	if s == nil {
		return 0
	}
	return s.order.Len()
}

// IsEmpty reports whether set has no items.
func (s *OrderedSet[T]) IsEmpty() bool {
	return s.Len() == 0
}

// Clear removes all items.
func (s *OrderedSet[T]) Clear() {
	if s == nil {
		return
	}
	s.order.Clear()
	s.items.Clear()
	s.index.Clear()
}

// Values returns items in insertion order.
func (s *OrderedSet[T]) Values() []T {
	if s == nil {
		return nil
	}
	values := s.order.Values()
	if len(values) == 0 {
		return nil
	}
	return values
}

// At returns item at insertion index.
func (s *OrderedSet[T]) At(pos int) (T, bool) {
	if s == nil {
		var zero T
		return zero, false
	}
	return s.order.Get(pos)
}

// Range iterates items in insertion order until fn returns false.
func (s *OrderedSet[T]) Range(fn func(item T) bool) {
	if s == nil || fn == nil {
		return
	}
	s.order.Range(func(_ int, item T) bool { return fn(item) })
}

// Clone returns a shallow copy.
func (s *OrderedSet[T]) Clone() *OrderedSet[T] {
	if s == nil {
		return NewOrderedSet[T]()
	}
	out := NewOrderedSetWithCapacity[T](s.order.Len())
	s.order.Range(func(_ int, item T) bool {
		out.order.Add(item)
		return true
	})
	s.items.Range(func(item T, _ struct{}) bool {
		out.items.Set(item, struct{}{})
		return true
	})
	s.index.Range(func(item T, index int) bool {
		out.index.Set(item, index)
		return true
	})
	return out
}
