package list

import "github.com/samber/mo"

// Where returns a new list containing only items that match predicate.
func (l *List[T]) Where(predicate func(index int, item T) bool) *List[T] {
	if l == nil || predicate == nil || len(l.items) == 0 {
		return NewList[T]()
	}
	filtered := NewListWithCapacity[T](len(l.items))
	for index, item := range l.items {
		if predicate(index, item) {
			filtered.Add(item)
		}
	}
	return filtered
}

// Reject returns a new list excluding items that match predicate.
func (l *List[T]) Reject(predicate func(index int, item T) bool) *List[T] {
	if l == nil || predicate == nil || len(l.items) == 0 {
		return NewList[T]()
	}
	rejected := NewListWithCapacity[T](len(l.items))
	for index, item := range l.items {
		if !predicate(index, item) {
			rejected.Add(item)
		}
	}
	return rejected
}

// Take returns the first n items as a new list.
func (l *List[T]) Take(n int) *List[T] {
	if l == nil || n <= 0 || len(l.items) == 0 {
		return NewList[T]()
	}
	if n >= len(l.items) {
		return l.Clone()
	}
	return NewList(l.items[:n]...)
}

// Drop returns a new list without the first n items.
func (l *List[T]) Drop(n int) *List[T] {
	if l == nil || len(l.items) == 0 {
		return NewList[T]()
	}
	if n <= 0 {
		return l.Clone()
	}
	if n >= len(l.items) {
		return NewList[T]()
	}
	return NewList(l.items[n:]...)
}

// Each invokes fn for every item and returns the receiver for chaining.
func (l *List[T]) Each(fn func(index int, item T)) *List[T] {
	if l == nil {
		return NewList[T]()
	}
	if fn == nil {
		return l
	}
	for index, item := range l.items {
		fn(index, item)
	}
	return l
}

// FirstWhere returns the first item matching predicate.
func (l *List[T]) FirstWhere(predicate func(index int, item T) bool) mo.Option[T] {
	if l == nil || predicate == nil || len(l.items) == 0 {
		return mo.None[T]()
	}
	for index, item := range l.items {
		if predicate(index, item) {
			return mo.Some(item)
		}
	}
	return mo.None[T]()
}

// AnyMatch reports whether any item matches predicate.
func (l *List[T]) AnyMatch(predicate func(index int, item T) bool) bool {
	_, ok := l.FirstWhere(predicate).Get()
	return ok
}

// AllMatch reports whether all items match predicate.
func (l *List[T]) AllMatch(predicate func(index int, item T) bool) bool {
	if l == nil || len(l.items) == 0 || predicate == nil {
		return false
	}
	for index, item := range l.items {
		if !predicate(index, item) {
			return false
		}
	}
	return true
}
