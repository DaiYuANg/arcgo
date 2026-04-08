package mapping

import (
	"maps"

	"github.com/samber/mo"
)

// Map is a strongly-typed map wrapper.
// Zero value is ready to use.
type Map[K comparable, V any] struct {
	items map[K]V

	jsonCache   []byte
	stringCache string
	jsonDirty   bool
}

// NewMap creates an empty map.
func NewMap[K comparable, V any]() *Map[K, V] {
	return NewMapWithCapacity[K, V](0)
}

// NewMapWithCapacity creates an empty map with preallocated capacity.
func NewMapWithCapacity[K comparable, V any](capacity int) *Map[K, V] {
	if capacity < 0 {
		capacity = 0
	}
	return &Map[K, V]{
		items: make(map[K]V, capacity),
	}
}

// NewMapFrom creates a map from source and copies all entries.
func NewMapFrom[K comparable, V any](source map[K]V) *Map[K, V] {
	m := NewMapWithCapacity[K, V](len(source))
	m.SetAll(source)
	return m
}

// Set puts a key-value pair.
func (m *Map[K, V]) Set(key K, value V) {
	if m == nil {
		return
	}
	m.ensureInit()
	m.items[key] = value
	m.invalidateSerializationCache()
}

// SetAll copies all entries from source.
func (m *Map[K, V]) SetAll(source map[K]V) {
	if m == nil || len(source) == 0 {
		return
	}
	m.ensureInit()
	maps.Copy(m.items, source)
	m.invalidateSerializationCache()
}

// Get returns the value for key.
func (m *Map[K, V]) Get(key K) (V, bool) {
	var zero V
	if m == nil || m.items == nil {
		return zero, false
	}
	v, ok := m.items[key]
	return v, ok
}

// GetOption returns value for key as mo.Option.
func (m *Map[K, V]) GetOption(key K) mo.Option[V] {
	value, ok := m.Get(key)
	if !ok {
		return mo.None[V]()
	}
	return mo.Some(value)
}

// GetOrDefault returns value for key or fallback when key does not exist.
func (m *Map[K, V]) GetOrDefault(key K, fallback V) V {
	v, ok := m.Get(key)
	if !ok {
		return fallback
	}
	return v
}

// Delete removes key and reports whether it existed.
func (m *Map[K, V]) Delete(key K) bool {
	if m == nil || m.items == nil {
		return false
	}
	_, existed := m.items[key]
	if existed {
		delete(m.items, key)
		m.invalidateSerializationCache()
	}
	return existed
}

// Len returns total entry count.
func (m *Map[K, V]) Len() int {
	if m == nil {
		return 0
	}
	return len(m.items)
}

// IsEmpty reports whether map has no entries.
func (m *Map[K, V]) IsEmpty() bool {
	return m.Len() == 0
}

// Clear removes all entries.
func (m *Map[K, V]) Clear() {
	if m == nil {
		return
	}
	clear(m.items)
	m.jsonCache = nil
	m.stringCache = ""
	m.jsonDirty = false
}

// Keys returns all keys.
func (m *Map[K, V]) Keys() []K {
	if m == nil || len(m.items) == 0 {
		return nil
	}

	keys := make([]K, 0, len(m.items))
	for key := range m.items {
		keys = append(keys, key)
	}
	return keys
}

// Values returns all values.
func (m *Map[K, V]) Values() []V {
	if m == nil || len(m.items) == 0 {
		return nil
	}

	values := make([]V, 0, len(m.items))
	for _, value := range m.items {
		values = append(values, value)
	}
	return values
}

// All returns a copied built-in map.
func (m *Map[K, V]) All() map[K]V {
	if m == nil || len(m.items) == 0 {
		return map[K]V{}
	}
	out := make(map[K]V, len(m.items))
	maps.Copy(out, m.items)
	return out
}

// Range iterates all entries until fn returns false.
func (m *Map[K, V]) Range(fn func(key K, value V) bool) {
	if m == nil || fn == nil {
		return
	}
	for k, v := range m.items {
		if !fn(k, v) {
			return
		}
	}
}

// Clone returns a shallow copy.
func (m *Map[K, V]) Clone() *Map[K, V] {
	if m == nil || len(m.items) == 0 {
		return NewMap[K, V]()
	}

	out := NewMapWithCapacity[K, V](len(m.items))
	maps.Copy(out.items, m.items)
	return out
}

// WhereEntries returns a new map containing only entries that match predicate.
func (m *Map[K, V]) WhereEntries(predicate func(key K, value V) bool) *Map[K, V] {
	if m == nil || predicate == nil || len(m.items) == 0 {
		return NewMap[K, V]()
	}
	filtered := NewMapWithCapacity[K, V](len(m.items))
	m.Range(func(key K, value V) bool {
		if predicate(key, value) {
			filtered.Set(key, value)
		}
		return true
	})
	return filtered
}

// RejectEntries returns a new map excluding entries that match predicate.
func (m *Map[K, V]) RejectEntries(predicate func(key K, value V) bool) *Map[K, V] {
	if m == nil || predicate == nil || len(m.items) == 0 {
		return NewMap[K, V]()
	}
	rejected := NewMapWithCapacity[K, V](len(m.items))
	m.Range(func(key K, value V) bool {
		if !predicate(key, value) {
			rejected.Set(key, value)
		}
		return true
	})
	return rejected
}

// EachEntry invokes fn for every entry and returns the receiver for chaining.
func (m *Map[K, V]) EachEntry(fn func(key K, value V)) *Map[K, V] {
	if m == nil {
		return NewMap[K, V]()
	}
	if fn == nil {
		return m
	}
	m.Range(func(key K, value V) bool {
		fn(key, value)
		return true
	})
	return m
}

// FirstEntryWhere returns the first entry matching predicate.
func (m *Map[K, V]) FirstEntryWhere(predicate func(key K, value V) bool) (K, V, bool) {
	var zeroK K
	var zeroV V
	if m == nil || predicate == nil || len(m.items) == 0 {
		return zeroK, zeroV, false
	}
	foundK, foundV := zeroK, zeroV
	ok := false
	m.Range(func(key K, value V) bool {
		if !predicate(key, value) {
			return true
		}
		foundK, foundV = key, value
		ok = true
		return false
	})
	return foundK, foundV, ok
}

// AnyEntryMatch reports whether any entry matches predicate.
func (m *Map[K, V]) AnyEntryMatch(predicate func(key K, value V) bool) bool {
	_, _, ok := m.FirstEntryWhere(predicate)
	return ok
}

// AllEntryMatch reports whether all entries match predicate.
func (m *Map[K, V]) AllEntryMatch(predicate func(key K, value V) bool) bool {
	if m == nil || len(m.items) == 0 || predicate == nil {
		return false
	}
	matched := true
	m.Range(func(key K, value V) bool {
		if predicate(key, value) {
			return true
		}
		matched = false
		return false
	})
	return matched
}

func (m *Map[K, V]) ensureInit() {
	if m.items == nil {
		m.items = make(map[K]V)
	}
}

func (m *Map[K, V]) invalidateSerializationCache() {
	if m == nil {
		return
	}
	m.jsonCache = nil
	m.stringCache = ""
	m.jsonDirty = true
}

func (m *Map[K, V]) cacheSerializationData(data []byte) {
	if m == nil {
		return
	}
	m.jsonCache = data
	m.stringCache = string(data)
	m.jsonDirty = false
}
