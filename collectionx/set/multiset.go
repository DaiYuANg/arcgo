package set

import (
	collectionmapping "github.com/DaiYuANg/arcgo/collectionx/mapping"
	"github.com/samber/lo"
)

// MultiSet is a bag-like set with occurrence counts.
// Zero value is ready to use.
type MultiSet[T comparable] struct {
	counts collectionmapping.Map[T, int]
	size   int
}

// NewMultiSet creates a multiset with optional items.
func NewMultiSet[T comparable](items ...T) *MultiSet[T] {
	return NewMultiSetWithCapacity(len(items), items...)
}

// NewMultiSetWithCapacity creates a multiset with preallocated capacity and optional items.
func NewMultiSetWithCapacity[T comparable](capacity int, items ...T) *MultiSet[T] {
	if capacity < len(items) {
		capacity = len(items)
	}
	s := &MultiSet[T]{}
	if capacity > 0 {
		s.counts = *collectionmapping.NewMapWithCapacity[T, int](capacity)
	}
	s.Add(items...)
	return s
}

// Add inserts items with count +1 each.
func (s *MultiSet[T]) Add(items ...T) {
	if s == nil || len(items) == 0 {
		return
	}
	lo.ForEach(items, func(item T, _ int) {
		current, _ := s.counts.Get(item)
		s.counts.Set(item, current+1)
		s.size++
	})
}

// AddN inserts item n times. n <= 0 does nothing.
func (s *MultiSet[T]) AddN(item T, n int) {
	if s == nil || n <= 0 {
		return
	}
	current, _ := s.counts.Get(item)
	s.counts.Set(item, current+n)
	s.size += n
}

// Remove removes one occurrence.
func (s *MultiSet[T]) Remove(item T) bool {
	return s.RemoveN(item, 1) > 0
}

// RemoveN removes up to n occurrences and returns removed count.
func (s *MultiSet[T]) RemoveN(item T, n int) int {
	if s == nil || n <= 0 {
		return 0
	}
	current, ok := s.counts.Get(item)
	if !ok || current <= 0 {
		return 0
	}

	removed := min(n, current)

	remain := current - removed
	if remain == 0 {
		s.counts.Delete(item)
	} else {
		s.counts.Set(item, remain)
	}
	s.size -= removed
	return removed
}

// Count returns occurrence count for item.
func (s *MultiSet[T]) Count(item T) int {
	if s == nil {
		return 0
	}
	value, _ := s.counts.Get(item)
	return value
}

// Contains reports whether item exists.
func (s *MultiSet[T]) Contains(item T) bool {
	return s.Count(item) > 0
}

// Len returns total occurrence count.
func (s *MultiSet[T]) Len() int {
	if s == nil {
		return 0
	}
	return s.size
}

// UniqueLen returns distinct key count.
func (s *MultiSet[T]) UniqueLen() int {
	if s == nil {
		return 0
	}
	return s.counts.Len()
}

// IsEmpty reports whether multiset has no elements.
func (s *MultiSet[T]) IsEmpty() bool {
	return s.Len() == 0
}

// Clear removes all elements.
func (s *MultiSet[T]) Clear() {
	if s == nil {
		return
	}
	s.counts.Clear()
	s.size = 0
}

// Distinct returns all distinct elements.
func (s *MultiSet[T]) Distinct() []T {
	if s == nil || s.counts.Len() == 0 {
		return nil
	}
	return s.counts.Keys()
}

// Elements returns flattened elements with duplicates.
func (s *MultiSet[T]) Elements() []T {
	if s == nil || s.size == 0 {
		return nil
	}
	out := make([]T, 0, s.size)
	s.counts.Range(func(item T, count int) bool {
		for range count {
			out = append(out, item)
		}
		return true
	})
	return out
}

// AllCounts returns copied count map.
func (s *MultiSet[T]) AllCounts() map[T]int {
	if s == nil || s.counts.Len() == 0 {
		return map[T]int{}
	}
	return s.counts.All()
}

// Range iterates all distinct elements with their counts until fn returns false.
func (s *MultiSet[T]) Range(fn func(item T, count int) bool) {
	if s == nil || fn == nil {
		return
	}
	s.counts.Range(fn)
}
