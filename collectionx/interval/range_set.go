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
func (s *RangeSet[T]) Add(start T, end T) bool {
	return s.AddRange(Range[T]{Start: start, End: end})
}

// AddRange inserts one range and merges overlaps/adjacent ranges.
func (s *RangeSet[T]) AddRange(r Range[T]) bool {
	if s == nil || !r.IsValid() {
		return false
	}

	if len(s.ranges) == 0 {
		s.ranges = append(s.ranges, r)
		return true
	}

	first := sort.Search(len(s.ranges), func(i int) bool {
		return s.ranges[i].End >= r.Start
	})
	if first == len(s.ranges) {
		s.ranges = append(s.ranges, r)
		return true
	}

	next := make([]Range[T], 0, len(s.ranges)+1)
	next = append(next, s.ranges[:first]...)

	merged := r
	index := first
	for ; index < len(s.ranges); index++ {
		current := s.ranges[index]
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

	next = append(next, merged)
	next = append(next, s.ranges[index:]...)
	s.ranges = next
	return true
}

// Remove removes interval part from the set.
func (s *RangeSet[T]) Remove(start T, end T) bool {
	if s == nil || len(s.ranges) == 0 {
		return false
	}
	cut := Range[T]{Start: start, End: end}
	if !cut.IsValid() {
		return false
	}

	first := sort.Search(len(s.ranges), func(i int) bool {
		return s.ranges[i].End > cut.Start
	})
	if first == len(s.ranges) {
		return false
	}

	changed := false
	next := make([]Range[T], 0, len(s.ranges))
	next = append(next, s.ranges[:first]...)
	for i := first; i < len(s.ranges); i++ {
		current := s.ranges[i]
		if current.Start >= cut.End {
			next = append(next, s.ranges[i:]...)
			s.ranges = next
			return changed
		}
		if current.End <= cut.Start {
			next = append(next, current)
			continue
		}

		changed = true
		if current.Start < cut.Start {
			next = append(next, Range[T]{Start: current.Start, End: cut.Start})
		}
		if cut.End < current.End {
			next = append(next, Range[T]{Start: cut.End, End: current.End})
			next = append(next, s.ranges[i+1:]...)
			s.ranges = next
			return true
		}
	}
	s.ranges = next
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
func (s *RangeSet[T]) Overlaps(start T, end T) bool {
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
