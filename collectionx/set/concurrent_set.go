package set

import (
	"sync"

	"github.com/samber/lo"
)

// ConcurrentSet is a goroutine-safe set.
// Zero value is ready to use.
type ConcurrentSet[T comparable] struct {
	mu   sync.RWMutex
	core *Set[T]
}

// NewConcurrentSet creates a new concurrent set.
func NewConcurrentSet[T comparable](items ...T) *ConcurrentSet[T] {
	return NewConcurrentSetWithCapacity(len(items), items...)
}

// NewConcurrentSetWithCapacity creates a new concurrent set with preallocated capacity.
func NewConcurrentSetWithCapacity[T comparable](capacity int, items ...T) *ConcurrentSet[T] {
	if capacity < len(items) {
		capacity = len(items)
	}
	if capacity <= 0 {
		return &ConcurrentSet[T]{}
	}
	return &ConcurrentSet[T]{
		core: NewSetWithCapacity[T](capacity, items...),
	}
}

// Add inserts one or more items.
func (s *ConcurrentSet[T]) Add(items ...T) {
	if s == nil || len(items) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureInitLocked()
	s.core.Add(items...)
}

// Merge inserts all items from a normal set.
func (s *ConcurrentSet[T]) Merge(other *Set[T]) *ConcurrentSet[T] {
	if s == nil {
		return nil
	}
	if other == nil {
		return s
	}
	s.Add(other.Values()...)
	return s
}

// MergeConcurrent inserts all items from another concurrent set snapshot.
func (s *ConcurrentSet[T]) MergeConcurrent(other *ConcurrentSet[T]) *ConcurrentSet[T] {
	if s == nil {
		return nil
	}
	if other == nil {
		return s
	}
	s.Add(other.Values()...)
	return s
}

// MergeSlice inserts all items from a slice.
func (s *ConcurrentSet[T]) MergeSlice(items []T) *ConcurrentSet[T] {
	if s == nil {
		return nil
	}
	s.Add(items...)
	return s
}

// AddIfAbsent inserts one item only when it does not exist.
// Returns true when inserted, false when it already exists.
func (s *ConcurrentSet[T]) AddIfAbsent(item T) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureInitLocked()
	if s.core.Contains(item) {
		return false
	}
	s.core.Add(item)
	return true
}

// Remove deletes an item and reports whether it existed.
func (s *ConcurrentSet[T]) Remove(item T) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.core == nil {
		return false
	}
	return s.core.Remove(item)
}

// Contains reports whether item exists.
func (s *ConcurrentSet[T]) Contains(item T) bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.core == nil {
		return false
	}
	return s.core.Contains(item)
}

// Len returns total item count.
func (s *ConcurrentSet[T]) Len() int {
	if s == nil {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.core == nil {
		return 0
	}
	return s.core.Len()
}

// IsEmpty reports whether set has no items.
func (s *ConcurrentSet[T]) IsEmpty() bool {
	return s.Len() == 0
}

// Clear removes all items.
func (s *ConcurrentSet[T]) Clear() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.core == nil {
		return
	}
	s.core.Clear()
}

// Values returns a snapshot of all items.
func (s *ConcurrentSet[T]) Values() []T {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.core == nil {
		return nil
	}
	return s.core.Values()
}

// Range iterates a stable snapshot until fn returns false.
func (s *ConcurrentSet[T]) Range(fn func(item T) bool) {
	if s == nil || fn == nil {
		return
	}
	lo.EveryBy(s.Values(), func(item T) bool {
		return fn(item)
	})
}

// Snapshot returns an immutable-style copy in a normal Set.
func (s *ConcurrentSet[T]) Snapshot() *Set[T] {
	out := &Set[T]{}
	if s == nil {
		return out
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.core == nil {
		return out
	}
	return s.core.Clone()
}

func (s *ConcurrentSet[T]) ensureInitLocked() {
	if s.core == nil {
		s.core = &Set[T]{}
	}
}
