package list

import (
	"sync"

	"github.com/samber/mo"
)

// ConcurrentRingBuffer is a goroutine-safe fixed-capacity ring buffer.
// Zero value is ready to use.
type ConcurrentRingBuffer[T any] struct {
	mu     sync.RWMutex
	buffer *RingBuffer[T]
}

// NewConcurrentRingBuffer creates a concurrent ring buffer with fixed capacity.
func NewConcurrentRingBuffer[T any](capacity int) *ConcurrentRingBuffer[T] {
	return &ConcurrentRingBuffer[T]{
		buffer: NewRingBuffer[T](capacity),
	}
}

// Capacity returns max item capacity.
func (r *ConcurrentRingBuffer[T]) Capacity() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.buffer == nil {
		return 0
	}
	return r.buffer.Capacity()
}

// Len returns current item count.
func (r *ConcurrentRingBuffer[T]) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.buffer == nil {
		return 0
	}
	return r.buffer.Len()
}

// IsEmpty reports whether buffer has no items.
func (r *ConcurrentRingBuffer[T]) IsEmpty() bool {
	return r.Len() == 0
}

// IsFull reports whether buffer reached capacity.
func (r *ConcurrentRingBuffer[T]) IsFull() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.buffer == nil {
		return false
	}
	return r.buffer.IsFull()
}

// Push writes value at tail.
// If full, oldest value is evicted and returned as mo.Option.
func (r *ConcurrentRingBuffer[T]) Push(value T) mo.Option[T] {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.buffer == nil {
		return mo.None[T]()
	}
	return r.buffer.Push(value)
}

// Pop removes and returns oldest value.
func (r *ConcurrentRingBuffer[T]) Pop() (T, bool) {
	var zero T
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.buffer == nil {
		return zero, false
	}
	return r.buffer.Pop()
}

// Peek returns oldest value without removing it.
func (r *ConcurrentRingBuffer[T]) Peek() (T, bool) {
	var zero T
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.buffer == nil {
		return zero, false
	}
	return r.buffer.Peek()
}

// Values returns items from oldest to newest.
func (r *ConcurrentRingBuffer[T]) Values() []T {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.buffer == nil {
		return nil
	}
	return r.buffer.Values()
}

// Clear removes all values.
func (r *ConcurrentRingBuffer[T]) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.buffer == nil {
		return
	}
	r.buffer.Clear()
}

// Range iterates a stable snapshot from oldest to newest until fn returns false.
func (r *ConcurrentRingBuffer[T]) Range(fn func(index int, item T) bool) {
	if fn == nil {
		return
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.buffer == nil || r.buffer.size == 0 {
		return
	}
	for index := range r.buffer.size {
		item := r.buffer.buf[(r.buffer.head+index)%len(r.buffer.buf)]
		if !fn(index, item) {
			return
		}
	}
}

// Snapshot returns an immutable-style copy in a normal RingBuffer.
func (r *ConcurrentRingBuffer[T]) Snapshot() *RingBuffer[T] {
	out := NewRingBuffer[T](0)
	if r == nil {
		return out
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.buffer == nil {
		return out
	}

	out = NewRingBuffer[T](r.buffer.Capacity())
	for index := range r.buffer.size {
		item := r.buffer.buf[(r.buffer.head+index)%len(r.buffer.buf)]
		_ = out.Push(item)
	}
	return out
}
