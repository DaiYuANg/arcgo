package list

import "sync"

// ConcurrentDeque is a goroutine-safe deque wrapper.
// Zero value is ready to use.
type ConcurrentDeque[T any] struct {
	mu    sync.RWMutex
	deque *Deque[T]
}

// NewConcurrentDeque creates a concurrent deque with optional initial items.
func NewConcurrentDeque[T any](items ...T) *ConcurrentDeque[T] {
	return &ConcurrentDeque[T]{
		deque: NewDeque(items...),
	}
}

// PushFront inserts items at front in argument order.
func (d *ConcurrentDeque[T]) PushFront(items ...T) {
	if len(items) == 0 {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureInitLocked()
	d.deque.PushFront(items...)
}

// PushBack appends items at back.
func (d *ConcurrentDeque[T]) PushBack(items ...T) {
	if len(items) == 0 {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureInitLocked()
	d.deque.PushBack(items...)
}

// PopFront removes and returns front item.
func (d *ConcurrentDeque[T]) PopFront() (T, bool) {
	var zero T
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.deque == nil {
		return zero, false
	}
	return d.deque.PopFront()
}

// PopBack removes and returns back item.
func (d *ConcurrentDeque[T]) PopBack() (T, bool) {
	var zero T
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.deque == nil {
		return zero, false
	}
	return d.deque.PopBack()
}

// Front returns front item without removing it.
func (d *ConcurrentDeque[T]) Front() (T, bool) {
	var zero T
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.deque == nil {
		return zero, false
	}
	return d.deque.Front()
}

// Back returns back item without removing it.
func (d *ConcurrentDeque[T]) Back() (T, bool) {
	var zero T
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.deque == nil {
		return zero, false
	}
	return d.deque.Back()
}

// Get returns item at logical index from front.
func (d *ConcurrentDeque[T]) Get(index int) (T, bool) {
	var zero T
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.deque == nil {
		return zero, false
	}
	return d.deque.Get(index)
}

// Len returns item count.
func (d *ConcurrentDeque[T]) Len() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.deque == nil {
		return 0
	}
	return d.deque.Len()
}

// IsEmpty reports whether deque has no items.
func (d *ConcurrentDeque[T]) IsEmpty() bool {
	return d.Len() == 0
}

// Clear removes all items.
func (d *ConcurrentDeque[T]) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.deque == nil {
		return
	}
	d.deque.Clear()
}

// Values returns items from front to back.
func (d *ConcurrentDeque[T]) Values() []T {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.deque == nil {
		return nil
	}
	return d.deque.Values()
}

// Range iterates a stable snapshot from front to back until fn returns false.
func (d *ConcurrentDeque[T]) Range(fn func(index int, item T) bool) {
	if fn == nil {
		return
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.deque == nil {
		return
	}
	d.deque.Range(fn)
}

// Snapshot returns an immutable-style copy in a normal Deque.
func (d *ConcurrentDeque[T]) Snapshot() *Deque[T] {
	out := NewDeque[T]()
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.deque == nil {
		return out
	}
	d.deque.Range(func(_ int, item T) bool {
		out.PushBack(item)
		return true
	})
	return out
}

func (d *ConcurrentDeque[T]) ensureInitLocked() {
	if d.deque == nil {
		d.deque = NewDeque[T]()
	}
}
