package set

import (
	collectionlist "github.com/DaiYuANg/archgo/collectionx/list"
	collectionmapping "github.com/DaiYuANg/archgo/collectionx/mapping"
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
	s := &OrderedSet[T]{}
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
	out := &OrderedSet[T]{}
	if s == nil {
		return out
	}
	out.order.Add(s.order.Values()...)
	out.items.SetAll(s.items.All())
	out.index.SetAll(s.index.All())
	return out
}
