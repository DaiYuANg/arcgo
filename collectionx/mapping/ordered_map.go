package mapping

import (
	collectionlist "github.com/DaiYuANg/arcgo/collectionx/list"
	"github.com/samber/lo"
	"github.com/samber/mo"
)

// OrderedMap keeps insertion order of keys.
// Updating existing key does not change its order.
// Zero value is ready to use.
type OrderedMap[K comparable, V any] struct {
	order collectionlist.List[K]
	items Map[K, V]
	index Map[K, int]
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
	return lo.Map(m.order.Values(), func(key K, _ int) V {
		value, _ := m.items.Get(key)
		return value
	})
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

func (m *OrderedMap[K, V]) ensureInit() {
	m.items.ensureInit()
	m.index.ensureInit()
}
