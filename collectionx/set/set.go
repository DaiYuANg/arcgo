package set

import (
	collectionmapping "github.com/DaiYuANg/arcgo/collectionx/mapping"
	"github.com/samber/lo"
	"github.com/samber/mo"
)

// Set is a generic hash set.
// Zero value is ready to use.
type Set[T comparable] struct {
	items collectionmapping.Map[T, struct{}]

	jsonCache   []byte
	stringCache string
	jsonDirty   bool
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
	lo.ForEach(items, func(item T, _ int) {
		s.items.Set(item, struct{}{})
	})
	s.invalidateSerializationCache()
}

// Merge inserts all items from other into set.
func (s *Set[T]) Merge(other *Set[T]) *Set[T] {
	if s == nil {
		return nil
	}
	if other == nil || other.items.Len() == 0 {
		return s
	}
	other.Range(func(item T) bool {
		s.items.Set(item, struct{}{})
		return true
	})
	s.invalidateSerializationCache()
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
	removed := s.items.Delete(item)
	if removed {
		s.invalidateSerializationCache()
	}
	return removed
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
	s.jsonCache = nil
	s.stringCache = ""
	s.jsonDirty = false
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
	s.Range(func(item T) bool {
		out.items.Set(item, struct{}{})
		return true
	})
	return out
}

// Where returns a new set containing only items that match predicate.
func (s *Set[T]) Where(predicate func(item T) bool) *Set[T] {
	if s == nil || predicate == nil || s.items.Len() == 0 {
		return NewSet[T]()
	}
	filtered := NewSetWithCapacity[T](s.Len())
	s.Range(func(item T) bool {
		if predicate(item) {
			filtered.items.Set(item, struct{}{})
		}
		return true
	})
	return filtered
}

// Reject returns a new set excluding items that match predicate.
func (s *Set[T]) Reject(predicate func(item T) bool) *Set[T] {
	if s == nil || predicate == nil || s.items.Len() == 0 {
		return NewSet[T]()
	}
	rejected := NewSetWithCapacity[T](s.Len())
	s.Range(func(item T) bool {
		if !predicate(item) {
			rejected.items.Set(item, struct{}{})
		}
		return true
	})
	return rejected
}

// Each invokes fn for every item and returns the receiver for chaining.
func (s *Set[T]) Each(fn func(item T)) *Set[T] {
	if s == nil {
		return NewSet[T]()
	}
	if fn == nil {
		return s
	}
	s.Range(func(item T) bool {
		fn(item)
		return true
	})
	return s
}

// FirstWhere returns the first item matching predicate.
func (s *Set[T]) FirstWhere(predicate func(item T) bool) mo.Option[T] {
	if s == nil || predicate == nil || s.items.Len() == 0 {
		return mo.None[T]()
	}
	var found T
	ok := false
	s.Range(func(item T) bool {
		if !predicate(item) {
			return true
		}
		found = item
		ok = true
		return false
	})
	if !ok {
		return mo.None[T]()
	}
	return mo.Some(found)
}

// AnyMatch reports whether any item matches predicate.
func (s *Set[T]) AnyMatch(predicate func(item T) bool) bool {
	_, ok := s.FirstWhere(predicate).Get()
	return ok
}

// AllMatch reports whether all items match predicate.
func (s *Set[T]) AllMatch(predicate func(item T) bool) bool {
	if s == nil || s.items.Len() == 0 || predicate == nil {
		return false
	}
	matched := true
	s.Range(func(item T) bool {
		if predicate(item) {
			return true
		}
		matched = false
		return false
	})
	return matched
}

// Union returns a new set that contains items from both sets.
func (s *Set[T]) Union(other *Set[T]) *Set[T] {
	out := s.Clone()
	if other == nil || other.items.Len() == 0 {
		return out
	}
	other.Range(func(item T) bool {
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

	out = NewSetWithCapacity[T](left.Len())
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
		return s.Clone()
	}

	out = NewSetWithCapacity[T](s.Len())
	s.items.Range(func(item T, _ struct{}) bool {
		if _, ok := other.items.Get(item); !ok {
			out.items.Set(item, struct{}{})
		}
		return true
	})
	return out
}

func (s *Set[T]) invalidateSerializationCache() {
	if s == nil {
		return
	}
	s.jsonCache = nil
	s.stringCache = ""
	s.jsonDirty = true
}

func (s *Set[T]) cacheSerializationData(data []byte) {
	if s == nil {
		return
	}
	s.jsonCache = data
	s.stringCache = string(data)
	s.jsonDirty = false
}
