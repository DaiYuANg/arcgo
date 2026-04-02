package list

import "github.com/samber/mo"

// Get returns item at index.
func (l *ConcurrentList[T]) Get(index int) (T, bool) {
	var zero T
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.core == nil {
		return zero, false
	}
	return l.core.Get(index)
}

// GetFirst returns the first item.
func (l *ConcurrentList[T]) GetFirst() (T, bool) {
	return l.Get(0)
}

// GetOption returns item at index as mo.Option.
func (l *ConcurrentList[T]) GetOption(index int) mo.Option[T] {
	value, ok := l.Get(index)
	if !ok {
		return mo.None[T]()
	}
	return mo.Some(value)
}

// GetFirstOption returns the first item as mo.Option.
func (l *ConcurrentList[T]) GetFirstOption() mo.Option[T] {
	return l.GetOption(0)
}

// GetLast returns the last item.
func (l *ConcurrentList[T]) GetLast() (T, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.core == nil {
		var zero T
		return zero, false
	}
	return l.core.GetLast()
}

// GetLastOption returns the last item as mo.Option.
func (l *ConcurrentList[T]) GetLastOption() mo.Option[T] {
	value, ok := l.GetLast()
	if !ok {
		return mo.None[T]()
	}
	return mo.Some(value)
}
