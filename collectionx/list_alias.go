package collectionx

import "github.com/DaiYuANg/arcgo/collectionx/list"

type List[T any] = list.List[T]

func NewList[T any](items ...T) *List[T] {
	return list.NewList(items...)
}

type ConcurrentList[T any] = list.ConcurrentList[T]

func NewConcurrentList[T any](items ...T) *ConcurrentList[T] {
	return list.NewConcurrentList(items...)
}

type Deque[T any] = list.Deque[T]

func NewDeque[T any](items ...T) *Deque[T] {
	return list.NewDeque(items...)
}

type ConcurrentDeque[T any] = list.ConcurrentDeque[T]

func NewConcurrentDeque[T any](items ...T) *ConcurrentDeque[T] {
	return list.NewConcurrentDeque(items...)
}

type RingBuffer[T any] = list.RingBuffer[T]

func NewRingBuffer[T any](capacity int) *RingBuffer[T] {
	return list.NewRingBuffer[T](capacity)
}

type ConcurrentRingBuffer[T any] = list.ConcurrentRingBuffer[T]

func NewConcurrentRingBuffer[T any](capacity int) *ConcurrentRingBuffer[T] {
	return list.NewConcurrentRingBuffer[T](capacity)
}

type PriorityQueue[T any] = list.PriorityQueue[T]

func NewPriorityQueue[T any](less func(a, b T) bool, items ...T) *PriorityQueue[T] {
	return list.NewPriorityQueue(less, items...)
}
