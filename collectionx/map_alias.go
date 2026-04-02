package collectionx

import (
	"github.com/DaiYuANg/arcgo/collectionx/mapping"
	"github.com/samber/mo"
)

type mapReadable[K comparable, V any] interface {
	Get(key K) (V, bool)
	GetOption(key K) mo.Option[V]
	GetOrDefault(key K, fallback V) V
	sized
	Keys() []K
	Values() []V
	All() map[K]V
	Range(fn func(key K, value V) bool)
}

type mapWritable[K comparable, V any] interface {
	Set(key K, value V)
	SetAll(source map[K]V)
	Delete(key K) bool
	clearable
}

type mapFluent[K comparable, V any] interface {
	WhereEntries(predicate func(key K, value V) bool) *mapping.Map[K, V]
	RejectEntries(predicate func(key K, value V) bool) *mapping.Map[K, V]
	EachEntry(fn func(key K, value V)) *mapping.Map[K, V]
	FirstEntryWhere(predicate func(key K, value V) bool) (K, V, bool)
	AnyEntryMatch(predicate func(key K, value V) bool) bool
	AllEntryMatch(predicate func(key K, value V) bool) bool
}

// Map is the root map interface exposed by collectionx.
type Map[K comparable, V any] interface {
	mapReadable[K, V]
	mapWritable[K, V]
	mapFluent[K, V]
	clonable[*mapping.Map[K, V]]
	jsonStringer
}

// NewMap creates an empty Map.
func NewMap[K comparable, V any]() Map[K, V] {
	return mapping.NewMap[K, V]()
}

// NewMapWithCapacity creates an empty Map with reserved capacity.
func NewMapWithCapacity[K comparable, V any](capacity int) Map[K, V] {
	return mapping.NewMapWithCapacity[K, V](capacity)
}

// NewMapFrom creates a Map initialized from source.
func NewMapFrom[K comparable, V any](source map[K]V) Map[K, V] {
	return mapping.NewMapFrom(source)
}

// ConcurrentMap is the thread-safe root map interface exposed by collectionx.
type ConcurrentMap[K comparable, V any] interface {
	mapReadable[K, V]
	mapWritable[K, V]
	mapFluent[K, V]
	GetOrStore(key K, value V) (actual V, loaded bool)
	LoadAndDelete(key K) (V, bool)
	LoadAndDeleteOption(key K) mo.Option[V]
	jsonStringer
}

// NewConcurrentMap creates an empty ConcurrentMap.
func NewConcurrentMap[K comparable, V any]() ConcurrentMap[K, V] {
	return mapping.NewConcurrentMap[K, V]()
}

// NewConcurrentMapWithCapacity creates an empty ConcurrentMap with reserved capacity.
func NewConcurrentMapWithCapacity[K comparable, V any](capacity int) ConcurrentMap[K, V] {
	return mapping.NewConcurrentMapWithCapacity[K, V](capacity)
}

// ShardedConcurrentMap is a ConcurrentMap with per-shard locks for lower contention.
// Use NewShardedConcurrentMap with a hash function for key distribution.
type ShardedConcurrentMap[K comparable, V any] interface {
	mapReadable[K, V]
	mapWritable[K, V]
	GetOrStore(key K, value V) (actual V, loaded bool)
	LoadAndDelete(key K) (V, bool)
	LoadAndDeleteOption(key K) mo.Option[V]
	jsonStringer
}

// NewShardedConcurrentMap creates a sharded concurrent map.
func NewShardedConcurrentMap[K comparable, V any](shardCount int, hash func(K) uint64) ShardedConcurrentMap[K, V] {
	return mapping.NewShardedConcurrentMap[K, V](shardCount, hash)
}

type biMapReadable[K comparable, V comparable] interface {
	GetByKey(key K) (V, bool)
	GetByValue(value V) (K, bool)
	GetValueOption(key K) mo.Option[V]
	GetKeyOption(value V) mo.Option[K]
	ContainsKey(key K) bool
	ContainsValue(value V) bool
	sized
	Keys() []K
	Values() []V
	All() map[K]V
	Inverse() map[V]K
	Range(fn func(key K, value V) bool)
}

type biMapWritable[K comparable, V comparable] interface {
	Put(key K, value V)
	DeleteByKey(key K) bool
	DeleteByValue(value V) bool
	clearable
}

// BiMap is the root bidirectional map interface exposed by collectionx.
type BiMap[K comparable, V comparable] interface {
	biMapReadable[K, V]
	biMapWritable[K, V]
	jsonStringer
}

// NewBiMap creates an empty BiMap.
func NewBiMap[K comparable, V comparable]() BiMap[K, V] {
	return mapping.NewBiMap[K, V]()
}

type orderedMapReadable[K comparable, V any] interface {
	Get(key K) (V, bool)
	GetOption(key K) mo.Option[V]
	At(pos int) (K, V, bool)
	sized
	Keys() []K
	Values() []V
	All() map[K]V
	Range(fn func(key K, value V) bool)
}

type orderedMapWritable[K comparable, V any] interface {
	Set(key K, value V)
	Delete(key K) bool
	clearable
}

type orderedMapFluent[K comparable, V any] interface {
	WhereEntries(predicate func(key K, value V) bool) *mapping.OrderedMap[K, V]
	RejectEntries(predicate func(key K, value V) bool) *mapping.OrderedMap[K, V]
	Take(n int) *mapping.OrderedMap[K, V]
	Drop(n int) *mapping.OrderedMap[K, V]
	EachEntry(fn func(key K, value V)) *mapping.OrderedMap[K, V]
	FirstEntryWhere(predicate func(key K, value V) bool) (K, V, bool)
	AnyEntryMatch(predicate func(key K, value V) bool) bool
	AllEntryMatch(predicate func(key K, value V) bool) bool
}

// OrderedMap is the root insertion-ordered map interface exposed by collectionx.
type OrderedMap[K comparable, V any] interface {
	orderedMapReadable[K, V]
	orderedMapWritable[K, V]
	orderedMapFluent[K, V]
	clonable[*mapping.OrderedMap[K, V]]
	jsonStringer
}

// NewOrderedMap creates an empty OrderedMap.
func NewOrderedMap[K comparable, V any]() OrderedMap[K, V] {
	return mapping.NewOrderedMap[K, V]()
}

// NewOrderedMapWithCapacity creates an empty OrderedMap with reserved capacity.
func NewOrderedMapWithCapacity[K comparable, V any](capacity int) OrderedMap[K, V] {
	return mapping.NewOrderedMapWithCapacity[K, V](capacity)
}

type multiMapReadable[K comparable, V any] interface {
	Get(key K) []V
	GetCopy(key K) []V
	GetOption(key K) mo.Option[[]V]
	ContainsKey(key K) bool
	sized
	ValueCount() int
	Keys() []K
	All() map[K][]V
	Range(fn func(key K, values []V) bool)
}

type multiMapWritable[K comparable, V any] interface {
	Put(key K, value V)
	PutAll(key K, values ...V)
	Set(key K, values ...V)
	Delete(key K) bool
	DeleteValueIf(key K, predicate func(value V) bool) int
	clearable
}

// MultiMap is the root multimap interface exposed by collectionx.
type MultiMap[K comparable, V any] interface {
	multiMapReadable[K, V]
	multiMapWritable[K, V]
	multiMapFluent[K, V]
	clonable[*mapping.MultiMap[K, V]]
	jsonStringer
}

// NewMultiMap creates an empty MultiMap.
func NewMultiMap[K comparable, V any]() MultiMap[K, V] {
	return mapping.NewMultiMap[K, V]()
}

// NewMultiMapWithCapacity creates an empty MultiMap with reserved capacity.
func NewMultiMapWithCapacity[K comparable, V any](capacity int) MultiMap[K, V] {
	return mapping.NewMultiMapWithCapacity[K, V](capacity)
}

// NewMultiMapFromAll creates a MultiMap initialized from source.
func NewMultiMapFromAll[K comparable, V any](source map[K][]V) MultiMap[K, V] {
	return mapping.NewMultiMapFromAll(source)
}

// ConcurrentMultiMap is the thread-safe root multimap interface exposed by collectionx.
type ConcurrentMultiMap[K comparable, V any] interface {
	multiMapReadable[K, V]
	multiMapWritable[K, V]
	multiMapFluent[K, V]
	snapshotable[*mapping.MultiMap[K, V]]
	jsonStringer
}

// NewConcurrentMultiMap creates an empty ConcurrentMultiMap.
func NewConcurrentMultiMap[K comparable, V any]() ConcurrentMultiMap[K, V] {
	return mapping.NewConcurrentMultiMap[K, V]()
}

// NewConcurrentMultiMapWithCapacity creates an empty ConcurrentMultiMap with reserved capacity.
func NewConcurrentMultiMapWithCapacity[K comparable, V any](capacity int) ConcurrentMultiMap[K, V] {
	return mapping.NewConcurrentMultiMapWithCapacity[K, V](capacity)
}

type tableReadable[R comparable, C comparable, V any] interface {
	Get(rowKey R, columnKey C) (V, bool)
	GetOption(rowKey R, columnKey C) mo.Option[V]
	Row(rowKey R) map[C]V
	Column(columnKey C) map[R]V
	Has(rowKey R, columnKey C) bool
	RowCount() int
	sized
	RowKeys() []R
	ColumnKeys() []C
	All() map[R]map[C]V
	Range(fn func(rowKey R, columnKey C, value V) bool)
}

type tableWritable[R comparable, C comparable, V any] interface {
	Put(rowKey R, columnKey C, value V)
	SetRow(rowKey R, rowValues map[C]V)
	Delete(rowKey R, columnKey C) bool
	DeleteRow(rowKey R) bool
	DeleteColumn(columnKey C) int
	clearable
}

// Table is the root two-dimensional table interface exposed by collectionx.
type Table[R comparable, C comparable, V any] interface {
	tableReadable[R, C, V]
	tableWritable[R, C, V]
	tableFluent[R, C, V]
	jsonStringer
}

// NewTable creates an empty Table.
func NewTable[R comparable, C comparable, V any]() Table[R, C, V] {
	return mapping.NewTable[R, C, V]()
}

// ConcurrentTable is the thread-safe root table interface exposed by collectionx.
type ConcurrentTable[R comparable, C comparable, V any] interface {
	tableReadable[R, C, V]
	tableWritable[R, C, V]
	tableFluent[R, C, V]
	snapshotable[*mapping.Table[R, C, V]]
	jsonStringer
}

// NewConcurrentTable creates an empty ConcurrentTable.
func NewConcurrentTable[R comparable, C comparable, V any]() ConcurrentTable[R, C, V] {
	return mapping.NewConcurrentTable[R, C, V]()
}
