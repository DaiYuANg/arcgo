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
func (m *RangeMap[T, V]) Put(start, end T, value V) bool {
	if m == nil {
		return false
	}
	input, ok := newRangeEntry(start, end, value)
	if !ok {
		return false
	}
	m.entries = putRangeEntry(m.entries, input)
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
func (m *RangeMap[T, V]) DeleteRange(start, end T) bool {
	if m == nil || len(m.entries) == 0 {
		return false
	}
	input := Range[T]{Start: start, End: end}
	if !input.IsValid() {
		return false
	}
	next, changed := deleteRangeEntries(m.entries, input)
	if changed {
		m.entries = next
	}
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

func newRangeEntry[T cmp.Ordered, V any](start, end T, value V) (RangeEntry[T, V], bool) {
	input := Range[T]{Start: start, End: end}
	if !input.IsValid() {
		return RangeEntry[T, V]{}, false
	}
	return RangeEntry[T, V]{Range: input, Value: value}, true
}

func putRangeEntry[T cmp.Ordered, V any](entries []RangeEntry[T, V], input RangeEntry[T, V]) []RangeEntry[T, V] {
	if len(entries) == 0 {
		return append(entries, input)
	}

	first := findFirstRangeEntryEndingAfter(entries, input.Range.Start)
	if first == len(entries) {
		return append(entries, input)
	}
	if entries[first].Range.Start >= input.Range.End {
		return spliceRangeEntries(entries, first, first, input)
	}

	replacement, end := buildPutRangeEntries(entries, first, input)
	return spliceRangeEntries(entries, first, end, replacement...)
}

func buildPutRangeEntries[T cmp.Ordered, V any](entries []RangeEntry[T, V], first int, input RangeEntry[T, V]) ([]RangeEntry[T, V], int) {
	replacement := make([]RangeEntry[T, V], 0, 3)
	if left, ok := splitLeftRangeEntry(entries[first], input.Range.Start); ok {
		replacement = append(replacement, left)
	}
	replacement = append(replacement, input)

	end, right, ok := splitRightRangeEntry(entries, first, input.Range.End)
	if ok {
		replacement = append(replacement, right)
	}
	return replacement, end
}

func findFirstRangeEntryEndingAfter[T cmp.Ordered, V any](entries []RangeEntry[T, V], point T) int {
	return sort.Search(len(entries), func(i int) bool {
		return entries[i].Range.End > point
	})
}

func splitLeftRangeEntry[T cmp.Ordered, V any](entry RangeEntry[T, V], splitAt T) (RangeEntry[T, V], bool) {
	if entry.Range.Start >= splitAt {
		return RangeEntry[T, V]{}, false
	}
	return RangeEntry[T, V]{
		Range: Range[T]{Start: entry.Range.Start, End: splitAt},
		Value: entry.Value,
	}, true
}

func splitRightRangeEntry[T cmp.Ordered, V any](entries []RangeEntry[T, V], first int, end T) (int, RangeEntry[T, V], bool) {
	for index := first; index < len(entries); index++ {
		entry := entries[index]
		if entry.Range.Start >= end {
			return index, RangeEntry[T, V]{}, false
		}
		if entry.Range.End > end {
			return index + 1, RangeEntry[T, V]{
				Range: Range[T]{Start: end, End: entry.Range.End},
				Value: entry.Value,
			}, true
		}
	}
	return len(entries), RangeEntry[T, V]{}, false
}

func deleteRangeEntries[T cmp.Ordered, V any](entries []RangeEntry[T, V], cut Range[T]) ([]RangeEntry[T, V], bool) {
	first := findFirstRangeEntryEndingAfter(entries, cut.Start)
	if first == len(entries) {
		return entries, false
	}

	next := make([]RangeEntry[T, V], 0, len(entries))
	next = append(next, entries[:first]...)
	changed := false

	for index := first; index < len(entries); index++ {
		entry := entries[index]
		if entry.Range.Start >= cut.End {
			next = append(next, entries[index:]...)
			return next, changed
		}

		changed = true
		fragments, stop := trimRangeEntry(entry, cut)
		next = append(next, fragments...)
		if stop {
			next = append(next, entries[index+1:]...)
			return next, changed
		}
	}

	return next, changed
}

func trimRangeEntry[T cmp.Ordered, V any](entry RangeEntry[T, V], cut Range[T]) ([]RangeEntry[T, V], bool) {
	fragments := make([]RangeEntry[T, V], 0, 2)
	if entry.Range.Start < cut.Start {
		fragments = append(fragments, RangeEntry[T, V]{
			Range: Range[T]{Start: entry.Range.Start, End: cut.Start},
			Value: entry.Value,
		})
	}
	if cut.End < entry.Range.End {
		fragments = append(fragments, RangeEntry[T, V]{
			Range: Range[T]{Start: cut.End, End: entry.Range.End},
			Value: entry.Value,
		})
		return fragments, true
	}
	return fragments, false
}

func spliceRangeEntries[T cmp.Ordered, V any](entries []RangeEntry[T, V], start, end int, replacement ...RangeEntry[T, V]) []RangeEntry[T, V] {
	next := make([]RangeEntry[T, V], 0, len(entries)-(end-start)+len(replacement))
	next = append(next, entries[:start]...)
	next = append(next, replacement...)
	next = append(next, entries[end:]...)
	return next
}
