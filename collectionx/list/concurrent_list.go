package list

import (
	"slices"
	"sync"

	"github.com/samber/lo"
	"github.com/samber/mo"
)

// ConcurrentList is a goroutine-safe strongly-typed list.
// Zero value is ready to use.
type ConcurrentList[T any] struct {
	mu    sync.RWMutex
	items []T
}

// NewConcurrentList creates a list and copies optional items.
func NewConcurrentList[T any](items ...T) *ConcurrentList[T] {
	l := &ConcurrentList[T]{}
	l.Add(items...)
	return l
}

// Add appends one or more items.
func (l *ConcurrentList[T]) Add(items ...T) {
	if l == nil || len(items) == 0 {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.items = append(l.items, items...)
}

// AddAt inserts one item at index. index == Len() is allowed.
func (l *ConcurrentList[T]) AddAt(index int, item T) bool {
	return l.AddAllAt(index, item)
}

// AddAllAt inserts items at index while preserving order.
func (l *ConcurrentList[T]) AddAllAt(index int, items ...T) bool {
	if l == nil {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	if index < 0 || index > len(l.items) {
		return false
	}
	if len(items) == 0 {
		return true
	}

	l.items = append(l.items, items...)
	copy(l.items[index+len(items):], l.items[index:len(l.items)-len(items)])
	copy(l.items[index:], items)
	return true
}

// Get returns item at index.
func (l *ConcurrentList[T]) Get(index int) (T, bool) {
	var zero T
	if l == nil {
		return zero, false
	}
	l.mu.RLock()
	defer l.mu.RUnlock()

	if index < 0 || index >= len(l.items) {
		return zero, false
	}
	return l.items[index], true
}

// GetOption returns item at index as mo.Option.
func (l *ConcurrentList[T]) GetOption(index int) mo.Option[T] {
	value, ok := l.Get(index)
	if !ok {
		return mo.None[T]()
	}
	return mo.Some(value)
}

// Set replaces item at index.
func (l *ConcurrentList[T]) Set(index int, item T) bool {
	if l == nil {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	if index < 0 || index >= len(l.items) {
		return false
	}
	l.items[index] = item
	return true
}

// RemoveAt removes and returns item at index.
func (l *ConcurrentList[T]) RemoveAt(index int) (T, bool) {
	var zero T
	if l == nil {
		return zero, false
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	if index < 0 || index >= len(l.items) {
		return zero, false
	}
	removed := l.items[index]
	copy(l.items[index:], l.items[index+1:])
	l.items[len(l.items)-1] = zero
	l.items = l.items[:len(l.items)-1]
	return removed, true
}

// RemoveAtOption removes item at index and returns it as mo.Option.
func (l *ConcurrentList[T]) RemoveAtOption(index int) mo.Option[T] {
	value, ok := l.RemoveAt(index)
	if !ok {
		return mo.None[T]()
	}
	return mo.Some(value)
}

// RemoveIf removes all items matched by predicate and returns removed count.
func (l *ConcurrentList[T]) RemoveIf(predicate func(item T) bool) int {
	if l == nil || predicate == nil {
		return 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.items) == 0 {
		return 0
	}

	next := lo.Filter(l.items, func(item T, _ int) bool {
		return !predicate(item)
	})
	removed := len(l.items) - len(next)
	l.items = next
	return removed
}

// Len returns item count.
func (l *ConcurrentList[T]) Len() int {
	if l == nil {
		return 0
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.items)
}

// IsEmpty reports whether list has no items.
func (l *ConcurrentList[T]) IsEmpty() bool {
	return l.Len() == 0
}

// Clear removes all items.
func (l *ConcurrentList[T]) Clear() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.items = nil
}

// Values returns a snapshot of items.
func (l *ConcurrentList[T]) Values() []T {
	if l == nil {
		return nil
	}
	l.mu.RLock()
	defer l.mu.RUnlock()

	if len(l.items) == 0 {
		return nil
	}
	return slices.Clone(l.items)
}

// Range iterates a stable snapshot from left to right until fn returns false.
func (l *ConcurrentList[T]) Range(fn func(index int, item T) bool) {
	if l == nil || fn == nil {
		return
	}
	for index, item := range l.Values() {
		if !fn(index, item) {
			return
		}
	}
}

// Snapshot returns an immutable-style copy in a normal List.
func (l *ConcurrentList[T]) Snapshot() *List[T] {
	return NewList(l.Values()...)
}
