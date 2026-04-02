package mapping

import (
	"slices"

	collectionlist "github.com/DaiYuANg/arcgo/collectionx/list"
	"github.com/samber/lo"
)

// WhereKeys returns a new multimap containing only keys that match predicate.
func (m *MultiMap[K, V]) WhereKeys(predicate func(key K, values []V) bool) *MultiMap[K, V] {
	if m == nil || predicate == nil || m.items.Len() == 0 {
		return NewMultiMap[K, V]()
	}
	filtered := NewMultiMapWithCapacity[K, V](m.items.Len())
	m.items.Range(func(key K, values []V) bool {
		if predicate(key, values) {
			filtered.items.Set(key, slices.Clone(values))
			filtered.valueCount += len(values)
		}
		return true
	})
	return filtered
}

// RejectKeys returns a new multimap excluding keys that match predicate.
func (m *MultiMap[K, V]) RejectKeys(predicate func(key K, values []V) bool) *MultiMap[K, V] {
	if m == nil || predicate == nil || m.items.Len() == 0 {
		return NewMultiMap[K, V]()
	}
	rejected := NewMultiMapWithCapacity[K, V](m.items.Len())
	m.items.Range(func(key K, values []V) bool {
		if !predicate(key, values) {
			rejected.items.Set(key, slices.Clone(values))
			rejected.valueCount += len(values)
		}
		return true
	})
	return rejected
}

// WhereValues returns a new multimap containing only values that match predicate.
func (m *MultiMap[K, V]) WhereValues(predicate func(key K, value V) bool) *MultiMap[K, V] {
	if m == nil || predicate == nil || m.items.Len() == 0 {
		return NewMultiMap[K, V]()
	}
	filtered := NewMultiMapWithCapacity[K, V](m.items.Len())
	m.items.Range(func(key K, values []V) bool {
		matching := lo.Filter(values, func(value V, _ int) bool { return predicate(key, value) })
		if len(matching) > 0 {
			filtered.items.Set(key, matching)
			filtered.valueCount += len(matching)
		}
		return true
	})
	return filtered
}

// RejectValues returns a new multimap excluding values that match predicate.
func (m *MultiMap[K, V]) RejectValues(predicate func(key K, value V) bool) *MultiMap[K, V] {
	if m == nil || predicate == nil || m.items.Len() == 0 {
		return NewMultiMap[K, V]()
	}
	rejected := NewMultiMapWithCapacity[K, V](m.items.Len())
	m.items.Range(func(key K, values []V) bool {
		remaining := lo.Reject(values, func(value V, _ int) bool { return predicate(key, value) })
		if len(remaining) > 0 {
			rejected.items.Set(key, remaining)
			rejected.valueCount += len(remaining)
		}
		return true
	})
	return rejected
}

// EachKey invokes fn for each key-value list and returns the receiver for chaining.
func (m *MultiMap[K, V]) EachKey(fn func(key K, values []V)) *MultiMap[K, V] {
	if m == nil {
		return NewMultiMap[K, V]()
	}
	if fn == nil {
		return m
	}
	m.items.Range(func(key K, values []V) bool {
		fn(key, values)
		return true
	})
	return m
}

// EachValue invokes fn for each value and returns the receiver for chaining.
func (m *MultiMap[K, V]) EachValue(fn func(key K, value V)) *MultiMap[K, V] {
	if m == nil {
		return NewMultiMap[K, V]()
	}
	if fn == nil {
		return m
	}
	m.items.Range(func(key K, values []V) bool {
		lo.ForEach(values, func(value V, _ int) { fn(key, value) })
		return true
	})
	return m
}

// FirstValueWhere returns the first value matching predicate.
func (m *MultiMap[K, V]) FirstValueWhere(predicate func(key K, value V) bool) (K, V, bool) {
	var zeroK K
	var zeroV V
	if m == nil || predicate == nil || m.items.Len() == 0 {
		return zeroK, zeroV, false
	}
	foundK, foundV := zeroK, zeroV
	ok := false
	m.items.Range(func(key K, values []V) bool {
		value, matched := lo.Find(values, func(value V) bool { return predicate(key, value) })
		if matched {
			foundK, foundV = key, value
			ok = true
			return false
		}
		return true
	})
	return foundK, foundV, ok
}

// AnyValueMatch reports whether any stored value matches predicate.
func (m *MultiMap[K, V]) AnyValueMatch(predicate func(key K, value V) bool) bool {
	_, _, ok := m.FirstValueWhere(predicate)
	return ok
}

// AllValuesMatch reports whether all stored values match predicate.
func (m *MultiMap[K, V]) AllValuesMatch(predicate func(key K, value V) bool) bool {
	if m == nil || m.valueCount == 0 || predicate == nil {
		return false
	}
	matched := true
	m.items.Range(func(key K, values []V) bool {
		if lo.EveryBy(values, func(value V) bool { return predicate(key, value) }) {
			return true
		}
		matched = false
		return false
	})
	return matched
}

// FlattenValues returns all values in a new list while preserving per-key value order.
func (m *MultiMap[K, V]) FlattenValues() *collectionlist.List[V] {
	if m == nil || m.valueCount == 0 {
		return collectionlist.NewList[V]()
	}
	flattened := collectionlist.NewListWithCapacity[V](m.valueCount)
	m.items.Range(func(_ K, values []V) bool {
		flattened.MergeSlice(values)
		return true
	})
	return flattened
}
