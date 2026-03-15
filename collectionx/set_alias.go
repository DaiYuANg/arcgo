package collectionx

import "github.com/DaiYuANg/archgo/collectionx/set"

type setReadable[T comparable] interface {
	Contains(item T) bool
	sized
	Values() []T
	Range(fn func(item T) bool)
}

type setWritable[T comparable] interface {
	Add(items ...T)
	Remove(item T) bool
	clearable
}

type Set[T comparable] interface {
	setReadable[T]
	setWritable[T]
	Merge(other *set.Set[T]) *set.Set[T]
	MergeSlice(items []T) *set.Set[T]
	clonable[*set.Set[T]]
	Union(other *set.Set[T]) *set.Set[T]
	Intersect(other *set.Set[T]) *set.Set[T]
	Difference(other *set.Set[T]) *set.Set[T]
	jsonStringer
}

func NewSet[T comparable](items ...T) Set[T] {
	return set.NewSet(items...)
}

type ConcurrentSet[T comparable] interface {
	setReadable[T]
	setWritable[T]
	Merge(other *set.Set[T]) *set.ConcurrentSet[T]
	MergeConcurrent(other *set.ConcurrentSet[T]) *set.ConcurrentSet[T]
	MergeSlice(items []T) *set.ConcurrentSet[T]
	AddIfAbsent(item T) bool
	snapshotable[*set.Set[T]]
	jsonStringer
}

func NewConcurrentSet[T comparable](items ...T) ConcurrentSet[T] {
	return set.NewConcurrentSet(items...)
}

type multiSetReadable[T comparable] interface {
	Count(item T) int
	Contains(item T) bool
	sized
	UniqueLen() int
	Distinct() []T
	Elements() []T
	AllCounts() map[T]int
	Range(fn func(item T, count int) bool)
}

type multiSetWritable[T comparable] interface {
	Add(items ...T)
	AddN(item T, n int)
	Remove(item T) bool
	RemoveN(item T, n int) int
	clearable
}

type MultiSet[T comparable] interface {
	multiSetReadable[T]
	multiSetWritable[T]
	jsonStringer
}

func NewMultiSet[T comparable](items ...T) MultiSet[T] {
	return set.NewMultiSet(items...)
}

type orderedSetReadable[T comparable] interface {
	setReadable[T]
	At(pos int) (T, bool)
}

type OrderedSet[T comparable] interface {
	orderedSetReadable[T]
	setWritable[T]
	clonable[*set.OrderedSet[T]]
	jsonStringer
}

func NewOrderedSet[T comparable](items ...T) OrderedSet[T] {
	return set.NewOrderedSet(items...)
}
