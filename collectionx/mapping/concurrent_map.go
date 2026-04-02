package mapping

import (
	"sync"

	"github.com/samber/mo"
)

// ConcurrentMap is a goroutine-safe strongly-typed map.
// Zero value is ready to use.
type ConcurrentMap[K comparable, V any] struct {
	mu   sync.RWMutex
	core *Map[K, V]
}

// NewConcurrentMap creates an empty concurrent map.
func NewConcurrentMap[K comparable, V any]() *ConcurrentMap[K, V] {
	return NewConcurrentMapWithCapacity[K, V](0)
}

// NewConcurrentMapWithCapacity creates an empty concurrent map with preallocated capacity.
func NewConcurrentMapWithCapacity[K comparable, V any](capacity int) *ConcurrentMap[K, V] {
	return &ConcurrentMap[K, V]{
		core: NewMapWithCapacity[K, V](capacity),
	}
}

// Set puts a key-value pair.
func (m *ConcurrentMap[K, V]) Set(key K, value V) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureInitLocked()
	m.core.Set(key, value)
}

// SetAll copies all entries from source.
func (m *ConcurrentMap[K, V]) SetAll(source map[K]V) {
	if m == nil || len(source) == 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureInitLocked()
	m.core.SetAll(source)
}

// Get returns the value for key.
func (m *ConcurrentMap[K, V]) Get(key K) (V, bool) {
	var zero V
	if m == nil {
		return zero, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.core == nil {
		return zero, false
	}
	return m.core.Get(key)
}

// GetOption returns value for key as mo.Option.
func (m *ConcurrentMap[K, V]) GetOption(key K) mo.Option[V] {
	value, ok := m.Get(key)
	if !ok {
		return mo.None[V]()
	}
	return mo.Some(value)
}

// GetOrDefault returns value for key or fallback when key does not exist.
func (m *ConcurrentMap[K, V]) GetOrDefault(key K, fallback V) V {
	value, ok := m.Get(key)
	if !ok {
		return fallback
	}
	return value
}

// GetOrStore returns existing value if key exists; otherwise stores and returns value.
// loaded is true when key already exists.
func (m *ConcurrentMap[K, V]) GetOrStore(key K, value V) (actual V, loaded bool) {
	if m == nil {
		return value, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureInitLocked()

	if existing, ok := m.core.Get(key); ok {
		return existing, true
	}
	m.core.Set(key, value)
	return value, false
}

// Delete removes key and reports whether it existed.
func (m *ConcurrentMap[K, V]) Delete(key K) bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.core == nil {
		return false
	}
	return m.core.Delete(key)
}

// LoadAndDelete removes key and returns previous value.
func (m *ConcurrentMap[K, V]) LoadAndDelete(key K) (V, bool) {
	var zero V
	if m == nil {
		return zero, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.core == nil {
		return zero, false
	}
	value, ok := m.core.Get(key)
	if !ok {
		return zero, false
	}
	_ = m.core.Delete(key)
	return value, true
}

// LoadAndDeleteOption removes key and returns previous value as mo.Option.
func (m *ConcurrentMap[K, V]) LoadAndDeleteOption(key K) mo.Option[V] {
	value, ok := m.LoadAndDelete(key)
	if !ok {
		return mo.None[V]()
	}
	return mo.Some(value)
}

// Len returns total entry count.
func (m *ConcurrentMap[K, V]) Len() int {
	if m == nil {
		return 0
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.core == nil {
		return 0
	}
	return m.core.Len()
}

// IsEmpty reports whether map has no entries.
func (m *ConcurrentMap[K, V]) IsEmpty() bool {
	return m.Len() == 0
}

// Clear removes all entries.
func (m *ConcurrentMap[K, V]) Clear() {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.core == nil {
		return
	}
	m.core.Clear()
}

// Keys returns a snapshot of keys.
func (m *ConcurrentMap[K, V]) Keys() []K {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.core == nil {
		return nil
	}
	return m.core.Keys()
}

// Values returns a snapshot of values.
func (m *ConcurrentMap[K, V]) Values() []V {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.core == nil {
		return nil
	}
	return m.core.Values()
}

// All returns a copied built-in map.
func (m *ConcurrentMap[K, V]) All() map[K]V {
	if m == nil {
		return map[K]V{}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.core == nil {
		return map[K]V{}
	}
	return m.core.All()
}

// Range iterates a stable snapshot until fn returns false.
func (m *ConcurrentMap[K, V]) Range(fn func(key K, value V) bool) {
	if m == nil || fn == nil {
		return
	}
	for key, value := range m.All() {
		if !fn(key, value) {
			return
		}
	}
}

// WhereEntries returns a filtered snapshot map.
func (m *ConcurrentMap[K, V]) WhereEntries(predicate func(key K, value V) bool) *Map[K, V] {
	return NewMapFrom(m.All()).WhereEntries(predicate)
}

// RejectEntries returns a filtered snapshot map that excludes matching entries.
func (m *ConcurrentMap[K, V]) RejectEntries(predicate func(key K, value V) bool) *Map[K, V] {
	return NewMapFrom(m.All()).RejectEntries(predicate)
}

// EachEntry iterates a stable snapshot and returns it for chaining.
func (m *ConcurrentMap[K, V]) EachEntry(fn func(key K, value V)) *Map[K, V] {
	return NewMapFrom(m.All()).EachEntry(fn)
}

// FirstEntryWhere returns the first entry matching predicate from a stable snapshot.
func (m *ConcurrentMap[K, V]) FirstEntryWhere(predicate func(key K, value V) bool) (K, V, bool) {
	return NewMapFrom(m.All()).FirstEntryWhere(predicate)
}

// AnyEntryMatch reports whether any entry in a stable snapshot matches predicate.
func (m *ConcurrentMap[K, V]) AnyEntryMatch(predicate func(key K, value V) bool) bool {
	return NewMapFrom(m.All()).AnyEntryMatch(predicate)
}

// AllEntryMatch reports whether all entries in a stable snapshot match predicate.
func (m *ConcurrentMap[K, V]) AllEntryMatch(predicate func(key K, value V) bool) bool {
	return NewMapFrom(m.All()).AllEntryMatch(predicate)
}

func (m *ConcurrentMap[K, V]) ensureInitLocked() {
	if m.core == nil {
		m.core = NewMap[K, V]()
	}
}
