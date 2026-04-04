package collectionx

import (
	"fmt"

	"github.com/DaiYuANg/arcgo/collectionx/list"
)

// List is the root list type exposed by collectionx.
type List[T any] = *list.List[T]

// NewList creates a List populated with items.
func NewList[T any](items ...T) List[T] {
	return list.NewList(items...)
}

// NewListWithCapacity creates a List with preallocated capacity and optional items.
func NewListWithCapacity[T any](capacity int, items ...T) List[T] {
	return list.NewListWithCapacity(capacity, items...)
}

// Grid is the root ordered two-dimensional container exposed by collectionx.
type Grid[T any] = *list.Grid[T]

// NewGrid creates a Grid populated with rows.
func NewGrid[T any](rows ...[]T) Grid[T] {
	return list.NewGrid(rows...)
}

// NewGridWithCapacity creates a Grid with preallocated row capacity and optional rows.
func NewGridWithCapacity[T any](capacity int, rows ...[]T) Grid[T] {
	return list.NewGridWithCapacity(capacity, rows...)
}

// ConcurrentGrid is the thread-safe root ordered two-dimensional container exposed by collectionx.
type ConcurrentGrid[T any] = *list.ConcurrentGrid[T]

// NewConcurrentGrid creates a ConcurrentGrid populated with rows.
func NewConcurrentGrid[T any](rows ...[]T) ConcurrentGrid[T] {
	return list.NewConcurrentGrid(rows...)
}

// NewConcurrentGridWithCapacity creates a ConcurrentGrid with preallocated row capacity and optional rows.
func NewConcurrentGridWithCapacity[T any](capacity int, rows ...[]T) ConcurrentGrid[T] {
	return list.NewConcurrentGridWithCapacity(capacity, rows...)
}

// RopeList aliases list.RopeList in the root collectionx package.
type RopeList[T any] = *list.RopeList[T]

// NewRopeList creates a RopeList optimized for frequent AddAt and RemoveAt calls.
func NewRopeList[T any](items ...T) RopeList[T] {
	return list.NewRopeList(items...)
}

// NewRopeListWithCapacity creates a RopeList with preallocated capacity and optional items.
func NewRopeListWithCapacity[T any](capacity int, items ...T) RopeList[T] {
	return list.NewRopeListWithCapacity(capacity, items...)
}

// ConcurrentList is the thread-safe root list type exposed by collectionx.
type ConcurrentList[T any] = *list.ConcurrentList[T]

// NewConcurrentList creates a ConcurrentList populated with items.
func NewConcurrentList[T any](items ...T) ConcurrentList[T] {
	return list.NewConcurrentList(items...)
}

// NewConcurrentListWithCapacity creates a ConcurrentList with preallocated capacity and optional items.
func NewConcurrentListWithCapacity[T any](capacity int, items ...T) ConcurrentList[T] {
	return list.NewConcurrentListWithCapacity(capacity, items...)
}

// Deque is the root double-ended queue type exposed by collectionx.
type Deque[T any] = *list.Deque[T]

// NewDeque creates a Deque populated with items.
func NewDeque[T any](items ...T) Deque[T] {
	return list.NewDeque(items...)
}

// ConcurrentDeque is the thread-safe root deque type exposed by collectionx.
type ConcurrentDeque[T any] = *list.ConcurrentDeque[T]

// NewConcurrentDeque creates a ConcurrentDeque populated with items.
func NewConcurrentDeque[T any](items ...T) ConcurrentDeque[T] {
	return list.NewConcurrentDeque(items...)
}

// RingBuffer is the root fixed-capacity ring buffer type exposed by collectionx.
type RingBuffer[T any] = *list.RingBuffer[T]

// NewRingBuffer creates a RingBuffer with the provided capacity.
func NewRingBuffer[T any](capacity int) RingBuffer[T] {
	return list.NewRingBuffer[T](capacity)
}

// ConcurrentRingBuffer is the thread-safe root ring buffer type exposed by collectionx.
type ConcurrentRingBuffer[T any] = *list.ConcurrentRingBuffer[T]

// NewConcurrentRingBuffer creates a ConcurrentRingBuffer with the provided capacity.
func NewConcurrentRingBuffer[T any](capacity int) ConcurrentRingBuffer[T] {
	return list.NewConcurrentRingBuffer[T](capacity)
}

// PriorityQueue is the root priority queue type exposed by collectionx.
type PriorityQueue[T any] = *list.PriorityQueue[T]

// NewPriorityQueue creates a PriorityQueue using less to order items.
func NewPriorityQueue[T any](less func(a, b T) bool, items ...T) (PriorityQueue[T], error) {
	queue, err := list.NewPriorityQueue(less, items...)
	if err != nil {
		return nil, fmt.Errorf("new priority queue: %w", err)
	}
	return queue, nil
}
