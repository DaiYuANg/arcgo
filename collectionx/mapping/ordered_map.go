//revive:disable:file-length-limit Ordered map methods are kept together to preserve the collection API surface.

package mapping

import (
	collectionlist "github.com/DaiYuANg/arcgo/collectionx/list"
	"github.com/samber/mo"
	"slices"
)

// OrderedMap keeps insertion order of keys. Zero value is ready to use.
type OrderedMap[K comparable, V any] struct {
	order collectionlist.List[K]
	items Map[K, V]
	index Map[K, int]

	valuesCache []V
	valuesDirty bool
	jsonCache   []byte
	stringCache string
	jsonDirty   bool
}

// NewOrderedMap creates an empty ordered map.
func NewOrderedMap[K comparable, V any]() *OrderedMap[K, V] {
	return NewOrderedMapWithCapacity[K, V](0)
}

// NewOrderedMapWithCapacity creates an empty ordered map with preallocated capacity.
func NewOrderedMapWithCapacity[K comparable, V any](capacity int) *OrderedMap[K, V] {
	if capacity <= 0 {
		return &OrderedMap[K, V]{}
	}
	return &OrderedMap[K, V]{
		order: *collectionlist.NewListWithCapacity[K](capacity),
		items: *NewMapWithCapacity[K, V](capacity),
		index: *NewMapWithCapacity[K, int](capacity),
	}
}

// Set inserts or updates key-value pair.
func (m *OrderedMap[K, V]) Set(key K, value V) {
	if m == nil {
		return
	}
	m.ensureInit()

	if _, exists := m.items.Get(key); !exists {
		m.order.Add(key)
		m.index.Set(key, m.order.Len()-1)
	}
	m.items.Set(key, value)
	m.invalidateValuesCache()
	m.invalidateSerializationCache()
}

// Get returns value by key.
func (m *OrderedMap[K, V]) Get(key K) (V, bool) {
	var zero V
	if m == nil {
		return zero, false
	}
	value, ok := m.items.Get(key)
	return value, ok
}

// GetOption returns value by key as mo.Option.
func (m *OrderedMap[K, V]) GetOption(key K) mo.Option[V] {
	value, ok := m.Get(key)
	if !ok {
		return mo.None[V]()
	}
	return mo.Some(value)
}

// At returns key-value pair at insertion index.
func (m *OrderedMap[K, V]) At(pos int) (K, V, bool) {
	var zeroK K
	var zeroV V
	if m == nil {
		return zeroK, zeroV, false
	}
	key, ok := m.order.Get(pos)
	if !ok {
		return zeroK, zeroV, false
	}
	value, _ := m.items.Get(key)
	return key, value, true
}

// Delete removes key.
func (m *OrderedMap[K, V]) Delete(key K) bool {
	if m == nil {
		return false
	}
	pos, ok := m.index.Get(key)
	if !ok {
		return false
	}

	m.items.Delete(key)
	m.index.Delete(key)

	_, _ = m.order.RemoveAt(pos)
	for i := pos; i < m.order.Len(); i++ {
		nextKey, _ := m.order.Get(i)
		m.index.Set(nextKey, i)
	}
	m.invalidateValuesCache()
	m.invalidateSerializationCache()
	return true
}

// Len returns pair count.
func (m *OrderedMap[K, V]) Len() int {
	if m == nil {
		return 0
	}
	return m.order.Len()
}

// IsEmpty reports whether map is empty.
func (m *OrderedMap[K, V]) IsEmpty() bool {
	return m.Len() == 0
}

// Clear removes all pairs.
func (m *OrderedMap[K, V]) Clear() {
	if m == nil {
		return
	}
	m.order.Clear()
	m.items.Clear()
	m.index.Clear()
	m.valuesCache = nil
	m.valuesDirty = false
	m.jsonCache = nil
	m.stringCache = ""
	m.jsonDirty = false
}

// Keys returns keys in insertion order.
func (m *OrderedMap[K, V]) Keys() []K {
	if m == nil {
		return nil
	}
	keys := m.order.Values()
	if len(keys) == 0 {
		return nil
	}
	return keys
}

// Values returns values in key insertion order.
func (m *OrderedMap[K, V]) Values() []V {
	if m == nil || m.order.Len() == 0 {
		return nil
	}
	if !m.valuesDirty && len(m.valuesCache) > 0 {
		return slices.Clone(m.valuesCache)
	}

	values := make([]V, 0, m.order.Len())
	m.order.Range(func(_ int, key K) bool {
		value, _ := m.items.Get(key)
		values = append(values, value)
		return true
	})
	m.valuesCache = values
	m.valuesDirty = false
	return slices.Clone(values)
}

// All returns copied unordered built-in map.
func (m *OrderedMap[K, V]) All() map[K]V {
	if m == nil || m.items.Len() == 0 {
		return map[K]V{}
	}
	return m.items.All()
}

// Range iterates in insertion order until fn returns false.
func (m *OrderedMap[K, V]) Range(fn func(key K, value V) bool) {
	if m == nil || fn == nil {
		return
	}
	m.order.Range(func(_ int, key K) bool {
		value, _ := m.items.Get(key)
		return fn(key, value)
	})
}

// Clone returns a shallow copy.
func (m *OrderedMap[K, V]) Clone() *OrderedMap[K, V] {
	if m == nil {
		return NewOrderedMap[K, V]()
	}
	out := NewOrderedMapWithCapacity[K, V](m.order.Len())
	out.order.MergeSlice(m.order.Values())
	out.items.SetAll(m.items.All())
	out.index.SetAll(m.index.All())
	return out
}

func (m *OrderedMap[K, V]) invalidateValuesCache() {
	if m == nil {
		return
	}
	m.valuesCache = nil
	m.valuesDirty = true
}

func (m *OrderedMap[K, V]) invalidateSerializationCache() {
	if m == nil {
		return
	}
	m.jsonCache = nil
	m.stringCache = ""
	m.jsonDirty = true
}

func (m *OrderedMap[K, V]) cacheSerializationData(data []byte) {
	if m == nil {
		return
	}
	m.jsonCache = data
	m.stringCache = string(data)
	m.jsonDirty = false
}

// WhereEntries returns a new ordered map containing only entries that match predicate.
func (m *OrderedMap[K, V]) WhereEntries(predicate func(key K, value V) bool) *OrderedMap[K, V] {
	if m == nil || predicate == nil || m.order.Len() == 0 {
		return NewOrderedMap[K, V]()
	}
	filtered := NewOrderedMapWithCapacity[K, V](m.order.Len())
	m.Range(func(key K, value V) bool {
		if predicate(key, value) {
			filtered.Set(key, value)
		}
		return true
	})
	return filtered
}

// RejectEntries returns a new ordered map excluding entries that match predicate.
func (m *OrderedMap[K, V]) RejectEntries(predicate func(key K, value V) bool) *OrderedMap[K, V] {
	if m == nil || predicate == nil || m.order.Len() == 0 {
		return NewOrderedMap[K, V]()
	}
	rejected := NewOrderedMapWithCapacity[K, V](m.order.Len())
	m.Range(func(key K, value V) bool {
		if !predicate(key, value) {
			rejected.Set(key, value)
		}
		return true
	})
	return rejected
}

// Take returns the first n entries as a new ordered map.
func (m *OrderedMap[K, V]) Take(n int) *OrderedMap[K, V] {
	if m == nil || n <= 0 || m.order.Len() == 0 {
		return NewOrderedMap[K, V]()
	}
	if n >= m.order.Len() {
		return m.Clone()
	}
	out := NewOrderedMapWithCapacity[K, V](n)
	for i := range n {
		key, value, _ := m.At(i)
		out.Set(key, value)
	}
	return out
}

// Drop returns a new ordered map without the first n entries.
func (m *OrderedMap[K, V]) Drop(n int) *OrderedMap[K, V] {
	if m == nil || m.order.Len() == 0 {
		return NewOrderedMap[K, V]()
	}
	if n <= 0 {
		return m.Clone()
	}
	if n >= m.order.Len() {
		return NewOrderedMap[K, V]()
	}
	out := NewOrderedMapWithCapacity[K, V](m.order.Len() - n)
	for i := n; i < m.order.Len(); i++ {
		key, value, _ := m.At(i)
		out.Set(key, value)
	}
	return out
}

// EachEntry invokes fn for every entry and returns the receiver for chaining.
func (m *OrderedMap[K, V]) EachEntry(fn func(key K, value V)) *OrderedMap[K, V] {
	if m == nil {
		return NewOrderedMap[K, V]()
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
func (m *OrderedMap[K, V]) FirstEntryWhere(predicate func(key K, value V) bool) (K, V, bool) {
	var zeroK K
	var zeroV V
	if m == nil || predicate == nil || m.order.Len() == 0 {
		return zeroK, zeroV, false
	}
	for i := range m.order.Len() {
		key, value, _ := m.At(i)
		if predicate(key, value) {
			return key, value, true
		}
	}
	return zeroK, zeroV, false
}

// AnyEntryMatch reports whether any entry matches predicate.
func (m *OrderedMap[K, V]) AnyEntryMatch(predicate func(key K, value V) bool) bool {
	_, _, ok := m.FirstEntryWhere(predicate)
	return ok
}

// AllEntryMatch reports whether all entries match predicate.
func (m *OrderedMap[K, V]) AllEntryMatch(predicate func(key K, value V) bool) bool {
	if m == nil || m.order.Len() == 0 || predicate == nil {
		return false
	}
	for i := range m.order.Len() {
		key, value, _ := m.At(i)
		if !predicate(key, value) {
			return false
		}
	}
	return true
}

func (m *OrderedMap[K, V]) ensureInit() {
	m.items.ensureInit()
	m.index.ensureInit()
}
