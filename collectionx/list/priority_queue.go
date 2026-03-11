package list

import (
	"container/heap"
	"errors"
)

var ErrNilPriorityQueueComparator = errors.New("list: priority queue comparator cannot be nil")

// PriorityQueue is a generic heap-based queue.
// less(a, b) should return true when a has higher priority than b.
type PriorityQueue[T any] struct {
	h *priorityQueueHeap[T]
}

// NewPriorityQueue creates a priority queue with comparator and optional items.
func NewPriorityQueue[T any](less func(a, b T) bool, items ...T) (*PriorityQueue[T], error) {
	if less == nil {
		return nil, ErrNilPriorityQueueComparator
	}
	pq := &PriorityQueue[T]{
		h: &priorityQueueHeap[T]{
			items: make([]T, 0, len(items)),
			less:  less,
		},
	}
	pq.h.items = append(pq.h.items, items...)
	heap.Init(pq.h)
	return pq, nil
}

// Push inserts value.
func (pq *PriorityQueue[T]) Push(value T) {
	if pq == nil || pq.h == nil {
		return
	}
	heap.Push(pq.h, value)
}

// Pop removes and returns top-priority value.
func (pq *PriorityQueue[T]) Pop() (T, bool) {
	var zero T
	if pq == nil || pq.h == nil || pq.h.Len() == 0 {
		return zero, false
	}
	value, ok := heap.Pop(pq.h).(T)
	if !ok {
		return zero, false
	}
	return value, true
}

// Peek returns top-priority value without removing it.
func (pq *PriorityQueue[T]) Peek() (T, bool) {
	var zero T
	if pq == nil || pq.h == nil || pq.h.Len() == 0 {
		return zero, false
	}
	return pq.h.items[0], true
}

// Len returns item count.
func (pq *PriorityQueue[T]) Len() int {
	if pq == nil || pq.h == nil {
		return 0
	}
	return pq.h.Len()
}

// IsEmpty reports whether queue has no items.
func (pq *PriorityQueue[T]) IsEmpty() bool {
	return pq.Len() == 0
}

// Clear removes all items.
func (pq *PriorityQueue[T]) Clear() {
	if pq == nil || pq.h == nil {
		return
	}
	pq.h.items = nil
}

// Values returns a snapshot of internal heap order (not sorted order).
func (pq *PriorityQueue[T]) Values() []T {
	if pq == nil || pq.h == nil || pq.h.Len() == 0 {
		return nil
	}
	out := make([]T, len(pq.h.items))
	copy(out, pq.h.items)
	return out
}

// ValuesSorted returns a sorted snapshot by popping from a temporary heap copy.
func (pq *PriorityQueue[T]) ValuesSorted() []T {
	if pq == nil || pq.h == nil || pq.h.Len() == 0 {
		return nil
	}
	temp := &priorityQueueHeap[T]{
		items: make([]T, len(pq.h.items)),
		less:  pq.h.less,
	}
	copy(temp.items, pq.h.items)
	heap.Init(temp)

	out := make([]T, 0, temp.Len())
	for temp.Len() > 0 {
		value, ok := heap.Pop(temp).(T)
		if !ok {
			break
		}
		out = append(out, value)
	}
	return out
}

type priorityQueueHeap[T any] struct {
	items []T
	less  func(a, b T) bool
}

func (h *priorityQueueHeap[T]) Len() int {
	return len(h.items)
}

func (h *priorityQueueHeap[T]) Less(i int, j int) bool {
	return h.less(h.items[i], h.items[j])
}

func (h *priorityQueueHeap[T]) Swap(i int, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
}

func (h *priorityQueueHeap[T]) Push(x any) {
	value, ok := x.(T)
	if !ok {
		return
	}
	h.items = append(h.items, value)
}

func (h *priorityQueueHeap[T]) Pop() any {
	old := h.items
	n := len(old)
	value := old[n-1]
	h.items = old[:n-1]
	return value
}
