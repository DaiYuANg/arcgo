package collectionx

import (
	"github.com/DaiYuANg/arcgo/collectionx/set"
	"github.com/samber/mo"
)

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

type setFluent[T comparable] interface {
	Where(predicate func(item T) bool) *set.Set[T]
	Reject(predicate func(item T) bool) *set.Set[T]
	Each(fn func(item T)) *set.Set[T]
	FirstWhere(predicate func(item T) bool) mo.Option[T]
	AnyMatch(predicate func(item T) bool) bool
	AllMatch(predicate func(item T) bool) bool
}

// Set is the root set interface exposed by collectionx.
type Set[T comparable] interface {
	setReadable[T]
	setWritable[T]
	setFluent[T]
	Merge(other *set.Set[T]) *set.Set[T]
	MergeSlice(items []T) *set.Set[T]
	clonable[*set.Set[T]]
	Union(other *set.Set[T]) *set.Set[T]
	Intersect(other *set.Set[T]) *set.Set[T]
	Difference(other *set.Set[T]) *set.Set[T]
	jsonStringer
}

// NewSet creates a Set populated with items.
func NewSet[T comparable](items ...T) Set[T] {
	return set.NewSet(items...)
}

// NewSetWithCapacity creates a Set with preallocated capacity and optional items.
func NewSetWithCapacity[T comparable](capacity int, items ...T) Set[T] {
	return set.NewSetWithCapacity(capacity, items...)
}

// ConcurrentSet is the thread-safe root set interface exposed by collectionx.
type ConcurrentSet[T comparable] interface {
	setReadable[T]
	setWritable[T]
	setFluent[T]
	Merge(other *set.Set[T]) *set.ConcurrentSet[T]
	MergeConcurrent(other *set.ConcurrentSet[T]) *set.ConcurrentSet[T]
	MergeSlice(items []T) *set.ConcurrentSet[T]
	AddIfAbsent(item T) bool
	snapshotable[*set.Set[T]]
	jsonStringer
}

// NewConcurrentSet creates a ConcurrentSet populated with items.
func NewConcurrentSet[T comparable](items ...T) ConcurrentSet[T] {
	return set.NewConcurrentSet(items...)
}

// NewConcurrentSetWithCapacity creates a ConcurrentSet with preallocated capacity and optional items.
func NewConcurrentSetWithCapacity[T comparable](capacity int, items ...T) ConcurrentSet[T] {
	return set.NewConcurrentSetWithCapacity(capacity, items...)
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

// MultiSet is the root multiset interface exposed by collectionx.
type MultiSet[T comparable] interface {
	multiSetReadable[T]
	multiSetWritable[T]
	jsonStringer
}

// NewMultiSet creates a MultiSet populated with items.
func NewMultiSet[T comparable](items ...T) MultiSet[T] {
	return set.NewMultiSet(items...)
}

// NewMultiSetWithCapacity creates a MultiSet with preallocated capacity and optional items.
func NewMultiSetWithCapacity[T comparable](capacity int, items ...T) MultiSet[T] {
	return set.NewMultiSetWithCapacity(capacity, items...)
}

type orderedSetReadable[T comparable] interface {
	setReadable[T]
	At(pos int) (T, bool)
}

type orderedSetFluent[T comparable] interface {
	Where(predicate func(item T) bool) *set.OrderedSet[T]
	Reject(predicate func(item T) bool) *set.OrderedSet[T]
	Take(n int) *set.OrderedSet[T]
	Drop(n int) *set.OrderedSet[T]
	Each(fn func(item T)) *set.OrderedSet[T]
	FirstWhere(predicate func(item T) bool) mo.Option[T]
	AnyMatch(predicate func(item T) bool) bool
	AllMatch(predicate func(item T) bool) bool
}

// OrderedSet is the root ordered set interface exposed by collectionx.
type OrderedSet[T comparable] interface {
	orderedSetReadable[T]
	setWritable[T]
	orderedSetFluent[T]
	clonable[*set.OrderedSet[T]]
	jsonStringer
}

// NewOrderedSet creates an OrderedSet populated with items.
func NewOrderedSet[T comparable](items ...T) OrderedSet[T] {
	return set.NewOrderedSet(items...)
}

// NewOrderedSetWithCapacity creates an OrderedSet with preallocated capacity and optional items.
func NewOrderedSetWithCapacity[T comparable](capacity int, items ...T) OrderedSet[T] {
	return set.NewOrderedSetWithCapacity(capacity, items...)
}
