package mapping

import (
	"slices"

	"github.com/samber/mo"
)

// MultiMap stores one key with multiple values.
// Zero value is ready to use.
type MultiMap[K comparable, V any] struct {
	items      Map[K, []V]
	valueCount int
}

// NewMultiMap creates an empty multimap.
func NewMultiMap[K comparable, V any]() *MultiMap[K, V] {
	return NewMultiMapWithCapacity[K, V](0)
}

// NewMultiMapWithCapacity creates an empty multimap with preallocated key capacity.
func NewMultiMapWithCapacity[K comparable, V any](capacity int) *MultiMap[K, V] {
	if capacity <= 0 {
		return &MultiMap[K, V]{}
	}
	return &MultiMap[K, V]{
		items: *NewMapWithCapacity[K, []V](capacity),
	}
}

// Put appends one value for key.
func (m *MultiMap[K, V]) Put(key K, value V) {
	m.PutAll(key, value)
}

// PutAll appends values for key.
func (m *MultiMap[K, V]) PutAll(key K, values ...V) {
	if m == nil || len(values) == 0 {
		return
	}
	m.ensureInit()

	current, _ := m.items.Get(key)
	next := make([]V, len(current)+len(values))
	copy(next, current)
	copy(next[len(current):], values)
	m.items.Set(key, next)
	m.valueCount += len(values)
}

// Set replaces all values for key.
// Passing no values removes the key.
func (m *MultiMap[K, V]) Set(key K, values ...V) {
	if m == nil {
		return
	}
	m.ensureInit()

	oldValues, _ := m.items.Get(key)
	oldCount := len(oldValues)
	if len(values) == 0 {
		m.items.Delete(key)
		m.valueCount -= oldCount
		return
	}
	m.items.Set(key, slices.Clone(values))
	m.valueCount += len(values) - oldCount
}

// Get returns a read-only slice view for key.
// Callers must not modify the returned slice.
func (m *MultiMap[K, V]) Get(key K) []V {
	if m == nil {
		return nil
	}
	values, ok := m.items.Get(key)
	if !ok || len(values) == 0 {
		return nil
	}
	return values
}

// GetCopy returns an owned copy of values for key.
func (m *MultiMap[K, V]) GetCopy(key K) []V {
	values := m.Get(key)
	if len(values) == 0 {
		return nil
	}
	return slices.Clone(values)
}

// GetOption returns values for key as mo.Option.
func (m *MultiMap[K, V]) GetOption(key K) mo.Option[[]V] {
	values := m.Get(key)
	if len(values) == 0 {
		return mo.None[[]V]()
	}
	return mo.Some(values)
}

// Delete removes all values for key.
func (m *MultiMap[K, V]) Delete(key K) bool {
	if m == nil {
		return false
	}
	values, existed := m.items.Get(key)
	if existed {
		m.items.Delete(key)
		m.valueCount -= len(values)
	}
	return existed
}

// DeleteValueIf removes values matching predicate under key and returns removed count.
func (m *MultiMap[K, V]) DeleteValueIf(key K, predicate func(value V) bool) int {
	if m == nil || predicate == nil {
		return 0
	}

	values, ok := m.items.Get(key)
	if !ok || len(values) == 0 {
		return 0
	}

	next := make([]V, 0, len(values))
	for _, value := range values {
		if predicate(value) {
			continue
		}
		next = append(next, value)
	}

	removed := len(values) - len(next)
	if removed == 0 {
		return 0
	}

	if len(next) == 0 {
		m.items.Delete(key)
	} else {
		m.items.Set(key, next)
	}
	m.valueCount -= removed
	return removed
}

// ContainsKey reports whether key exists.
func (m *MultiMap[K, V]) ContainsKey(key K) bool {
	if m == nil {
		return false
	}
	_, ok := m.items.Get(key)
	return ok
}

// Len returns key count.
func (m *MultiMap[K, V]) Len() int {
	if m == nil {
		return 0
	}
	return m.items.Len()
}

// ValueCount returns total stored value count.
func (m *MultiMap[K, V]) ValueCount() int {
	if m == nil {
		return 0
	}
	return m.valueCount
}

// IsEmpty reports whether map has no keys.
func (m *MultiMap[K, V]) IsEmpty() bool {
	return m.Len() == 0
}

// Clear removes all entries.
func (m *MultiMap[K, V]) Clear() {
	if m == nil {
		return
	}
	m.items.Clear()
	m.valueCount = 0
}

// Keys returns all keys.
func (m *MultiMap[K, V]) Keys() []K {
	if m == nil || m.items.Len() == 0 {
		return nil
	}
	return m.items.Keys()
}

// All returns a deep-copied built-in map.
func (m *MultiMap[K, V]) All() map[K][]V {
	if m == nil || m.items.Len() == 0 {
		return map[K][]V{}
	}
	out := make(map[K][]V, m.items.Len())
	m.items.Range(func(key K, values []V) bool {
		out[key] = slices.Clone(values)
		return true
	})
	return out
}

// Range iterates key-values snapshots until fn returns false.
func (m *MultiMap[K, V]) Range(fn func(key K, values []V) bool) {
	if m == nil || fn == nil {
		return
	}
	m.items.Range(func(key K, values []V) bool {
		return fn(key, slices.Clone(values))
	})
}

func (m *MultiMap[K, V]) ensureInit() {
	m.items.ensureInit()
}
