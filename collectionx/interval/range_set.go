package interval

import (
	"cmp"
	"slices"
	"sort"
)

// RangeSet is a normalized set of half-open ranges [start, end).
// Internal ranges are kept sorted and non-overlapping.
type RangeSet[T cmp.Ordered] struct {
	ranges []Range[T]
}

// NewRangeSet creates an empty range set.
func NewRangeSet[T cmp.Ordered]() *RangeSet[T] {
	return &RangeSet[T]{
		ranges: make([]Range[T], 0),
	}
}

// Add inserts one range and merges overlaps/adjacent ranges.
func (s *RangeSet[T]) Add(start, end T) bool {
	return s.AddRange(Range[T]{Start: start, End: end})
}

// AddRange inserts one range and merges overlaps/adjacent ranges.
func (s *RangeSet[T]) AddRange(r Range[T]) bool {
	if s == nil || !r.IsValid() {
		return false
	}
	s.ranges = addRangeToSet(s.ranges, r)
	return true
}

// Remove removes interval part from the set.
func (s *RangeSet[T]) Remove(start, end T) bool {
	if s == nil || len(s.ranges) == 0 {
		return false
	}
	cut := Range[T]{Start: start, End: end}
	if !cut.IsValid() {
		return false
	}
	next, changed := removeRangeFromSet(s.ranges, cut)
	if changed {
		s.ranges = next
	}
	return changed
}

// Contains reports whether value is in any range.
func (s *RangeSet[T]) Contains(value T) bool {
	if s == nil || len(s.ranges) == 0 {
		return false
	}
	index := sort.Search(len(s.ranges), func(i int) bool {
		return s.ranges[i].End > value
	})
	return index < len(s.ranges) && s.ranges[index].Contains(value)
}

// Overlaps reports whether input range overlaps any stored range.
func (s *RangeSet[T]) Overlaps(start, end T) bool {
	if s == nil || len(s.ranges) == 0 {
		return false
	}
	input := Range[T]{Start: start, End: end}
	if !input.IsValid() {
		return false
	}
	index := sort.Search(len(s.ranges), func(i int) bool {
		return s.ranges[i].End > input.Start
	})
	return index < len(s.ranges) && s.ranges[index].Start < input.End
}

// Ranges returns copied normalized ranges.
func (s *RangeSet[T]) Ranges() []Range[T] {
	if s == nil || len(s.ranges) == 0 {
		return nil
	}
	return slices.Clone(s.ranges)
}

// Len returns number of normalized ranges.
func (s *RangeSet[T]) Len() int {
	if s == nil {
		return 0
	}
	return len(s.ranges)
}

// IsEmpty reports whether set has no ranges.
func (s *RangeSet[T]) IsEmpty() bool {
	return s.Len() == 0
}

// Clear removes all ranges.
func (s *RangeSet[T]) Clear() {
	if s == nil {
		return
	}
	s.ranges = nil
}

// Range iterates normalized ranges until fn returns false.
func (s *RangeSet[T]) Range(fn func(r Range[T]) bool) {
	if s == nil || fn == nil {
		return
	}
	for _, r := range s.ranges {
		if !fn(r) {
			return
		}
	}
}

func addRangeToSet[T cmp.Ordered](ranges []Range[T], input Range[T]) []Range[T] {
	if len(ranges) == 0 {
		return append(ranges, input)
	}

	first := findFirstRangeEndingAtOrAfter(ranges, input.Start)
	if first == len(ranges) {
		return append(ranges, input)
	}
	if ranges[first].Start > input.End {
		return spliceRanges(ranges, first, first, input)
	}

	merged, end := mergeSetRanges(ranges, first, input)
	return spliceRanges(ranges, first, end, merged)
}

func findFirstRangeEndingAtOrAfter[T cmp.Ordered](ranges []Range[T], point T) int {
	return sort.Search(len(ranges), func(i int) bool {
		return ranges[i].End >= point
	})
}

func mergeSetRanges[T cmp.Ordered](ranges []Range[T], first int, merged Range[T]) (Range[T], int) {
	end := first
	for ; end < len(ranges); end++ {
		current := ranges[end]
		if current.Start > merged.End {
			break
		}
		if current.Start < merged.Start {
			merged.Start = current.Start
		}
		if current.End > merged.End {
			merged.End = current.End
		}
	}
	return merged, end
}

func removeRangeFromSet[T cmp.Ordered](ranges []Range[T], cut Range[T]) ([]Range[T], bool) {
	first := findFirstRangeEndingAfter(ranges, cut.Start)
	if first == len(ranges) {
		return ranges, false
	}

	next := make([]Range[T], 0, len(ranges))
	next = append(next, ranges[:first]...)
	changed := false

	for index := first; index < len(ranges); index++ {
		current := ranges[index]
		if current.Start >= cut.End {
			next = append(next, ranges[index:]...)
			return next, changed
		}

		changed = true
		fragments, stop := trimSetRange(current, cut)
		next = append(next, fragments...)
		if stop {
			next = append(next, ranges[index+1:]...)
			return next, changed
		}
	}

	return next, changed
}

func findFirstRangeEndingAfter[T cmp.Ordered](ranges []Range[T], point T) int {
	return sort.Search(len(ranges), func(i int) bool {
		return ranges[i].End > point
	})
}

func trimSetRange[T cmp.Ordered](current, cut Range[T]) ([]Range[T], bool) {
	fragments := make([]Range[T], 0, 2)
	if current.Start < cut.Start {
		fragments = append(fragments, Range[T]{Start: current.Start, End: cut.Start})
	}
	if cut.End < current.End {
		fragments = append(fragments, Range[T]{Start: cut.End, End: current.End})
		return fragments, true
	}
	return fragments, false
}

func spliceRanges[T cmp.Ordered](ranges []Range[T], start, end int, replacement ...Range[T]) []Range[T] {
	next := make([]Range[T], 0, len(ranges)-(end-start)+len(replacement))
	next = append(next, ranges[:start]...)
	next = append(next, replacement...)
	next = append(next, ranges[end:]...)
	return next
}
