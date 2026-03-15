package collectionx

import (
	"github.com/DaiYuANg/archgo/collectionx/list"
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

type List[T any] interface {
	listReadable[T]
	listWritable[T]
	Merge(other *list.List[T]) *list.List[T]
	MergeSlice(items []T) *list.List[T]
	clonable[*list.List[T]]
	jsonStringer
}

func NewList[T any](items ...T) List[T] {
	return list.NewList(items...)
}

type ConcurrentList[T any] interface {
	listReadable[T]
	listWritable[T]
	Merge(other *list.List[T]) *list.ConcurrentList[T]
	MergeSlice(items []T) *list.ConcurrentList[T]
	MergeConcurrent(other *list.ConcurrentList[T]) *list.ConcurrentList[T]
	snapshotable[*list.List[T]]
	jsonStringer
}

func NewConcurrentList[T any](items ...T) ConcurrentList[T] {
	return list.NewConcurrentList(items...)
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

type Deque[T any] interface {
	dequeReadable[T]
	dequeWritable[T]
	jsonStringer
}

func NewDeque[T any](items ...T) Deque[T] {
	return list.NewDeque(items...)
}

type ConcurrentDeque[T any] interface {
	dequeReadable[T]
	dequeWritable[T]
	snapshotable[*list.Deque[T]]
	jsonStringer
}

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

type RingBuffer[T any] interface {
	ringReadable[T]
	ringWritable[T]
	jsonStringer
}

func NewRingBuffer[T any](capacity int) RingBuffer[T] {
	return list.NewRingBuffer[T](capacity)
}

type ConcurrentRingBuffer[T any] interface {
	ringReadable[T]
	ringWritable[T]
	Range(fn func(index int, item T) bool)
	snapshotable[*list.RingBuffer[T]]
	jsonStringer
}

func NewConcurrentRingBuffer[T any](capacity int) ConcurrentRingBuffer[T] {
	return list.NewConcurrentRingBuffer[T](capacity)
}

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

func NewPriorityQueue[T any](less func(a, b T) bool, items ...T) (PriorityQueue[T], error) {
	return list.NewPriorityQueue(less, items...)
}
