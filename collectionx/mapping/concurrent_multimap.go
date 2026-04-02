package mapping

import (
	"sync"

	collectionlist "github.com/DaiYuANg/arcgo/collectionx/list"
	"github.com/samber/mo"
)

// ConcurrentMultiMap is a goroutine-safe multimap.
// Zero value is ready to use.
type ConcurrentMultiMap[K comparable, V any] struct {
	mu   sync.RWMutex
	core *MultiMap[K, V]
}

// NewConcurrentMultiMap creates an empty concurrent multimap.
func NewConcurrentMultiMap[K comparable, V any]() *ConcurrentMultiMap[K, V] {
	return NewConcurrentMultiMapWithCapacity[K, V](0)
}

// NewConcurrentMultiMapWithCapacity creates an empty concurrent multimap with preallocated key capacity.
func NewConcurrentMultiMapWithCapacity[K comparable, V any](capacity int) *ConcurrentMultiMap[K, V] {
	return &ConcurrentMultiMap[K, V]{
		core: NewMultiMapWithCapacity[K, V](capacity),
	}
}

// Put appends one value for key.
func (m *ConcurrentMultiMap[K, V]) Put(key K, value V) {
	m.PutAll(key, value)
}

// PutAll appends values for key.
func (m *ConcurrentMultiMap[K, V]) PutAll(key K, values ...V) {
	if m == nil || len(values) == 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureInitLocked()
	m.core.PutAll(key, values...)
}

// Set replaces all values for key.
// Passing no values removes the key.
func (m *ConcurrentMultiMap[K, V]) Set(key K, values ...V) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureInitLocked()
	m.core.Set(key, values...)
}

// Get returns a read-only slice view for key.
// Callers must not modify the returned slice.
func (m *ConcurrentMultiMap[K, V]) Get(key K) []V {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.core == nil {
		return nil
	}
	return m.core.Get(key)
}

// GetCopy returns an owned copy of values for key.
func (m *ConcurrentMultiMap[K, V]) GetCopy(key K) []V {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.core == nil {
		return nil
	}
	return m.core.GetCopy(key)
}

// GetOption returns values for key as mo.Option.
func (m *ConcurrentMultiMap[K, V]) GetOption(key K) mo.Option[[]V] {
	values := m.Get(key)
	if len(values) == 0 {
		return mo.None[[]V]()
	}
	return mo.Some(values)
}

// Delete removes all values for key.
func (m *ConcurrentMultiMap[K, V]) Delete(key K) bool {
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

// DeleteValueIf removes values matching predicate under key and returns removed count.
func (m *ConcurrentMultiMap[K, V]) DeleteValueIf(key K, predicate func(value V) bool) int {
	if m == nil || predicate == nil {
		return 0
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.core == nil {
		return 0
	}
	return m.core.DeleteValueIf(key, predicate)
}

// ContainsKey reports whether key exists.
func (m *ConcurrentMultiMap[K, V]) ContainsKey(key K) bool {
	if m == nil {
		return false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.core == nil {
		return false
	}
	return m.core.ContainsKey(key)
}

// Len returns key count.
func (m *ConcurrentMultiMap[K, V]) Len() int {
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

// ValueCount returns total stored value count.
func (m *ConcurrentMultiMap[K, V]) ValueCount() int {
	if m == nil {
		return 0
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.core == nil {
		return 0
	}
	return m.core.ValueCount()
}

// IsEmpty reports whether map has no keys.
func (m *ConcurrentMultiMap[K, V]) IsEmpty() bool {
	return m.Len() == 0
}

// Clear removes all entries.
func (m *ConcurrentMultiMap[K, V]) Clear() {
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

// Keys returns all keys.
func (m *ConcurrentMultiMap[K, V]) Keys() []K {
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

// All returns a deep-copied built-in map.
func (m *ConcurrentMultiMap[K, V]) All() map[K][]V {
	if m == nil {
		return map[K][]V{}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.core == nil {
		return map[K][]V{}
	}
	return m.core.All()
}

// Snapshot returns an immutable-style copy in a normal MultiMap.
func (m *ConcurrentMultiMap[K, V]) Snapshot() *MultiMap[K, V] {
	if m == nil {
		return NewMultiMap[K, V]()
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.core == nil {
		return NewMultiMap[K, V]()
	}
	return m.core.Clone()
}

// WhereKeys returns a filtered multimap snapshot.
func (m *ConcurrentMultiMap[K, V]) WhereKeys(predicate func(key K, values []V) bool) *MultiMap[K, V] {
	return m.Snapshot().WhereKeys(predicate)
}

// RejectKeys returns a filtered multimap snapshot that excludes matching keys.
func (m *ConcurrentMultiMap[K, V]) RejectKeys(predicate func(key K, values []V) bool) *MultiMap[K, V] {
	return m.Snapshot().RejectKeys(predicate)
}

// WhereValues returns a filtered multimap snapshot containing only matching values.
func (m *ConcurrentMultiMap[K, V]) WhereValues(predicate func(key K, value V) bool) *MultiMap[K, V] {
	return m.Snapshot().WhereValues(predicate)
}

// RejectValues returns a filtered multimap snapshot excluding matching values.
func (m *ConcurrentMultiMap[K, V]) RejectValues(predicate func(key K, value V) bool) *MultiMap[K, V] {
	return m.Snapshot().RejectValues(predicate)
}

// EachKey iterates a stable snapshot and returns it for chaining.
func (m *ConcurrentMultiMap[K, V]) EachKey(fn func(key K, values []V)) *MultiMap[K, V] {
	return m.Snapshot().EachKey(fn)
}

// EachValue iterates a stable snapshot and returns it for chaining.
func (m *ConcurrentMultiMap[K, V]) EachValue(fn func(key K, value V)) *MultiMap[K, V] {
	return m.Snapshot().EachValue(fn)
}

// FirstValueWhere returns the first value matching predicate from a stable snapshot.
func (m *ConcurrentMultiMap[K, V]) FirstValueWhere(predicate func(key K, value V) bool) (K, V, bool) {
	return m.Snapshot().FirstValueWhere(predicate)
}

// AnyValueMatch reports whether any value in a stable snapshot matches predicate.
func (m *ConcurrentMultiMap[K, V]) AnyValueMatch(predicate func(key K, value V) bool) bool {
	return m.Snapshot().AnyValueMatch(predicate)
}

// AllValuesMatch reports whether all values in a stable snapshot match predicate.
func (m *ConcurrentMultiMap[K, V]) AllValuesMatch(predicate func(key K, value V) bool) bool {
	return m.Snapshot().AllValuesMatch(predicate)
}

// FlattenValues returns all values from a stable snapshot.
func (m *ConcurrentMultiMap[K, V]) FlattenValues() *collectionlist.List[V] {
	return m.Snapshot().FlattenValues()
}

// Range iterates key-values snapshots until fn returns false.
func (m *ConcurrentMultiMap[K, V]) Range(fn func(key K, values []V) bool) {
	if m == nil || fn == nil {
		return
	}
	for key, values := range m.All() {
		if !fn(key, values) {
			return
		}
	}
}

func (m *ConcurrentMultiMap[K, V]) ensureInitLocked() {
	if m.core == nil {
		m.core = NewMultiMap[K, V]()
	}
}
