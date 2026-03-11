package list

import (
	"slices"

	"github.com/samber/lo"
	"github.com/samber/mo"
)

// List is a strongly-typed list backed by a slice.
// Zero value is ready to use.
type List[T any] struct {
	items []T
}

// NewList creates a list and copies optional items.
func NewList[T any](items ...T) *List[T] {
	l := &List[T]{}
	l.Add(items...)
	return l
}

// Add appends one or more items.
func (l *List[T]) Add(items ...T) {
	if l == nil || len(items) == 0 {
		return
	}
	l.items = append(l.items, items...)
}

// AddAt inserts one item at index. index == Len() is allowed.
func (l *List[T]) AddAt(index int, item T) bool {
	return l.AddAllAt(index, item)
}

// AddAllAt inserts items at index while preserving order.
func (l *List[T]) AddAllAt(index int, items ...T) bool {
	if l == nil {
		return false
	}
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
func (l *List[T]) Get(index int) (T, bool) {
	var zero T
	if l == nil || index < 0 || index >= len(l.items) {
		return zero, false
	}
	return l.items[index], true
}

// GetOption returns item at index as mo.Option.
func (l *List[T]) GetOption(index int) mo.Option[T] {
	value, ok := l.Get(index)
	if !ok {
		return mo.None[T]()
	}
	return mo.Some(value)
}

// Set replaces item at index.
func (l *List[T]) Set(index int, item T) bool {
	if l == nil || index < 0 || index >= len(l.items) {
		return false
	}
	l.items[index] = item
	return true
}

// RemoveAt removes and returns item at index.
func (l *List[T]) RemoveAt(index int) (T, bool) {
	var zero T
	if l == nil || index < 0 || index >= len(l.items) {
		return zero, false
	}
	removed := l.items[index]
	copy(l.items[index:], l.items[index+1:])
	l.items[len(l.items)-1] = zero
	l.items = l.items[:len(l.items)-1]
	return removed, true
}

// RemoveAtOption removes item at index and returns it as mo.Option.
func (l *List[T]) RemoveAtOption(index int) mo.Option[T] {
	value, ok := l.RemoveAt(index)
	if !ok {
		return mo.None[T]()
	}
	return mo.Some(value)
}

// RemoveIf removes all items matched by predicate and returns removed count.
func (l *List[T]) RemoveIf(predicate func(item T) bool) int {
	if l == nil || predicate == nil || len(l.items) == 0 {
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
func (l *List[T]) Len() int {
	if l == nil {
		return 0
	}
	return len(l.items)
}

// IsEmpty reports whether list has no items.
func (l *List[T]) IsEmpty() bool {
	return l.Len() == 0
}

// Clear removes all items.
func (l *List[T]) Clear() {
	if l == nil {
		return
	}
	l.items = nil
}

// Values returns a copy of items.
func (l *List[T]) Values() []T {
	if l == nil || len(l.items) == 0 {
		return nil
	}
	return slices.Clone(l.items)
}

// Range iterates list from left to right until fn returns false.
func (l *List[T]) Range(fn func(index int, item T) bool) {
	if l == nil || fn == nil {
		return
	}
	for index, item := range l.items {
		if !fn(index, item) {
			return
		}
	}
}

// Clone returns a shallow copy.
func (l *List[T]) Clone() *List[T] {
	return NewList(l.Values()...)
}
