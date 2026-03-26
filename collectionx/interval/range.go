package interval

import "cmp"

// Range represents half-open interval [Start, End).
// Valid range requires Start < End.
type Range[T cmp.Ordered] struct {
	Start T
	End   T
}

// NewRange creates a valid half-open range.
func NewRange[T cmp.Ordered](start, end T) (Range[T], bool) {
	r := Range[T]{Start: start, End: end}
	return r, r.IsValid()
}

// IsValid reports whether range is valid.
func (r Range[T]) IsValid() bool {
	return r.Start < r.End
}

// Contains reports whether value is inside [Start, End).
func (r Range[T]) Contains(value T) bool {
	return r.Start <= value && value < r.End
}

// Overlaps reports whether two ranges overlap.
func (r Range[T]) Overlaps(other Range[T]) bool {
	return r.Start < other.End && other.Start < r.End
}

// IsAdjacent reports whether two ranges touch each other.
func (r Range[T]) IsAdjacent(other Range[T]) bool {
	return r.End == other.Start || other.End == r.Start
}

// Merge merges overlapping or adjacent ranges and reports success.
func (r Range[T]) Merge(other Range[T]) (Range[T], bool) {
	if !r.IsValid() || !other.IsValid() {
		return Range[T]{}, false
	}
	if !r.Overlaps(other) && !r.IsAdjacent(other) {
		return Range[T]{}, false
	}

	start := r.Start
	if other.Start < start {
		start = other.Start
	}
	end := r.End
	if other.End > end {
		end = other.End
	}
	return Range[T]{Start: start, End: end}, true
}
