package collectionx

import (
	"github.com/DaiYuANg/archgo/collectionx/mapping"
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

type Map[K comparable, V any] interface {
	mapReadable[K, V]
	mapWritable[K, V]
	clonable[*mapping.Map[K, V]]
	jsonStringer
}

func NewMap[K comparable, V any]() Map[K, V] {
	return mapping.NewMap[K, V]()
}

func NewMapFrom[K comparable, V any](source map[K]V) Map[K, V] {
	return mapping.NewMapFrom(source)
}

type ConcurrentMap[K comparable, V any] interface {
	mapReadable[K, V]
	mapWritable[K, V]
	GetOrStore(key K, value V) (actual V, loaded bool)
	LoadAndDelete(key K) (V, bool)
	LoadAndDeleteOption(key K) mo.Option[V]
	jsonStringer
}

func NewConcurrentMap[K comparable, V any]() ConcurrentMap[K, V] {
	return mapping.NewConcurrentMap[K, V]()
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

type BiMap[K comparable, V comparable] interface {
	biMapReadable[K, V]
	biMapWritable[K, V]
	jsonStringer
}

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

type OrderedMap[K comparable, V any] interface {
	orderedMapReadable[K, V]
	orderedMapWritable[K, V]
	clonable[*mapping.OrderedMap[K, V]]
	jsonStringer
}

func NewOrderedMap[K comparable, V any]() OrderedMap[K, V] {
	return mapping.NewOrderedMap[K, V]()
}

type multiMapReadable[K comparable, V any] interface {
	Get(key K) []V
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

type MultiMap[K comparable, V any] interface {
	multiMapReadable[K, V]
	multiMapWritable[K, V]
	jsonStringer
}

func NewMultiMap[K comparable, V any]() MultiMap[K, V] {
	return mapping.NewMultiMap[K, V]()
}

func NewMultiMapFromAll[K comparable, V any](source map[K][]V) MultiMap[K, V] {
	return mapping.NewMultiMapFromAll(source)
}

type ConcurrentMultiMap[K comparable, V any] interface {
	multiMapReadable[K, V]
	multiMapWritable[K, V]
	snapshotable[*mapping.MultiMap[K, V]]
	jsonStringer
}

func NewConcurrentMultiMap[K comparable, V any]() ConcurrentMultiMap[K, V] {
	return mapping.NewConcurrentMultiMap[K, V]()
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

type Table[R comparable, C comparable, V any] interface {
	tableReadable[R, C, V]
	tableWritable[R, C, V]
	jsonStringer
}

func NewTable[R comparable, C comparable, V any]() Table[R, C, V] {
	return mapping.NewTable[R, C, V]()
}

type ConcurrentTable[R comparable, C comparable, V any] interface {
	tableReadable[R, C, V]
	tableWritable[R, C, V]
	snapshotable[*mapping.Table[R, C, V]]
	jsonStringer
}

func NewConcurrentTable[R comparable, C comparable, V any]() ConcurrentTable[R, C, V] {
	return mapping.NewConcurrentTable[R, C, V]()
}
