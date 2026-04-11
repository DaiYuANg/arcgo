//revive:disable:file-length-limit List methods are kept together to preserve the collection API surface.

package list

import (
	"slices"

	"github.com/samber/mo"
)

// List is a strongly-typed list backed by a slice.
// Zero value is ready to use.
type List[T any] struct {
	items []T

	jsonCache   []byte
	stringCache string
	jsonDirty   bool
}

// NewList creates a list and copies optional items.
func NewList[T any](items ...T) *List[T] {
	return NewListWithCapacity(len(items), items...)
}

// NewListWithCapacity creates a list with preallocated capacity and optional items.
func NewListWithCapacity[T any](capacity int, items ...T) *List[T] {
	if capacity < len(items) {
		capacity = len(items)
	}
	if capacity <= 0 {
		return &List[T]{}
	}

	l := &List[T]{
		items: make([]T, 0, capacity),
	}
	l.Add(items...)
	return l
}

// Add appends one or more items.
func (l *List[T]) Add(items ...T) {
	if l == nil || len(items) == 0 {
		return
	}
	l.items = append(l.items, items...)
	l.invalidateSerializationCache()
}

// Merge appends all items from other list.
func (l *List[T]) Merge(other *List[T]) *List[T] {
	if l == nil {
		return nil
	}
	if other == nil || len(other.items) == 0 {
		return l
	}
	l.items = append(l.items, other.items...)
	l.invalidateSerializationCache()
	return l
}

// MergeSlice appends all items from a slice.
func (l *List[T]) MergeSlice(items []T) *List[T] {
	if l == nil {
		return nil
	}
	l.Add(items...)
	return l
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
	l.invalidateSerializationCache()
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

// GetFirst returns the first item.
func (l *List[T]) GetFirst() (T, bool) {
	return l.Get(0)
}

// GetOption returns item at index as mo.Option.
func (l *List[T]) GetOption(index int) mo.Option[T] {
	value, ok := l.Get(index)
	if !ok {
		return mo.None[T]()
	}
	return mo.Some(value)
}

// GetFirstOption returns the first item as mo.Option.
func (l *List[T]) GetFirstOption() mo.Option[T] {
	return l.GetOption(0)
}

// GetLast returns the last item.
func (l *List[T]) GetLast() (T, bool) {
	return l.Get(l.Len() - 1)
}

// GetLastOption returns the last item as mo.Option.
func (l *List[T]) GetLastOption() mo.Option[T] {
	value, ok := l.GetLast()
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
	l.invalidateSerializationCache()
	return true
}

// SetAll applies mapper to each item and replaces all items in-place.
// Returns updated item count.
func (l *List[T]) SetAll(mapper func(item T) T) int {
	if mapper == nil {
		return 0
	}
	return l.SetAllIndexed(func(_ int, item T) T {
		return mapper(item)
	})
}

// SetAllIndexed applies mapper(index, item) to each item and replaces all items in-place.
// Returns updated item count.
func (l *List[T]) SetAllIndexed(mapper func(index int, item T) T) int {
	if l == nil || mapper == nil || len(l.items) == 0 {
		return 0
	}
	for index, item := range l.items {
		l.items[index] = mapper(index, item)
	}
	l.invalidateSerializationCache()
	return len(l.items)
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
	l.invalidateSerializationCache()
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

	writeIndex := 0
	for _, item := range l.items {
		if predicate(item) {
			continue
		}
		l.items[writeIndex] = item
		writeIndex++
	}

	removed := len(l.items) - writeIndex
	if removed == 0 {
		return 0
	}

	var zero T
	for index := writeIndex; index < len(l.items); index++ {
		l.items[index] = zero
	}
	l.items = l.items[:writeIndex]
	l.invalidateSerializationCache()
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
	l.jsonCache = nil
	l.stringCache = ""
	l.jsonDirty = false
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
	if l == nil || len(l.items) == 0 {
		return &List[T]{}
	}
	return &List[T]{items: slices.Clone(l.items)}
}

// Sort sorts items in place and returns the receiver for chaining.
func (l *List[T]) Sort(compare func(left, right T) int) *List[T] {
	if l == nil {
		return NewList[T]()
	}
	if compare == nil || len(l.items) < 2 {
		return l
	}
	slices.SortFunc(l.items, compare)
	l.invalidateSerializationCache()
	return l
}

func (l *List[T]) invalidateSerializationCache() {
	if l == nil {
		return
	}
	l.jsonCache = nil
	l.stringCache = ""
	l.jsonDirty = true
}

func (l *List[T]) cacheSerializationData(data []byte) {
	if l == nil {
		return
	}
	l.jsonCache = data
	l.stringCache = string(data)
	l.jsonDirty = false
}
