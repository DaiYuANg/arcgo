//revive:disable:file-length-limit Ordered set methods are kept together to preserve the collection API surface.

package set

import (
	collectionlist "github.com/DaiYuANg/arcgo/collectionx/list"
	collectionmapping "github.com/DaiYuANg/arcgo/collectionx/mapping"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"slices"
)

// OrderedSet keeps insertion order of unique items.
// Zero value is ready to use.
type OrderedSet[T comparable] struct {
	order collectionlist.List[T]
	items collectionmapping.Map[T, struct{}]
	index collectionmapping.Map[T, int]

	valuesCache []T
	valuesDirty bool
	jsonCache   []byte
	stringCache string
	jsonDirty   bool
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

	lo.ForEach(items, func(item T, _ int) {
		if _, exists := s.items.Get(item); exists {
			return
		}
		s.order.Add(item)
		s.items.Set(item, struct{}{})
		s.index.Set(item, s.order.Len()-1)
	})
	s.invalidateValuesCache()
	s.invalidateSerializationCache()
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
	s.invalidateValuesCache()
	s.invalidateSerializationCache()
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
	s.valuesCache = nil
	s.valuesDirty = false
	s.jsonCache = nil
	s.stringCache = ""
	s.jsonDirty = false
}

// Values returns items in insertion order.
func (s *OrderedSet[T]) Values() []T {
	if s == nil {
		return nil
	}
	if !s.valuesDirty && len(s.valuesCache) > 0 {
		return slices.Clone(s.valuesCache)
	}
	values := s.order.Values()
	if len(values) == 0 {
		return nil
	}
	s.valuesCache = values
	s.valuesDirty = false
	return slices.Clone(values)
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
	index := 0
	s.order.Range(func(_ int, item T) bool {
		out.order.Add(item)
		out.items.Set(item, struct{}{})
		out.index.Set(item, index)
		index++
		return true
	})
	return out
}

// Where returns a new ordered set containing only items that match predicate.
func (s *OrderedSet[T]) Where(predicate func(item T) bool) *OrderedSet[T] {
	if s == nil || predicate == nil || s.order.Len() == 0 {
		return NewOrderedSet[T]()
	}
	filtered := NewOrderedSetWithCapacity[T](s.order.Len())
	index := 0
	s.Range(func(item T) bool {
		if predicate(item) {
			filtered.order.Add(item)
			filtered.items.Set(item, struct{}{})
			filtered.index.Set(item, index)
			index++
		}
		return true
	})
	return filtered
}

// Reject returns a new ordered set excluding items that match predicate.
func (s *OrderedSet[T]) Reject(predicate func(item T) bool) *OrderedSet[T] {
	if s == nil || predicate == nil || s.order.Len() == 0 {
		return NewOrderedSet[T]()
	}
	rejected := NewOrderedSetWithCapacity[T](s.order.Len())
	index := 0
	s.Range(func(item T) bool {
		if !predicate(item) {
			rejected.order.Add(item)
			rejected.items.Set(item, struct{}{})
			rejected.index.Set(item, index)
			index++
		}
		return true
	})
	return rejected
}

// Take returns the first n items as a new ordered set.
func (s *OrderedSet[T]) Take(n int) *OrderedSet[T] {
	if s == nil || n <= 0 || s.order.Len() == 0 {
		return NewOrderedSet[T]()
	}
	if n >= s.order.Len() {
		return s.Clone()
	}
	out := NewOrderedSetWithCapacity[T](n)
	index := 0
	s.order.Range(func(_ int, item T) bool {
		out.order.Add(item)
		out.items.Set(item, struct{}{})
		out.index.Set(item, index)
		index++
		return index < n
	})
	return out
}

// Drop returns a new ordered set without the first n items.
func (s *OrderedSet[T]) Drop(n int) *OrderedSet[T] {
	if s == nil || s.order.Len() == 0 {
		return NewOrderedSet[T]()
	}
	if n <= 0 {
		return s.Clone()
	}
	if n >= s.order.Len() {
		return NewOrderedSet[T]()
	}
	out := NewOrderedSetWithCapacity[T](s.order.Len() - n)
	index := 0
	skipped := 0
	s.order.Range(func(_ int, item T) bool {
		if skipped < n {
			skipped++
			return true
		}
		out.order.Add(item)
		out.items.Set(item, struct{}{})
		out.index.Set(item, index)
		index++
		return true
	})
	return out
}

// Each invokes fn for every item and returns the receiver for chaining.
func (s *OrderedSet[T]) Each(fn func(item T)) *OrderedSet[T] {
	if s == nil {
		return NewOrderedSet[T]()
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
func (s *OrderedSet[T]) FirstWhere(predicate func(item T) bool) mo.Option[T] {
	if s == nil || predicate == nil || s.order.Len() == 0 {
		return mo.None[T]()
	}
	for i := range s.order.Len() {
		item, _ := s.order.Get(i)
		if predicate(item) {
			return mo.Some(item)
		}
	}
	return mo.None[T]()
}

// AnyMatch reports whether any item matches predicate.
func (s *OrderedSet[T]) AnyMatch(predicate func(item T) bool) bool {
	_, ok := s.FirstWhere(predicate).Get()
	return ok
}

// AllMatch reports whether all items match predicate.
func (s *OrderedSet[T]) AllMatch(predicate func(item T) bool) bool {
	if s == nil || s.order.Len() == 0 || predicate == nil {
		return false
	}
	for i := range s.order.Len() {
		item, _ := s.order.Get(i)
		if !predicate(item) {
			return false
		}
	}
	return true
}

func (s *OrderedSet[T]) invalidateValuesCache() {
	if s == nil {
		return
	}
	s.valuesCache = nil
	s.valuesDirty = true
}

func (s *OrderedSet[T]) invalidateSerializationCache() {
	if s == nil {
		return
	}
	s.jsonCache = nil
	s.stringCache = ""
	s.jsonDirty = true
}

func (s *OrderedSet[T]) cacheSerializationData(data []byte) {
	if s == nil {
		return
	}
	s.jsonCache = data
	s.stringCache = string(data)
	s.jsonDirty = false
}
