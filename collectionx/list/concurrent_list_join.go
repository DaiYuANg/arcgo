package list

// Join concatenates a stable snapshot of items with sep.
func (l *ConcurrentList[T]) Join(sep string, formatters ...func(index int, item T) string) string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.core == nil {
		return ""
	}
	return l.core.Join(sep, formatters...)
}
