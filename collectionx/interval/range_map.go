package interval

import (
	"cmp"
	"slices"
	"sort"

	"github.com/samber/mo"
)

// RangeEntry is one range-value pair used by RangeMap.
type RangeEntry[T cmp.Ordered, V any] struct {
	Range Range[T]
	Value V
}

// RangeMap maps half-open ranges [start, end) to values.
// Overlapping Put overrides existing values in the input interval.
// Internal entries are kept sorted and non-overlapping.
type RangeMap[T cmp.Ordered, V any] struct {
	entries []RangeEntry[T, V]
}

// NewRangeMap creates an empty range map.
func NewRangeMap[T cmp.Ordered, V any]() *RangeMap[T, V] {
	return &RangeMap[T, V]{
		entries: make([]RangeEntry[T, V], 0),
	}
}

// Put assigns value to [start, end), overriding any overlaps.
func (m *RangeMap[T, V]) Put(start T, end T, value V) bool {
	if m == nil {
		return false
	}
	input := Range[T]{Start: start, End: end}
	if !input.IsValid() {
		return false
	}
	inputEntry := RangeEntry[T, V]{Range: input, Value: value}

	if len(m.entries) == 0 {
		m.entries = append(m.entries, inputEntry)
		return true
	}

	first := sort.Search(len(m.entries), func(i int) bool {
		return m.entries[i].Range.End > input.Start
	})
	if first == len(m.entries) {
		m.entries = append(m.entries, inputEntry)
		return true
	}

	next := make([]RangeEntry[T, V], 0, len(m.entries)+1)
	next = append(next, m.entries[:first]...)

	for i := first; i < len(m.entries); i++ {
		entry := m.entries[i]
		if entry.Range.Start >= input.End {
			next = append(next, inputEntry)
			next = append(next, m.entries[i:]...)
			m.entries = next
			return true
		}

		if entry.Range.End <= input.Start {
			next = append(next, entry)
			continue
		}

		if entry.Range.Start < input.Start {
			next = append(next, RangeEntry[T, V]{
				Range: Range[T]{Start: entry.Range.Start, End: input.Start},
				Value: entry.Value,
			})
		}

		if input.End < entry.Range.End {
			next = append(next, inputEntry)
			next = append(next, RangeEntry[T, V]{
				Range: Range[T]{Start: input.End, End: entry.Range.End},
				Value: entry.Value,
			})
			next = append(next, m.entries[i+1:]...)
			m.entries = next
			return true
		}
	}

	next = append(next, inputEntry)
	m.entries = next
	return true
}

// Get returns value for point query.
func (m *RangeMap[T, V]) Get(point T) (V, bool) {
	var zero V
	if m == nil || len(m.entries) == 0 {
		return zero, false
	}
	index := sort.Search(len(m.entries), func(i int) bool {
		return m.entries[i].Range.End > point
	})
	if index < len(m.entries) && m.entries[index].Range.Contains(point) {
		return m.entries[index].Value, true
	}
	return zero, false
}

// GetOption returns value for point query as mo.Option.
func (m *RangeMap[T, V]) GetOption(point T) mo.Option[V] {
	value, ok := m.Get(point)
	if !ok {
		return mo.None[V]()
	}
	return mo.Some(value)
}

// DeleteRange removes mappings in [start, end).
func (m *RangeMap[T, V]) DeleteRange(start T, end T) bool {
	if m == nil || len(m.entries) == 0 {
		return false
	}
	input := Range[T]{Start: start, End: end}
	if !input.IsValid() {
		return false
	}

	first := sort.Search(len(m.entries), func(i int) bool {
		return m.entries[i].Range.End > input.Start
	})
	if first == len(m.entries) {
		return false
	}

	changed := false
	next := make([]RangeEntry[T, V], 0, len(m.entries))
	next = append(next, m.entries[:first]...)

	for i := first; i < len(m.entries); i++ {
		entry := m.entries[i]
		if entry.Range.Start >= input.End {
			next = append(next, m.entries[i:]...)
			m.entries = next
			return changed
		}
		if entry.Range.End <= input.Start {
			next = append(next, entry)
			continue
		}

		changed = true
		if entry.Range.Start < input.Start {
			next = append(next, RangeEntry[T, V]{
				Range: Range[T]{Start: entry.Range.Start, End: input.Start},
				Value: entry.Value,
			})
		}
		if input.End < entry.Range.End {
			next = append(next, RangeEntry[T, V]{
				Range: Range[T]{Start: input.End, End: entry.Range.End},
				Value: entry.Value,
			})
			next = append(next, m.entries[i+1:]...)
			m.entries = next
			return true
		}
	}

	m.entries = next
	return changed
}

// Entries returns copied entries sorted by range start.
func (m *RangeMap[T, V]) Entries() []RangeEntry[T, V] {
	if m == nil || len(m.entries) == 0 {
		return nil
	}
	return slices.Clone(m.entries)
}

// Len returns non-overlapping entry count.
func (m *RangeMap[T, V]) Len() int {
	if m == nil {
		return 0
	}
	return len(m.entries)
}

// IsEmpty reports whether map has no entries.
func (m *RangeMap[T, V]) IsEmpty() bool {
	return m.Len() == 0
}

// Clear removes all entries.
func (m *RangeMap[T, V]) Clear() {
	if m == nil {
		return
	}
	m.entries = nil
}

// Range iterates entries in start order until fn returns false.
func (m *RangeMap[T, V]) Range(fn func(entry RangeEntry[T, V]) bool) {
	if m == nil || fn == nil {
		return
	}
	for _, entry := range m.entries {
		if !fn(entry) {
			return
		}
	}
}
