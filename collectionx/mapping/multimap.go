package mapping

import (
	"slices"

	"github.com/samber/lo"
	"github.com/samber/mo"
)

// MultiMap stores one key with multiple values.
// Zero value is ready to use.
type MultiMap[K comparable, V any] struct {
	items      map[K][]V
	valueCount int
}

// NewMultiMap creates an empty multimap.
func NewMultiMap[K comparable, V any]() *MultiMap[K, V] {
	return &MultiMap[K, V]{
		items: make(map[K][]V),
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
	m.items[key] = append(m.items[key], values...)
	m.valueCount += len(values)
}

// Set replaces all values for key.
// Passing no values removes the key.
func (m *MultiMap[K, V]) Set(key K, values ...V) {
	if m == nil {
		return
	}
	m.ensureInit()

	oldCount := len(m.items[key])
	if len(values) == 0 {
		delete(m.items, key)
		m.valueCount -= oldCount
		return
	}
	m.items[key] = slices.Clone(values)
	m.valueCount += len(values) - oldCount
}

// Get returns a copy of values for key.
func (m *MultiMap[K, V]) Get(key K) []V {
	if m == nil || m.items == nil {
		return nil
	}
	values, ok := m.items[key]
	if !ok || len(values) == 0 {
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
	if m == nil || m.items == nil {
		return false
	}
	values, existed := m.items[key]
	if existed {
		delete(m.items, key)
		m.valueCount -= len(values)
	}
	return existed
}

// DeleteValueIf removes values matching predicate under key and returns removed count.
func (m *MultiMap[K, V]) DeleteValueIf(key K, predicate func(value V) bool) int {
	if m == nil || m.items == nil || predicate == nil {
		return 0
	}

	values, ok := m.items[key]
	if !ok || len(values) == 0 {
		return 0
	}

	next := lo.Filter(values, func(item V, _ int) bool {
		return !predicate(item)
	})
	removed := len(values) - len(next)
	if len(next) == 0 {
		delete(m.items, key)
	} else {
		m.items[key] = next
	}
	m.valueCount -= removed
	return removed
}

// ContainsKey reports whether key exists.
func (m *MultiMap[K, V]) ContainsKey(key K) bool {
	if m == nil || m.items == nil {
		return false
	}
	_, ok := m.items[key]
	return ok
}

// Len returns key count.
func (m *MultiMap[K, V]) Len() int {
	if m == nil {
		return 0
	}
	return len(m.items)
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
	clear(m.items)
	m.valueCount = 0
}

// Keys returns all keys.
func (m *MultiMap[K, V]) Keys() []K {
	if m == nil || len(m.items) == 0 {
		return nil
	}
	return lo.Keys(m.items)
}

// All returns a deep-copied built-in map.
func (m *MultiMap[K, V]) All() map[K][]V {
	if m == nil || len(m.items) == 0 {
		return map[K][]V{}
	}
	out := make(map[K][]V, len(m.items))
	for key, values := range m.items {
		out[key] = slices.Clone(values)
	}
	return out
}

// Range iterates key-values snapshots until fn returns false.
func (m *MultiMap[K, V]) Range(fn func(key K, values []V) bool) {
	if m == nil || fn == nil {
		return
	}
	for key, values := range m.items {
		if !fn(key, slices.Clone(values)) {
			return
		}
	}
}

func (m *MultiMap[K, V]) ensureInit() {
	if m.items == nil {
		m.items = make(map[K][]V)
	}
}
