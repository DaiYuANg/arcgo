package collectionx

import (
	"fmt"

	"github.com/DaiYuANg/arcgo/collectionx/list"
	"github.com/samber/mo"
)

type listReadable[T any] interface {
	Get(index int) (T, bool)
	GetOption(index int) mo.Option[T]
	sized
	Values() []T
	Range(fn func(index int, item T) bool)
}

type listWritable[T any] interface {
	Add(items ...T)
	AddAt(index int, item T) bool
	AddAllAt(index int, items ...T) bool
	Set(index int, item T) bool
	SetAll(mapper func(item T) T) int
	SetAllIndexed(mapper func(index int, item T) T) int
	RemoveAt(index int) (T, bool)
	RemoveAtOption(index int) mo.Option[T]
	RemoveIf(predicate func(item T) bool) int
	clearable
}

type listFluent[T any] interface {
	Where(predicate func(index int, item T) bool) *list.List[T]
	Reject(predicate func(index int, item T) bool) *list.List[T]
	Take(n int) *list.List[T]
	Drop(n int) *list.List[T]
	Each(fn func(index int, item T)) *list.List[T]
	FirstWhere(predicate func(index int, item T) bool) mo.Option[T]
	AnyMatch(predicate func(index int, item T) bool) bool
	AllMatch(predicate func(index int, item T) bool) bool
}

// List is the root list interface exposed by collectionx.
type List[T any] interface {
	listReadable[T]
	listWritable[T]
	listFluent[T]
	Merge(other *list.List[T]) *list.List[T]
	MergeSlice(items []T) *list.List[T]
	clonable[*list.List[T]]
	jsonStringer
}

// NewList creates a List populated with items.
func NewList[T any](items ...T) List[T] {
	return list.NewList(items...)
}

// NewListWithCapacity creates a List with preallocated capacity and optional items.
func NewListWithCapacity[T any](capacity int, items ...T) List[T] {
	return list.NewListWithCapacity(capacity, items...)
}

// NewRopeList creates a RopeList optimized for frequent AddAt and RemoveAt calls.
func NewRopeList[T any](items ...T) *list.RopeList[T] {
	return list.NewRopeList(items...)
}

// NewRopeListWithCapacity creates a RopeList with preallocated capacity and optional items.
func NewRopeListWithCapacity[T any](capacity int, items ...T) *list.RopeList[T] {
	return list.NewRopeListWithCapacity(capacity, items...)
}

// ConcurrentList is the thread-safe root list interface exposed by collectionx.
type ConcurrentList[T any] interface {
	listReadable[T]
	listWritable[T]
	listFluent[T]
	Merge(other *list.List[T]) *list.ConcurrentList[T]
	MergeSlice(items []T) *list.ConcurrentList[T]
	MergeConcurrent(other *list.ConcurrentList[T]) *list.ConcurrentList[T]
	snapshotable[*list.List[T]]
	jsonStringer
}

// NewConcurrentList creates a ConcurrentList populated with items.
func NewConcurrentList[T any](items ...T) ConcurrentList[T] {
	return list.NewConcurrentList(items...)
}

// NewConcurrentListWithCapacity creates a ConcurrentList with preallocated capacity and optional items.
func NewConcurrentListWithCapacity[T any](capacity int, items ...T) ConcurrentList[T] {
	return list.NewConcurrentListWithCapacity(capacity, items...)
}

type dequeReadable[T any] interface {
	PopFront() (T, bool)
	PopBack() (T, bool)
	Front() (T, bool)
	Back() (T, bool)
	Get(index int) (T, bool)
	sized
	Values() []T
	Range(fn func(index int, item T) bool)
}

type dequeWritable[T any] interface {
	PushFront(items ...T)
	PushBack(items ...T)
	clearable
}

// Deque is the root double-ended queue interface exposed by collectionx.
type Deque[T any] interface {
	dequeReadable[T]
	dequeWritable[T]
	jsonStringer
}

// NewDeque creates a Deque populated with items.
func NewDeque[T any](items ...T) Deque[T] {
	return list.NewDeque(items...)
}

// ConcurrentDeque is the thread-safe root deque interface exposed by collectionx.
type ConcurrentDeque[T any] interface {
	dequeReadable[T]
	dequeWritable[T]
	snapshotable[*list.Deque[T]]
	jsonStringer
}

// NewConcurrentDeque creates a ConcurrentDeque populated with items.
func NewConcurrentDeque[T any](items ...T) ConcurrentDeque[T] {
	return list.NewConcurrentDeque(items...)
}

type ringReadable[T any] interface {
	Capacity() int
	sized
	IsFull() bool
	Pop() (T, bool)
	Peek() (T, bool)
	Values() []T
}

type ringWritable[T any] interface {
	Push(value T) mo.Option[T]
	clearable
}

// RingBuffer is the root fixed-capacity ring buffer interface exposed by collectionx.
type RingBuffer[T any] interface {
	ringReadable[T]
	ringWritable[T]
	jsonStringer
}

// NewRingBuffer creates a RingBuffer with the provided capacity.
func NewRingBuffer[T any](capacity int) RingBuffer[T] {
	return list.NewRingBuffer[T](capacity)
}

// ConcurrentRingBuffer is the thread-safe root ring buffer interface exposed by collectionx.
type ConcurrentRingBuffer[T any] interface {
	ringReadable[T]
	ringWritable[T]
	Range(fn func(index int, item T) bool)
	snapshotable[*list.RingBuffer[T]]
	jsonStringer
}

// NewConcurrentRingBuffer creates a ConcurrentRingBuffer with the provided capacity.
func NewConcurrentRingBuffer[T any](capacity int) ConcurrentRingBuffer[T] {
	return list.NewConcurrentRingBuffer[T](capacity)
}

// PriorityQueue is the root priority queue interface exposed by collectionx.
type PriorityQueue[T any] interface {
	Push(value T)
	Pop() (T, bool)
	Peek() (T, bool)
	sized
	clearable
	Values() []T
	ValuesSorted() []T
	jsonStringer
}

// NewPriorityQueue creates a PriorityQueue using less to order items.
func NewPriorityQueue[T any](less func(a, b T) bool, items ...T) (PriorityQueue[T], error) {
	queue, err := list.NewPriorityQueue(less, items...)
	if err != nil {
		return nil, fmt.Errorf("new priority queue: %w", err)
	}
	return queue, nil
}
