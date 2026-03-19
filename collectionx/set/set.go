package set

import collectionmapping "github.com/DaiYuANg/arcgo/collectionx/mapping"

// Set is a generic hash set.
// Zero value is ready to use.
type Set[T comparable] struct {
	items collectionmapping.Map[T, struct{}]
}

// NewSet creates a new set and fills it with optional items.
func NewSet[T comparable](items ...T) *Set[T] {
	return NewSetWithCapacity(len(items), items...)
}

// NewSetWithCapacity creates a new set with preallocated capacity and optional items.
func NewSetWithCapacity[T comparable](capacity int, items ...T) *Set[T] {
	if capacity < len(items) {
		capacity = len(items)
	}
	s := &Set[T]{}
	if capacity > 0 {
		s.items = *collectionmapping.NewMapWithCapacity[T, struct{}](capacity)
	}
	s.Add(items...)
	return s
}

// Add inserts one or more items.
func (s *Set[T]) Add(items ...T) {
	if s == nil || len(items) == 0 {
		return
	}
	for _, item := range items {
		s.items.Set(item, struct{}{})
	}
}

// Merge inserts all items from other into set.
func (s *Set[T]) Merge(other *Set[T]) *Set[T] {
	if s == nil {
		return nil
	}
	if other == nil || other.items.Len() == 0 {
		return s
	}
	other.items.Range(func(item T, _ struct{}) bool {
		s.items.Set(item, struct{}{})
		return true
	})
	return s
}

// MergeSlice inserts all items from a slice into set.
func (s *Set[T]) MergeSlice(items []T) *Set[T] {
	if s == nil {
		return nil
	}
	s.Add(items...)
	return s
}

// Remove deletes an item and reports whether it existed.
func (s *Set[T]) Remove(item T) bool {
	if s == nil {
		return false
	}
	return s.items.Delete(item)
}

// Contains reports whether item exists.
func (s *Set[T]) Contains(item T) bool {
	if s == nil {
		return false
	}
	_, ok := s.items.Get(item)
	return ok
}

// Len returns total item count.
func (s *Set[T]) Len() int {
	if s == nil {
		return 0
	}
	return s.items.Len()
}

// IsEmpty reports whether the set has no items.
func (s *Set[T]) IsEmpty() bool {
	return s.Len() == 0
}

// Clear removes all items.
func (s *Set[T]) Clear() {
	if s == nil {
		return
	}
	s.items.Clear()
}

// Values returns all items as a slice.
func (s *Set[T]) Values() []T {
	if s == nil || s.items.Len() == 0 {
		return nil
	}
	return s.items.Keys()
}

// Range iterates all items until fn returns false.
func (s *Set[T]) Range(fn func(item T) bool) {
	if s == nil || fn == nil {
		return
	}
	s.items.Range(func(item T, _ struct{}) bool {
		return fn(item)
	})
}

// Clone returns a shallow copy.
func (s *Set[T]) Clone() *Set[T] {
	if s == nil || s.items.Len() == 0 {
		return &Set[T]{}
	}
	out := NewSetWithCapacity[T](s.Len())
	s.items.Range(func(item T, _ struct{}) bool {
		out.items.Set(item, struct{}{})
		return true
	})
	return out
}

// Union returns a new set that contains items from both sets.
func (s *Set[T]) Union(other *Set[T]) *Set[T] {
	out := s.Clone()
	if other == nil || other.items.Len() == 0 {
		return out
	}
	other.items.Range(func(item T, _ struct{}) bool {
		out.items.Set(item, struct{}{})
		return true
	})
	return out
}

// Intersect returns a new set that contains shared items.
func (s *Set[T]) Intersect(other *Set[T]) *Set[T] {
	out := &Set[T]{}
	if s == nil || other == nil || s.items.Len() == 0 || other.items.Len() == 0 {
		return out
	}

	left := &s.items
	right := &other.items
	if left.Len() > right.Len() {
		left, right = right, left
	}

	left.Range(func(item T, _ struct{}) bool {
		if _, ok := right.Get(item); ok {
			out.items.Set(item, struct{}{})
		}
		return true
	})
	return out
}

// Difference returns a new set with items in s but not in other.
func (s *Set[T]) Difference(other *Set[T]) *Set[T] {
	out := &Set[T]{}
	if s == nil || s.items.Len() == 0 {
		return out
	}

	if other == nil || other.items.Len() == 0 {
		s.items.Range(func(item T, _ struct{}) bool {
			out.items.Set(item, struct{}{})
			return true
		})
		return out
	}

	s.items.Range(func(item T, _ struct{}) bool {
		if _, ok := other.items.Get(item); !ok {
			out.items.Set(item, struct{}{})
		}
		return true
	})
	return out
}
