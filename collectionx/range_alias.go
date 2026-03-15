package collectionx

import (
	"cmp"

	"github.com/DaiYuANg/archgo/collectionx/interval"
	"github.com/samber/mo"
)

type Range[T cmp.Ordered] = interval.Range[T]

func NewRange[T cmp.Ordered](start T, end T) (Range[T], bool) {
	return interval.NewRange(start, end)
}

type RangeSet[T cmp.Ordered] interface {
	rangeSetWritable[T]
	rangeSetReadable[T]
	jsonStringer
}

type rangeSetWritable[T cmp.Ordered] interface {
	Add(start T, end T) bool
	AddRange(r interval.Range[T]) bool
	Remove(start T, end T) bool
	clearable
}

type rangeSetReadable[T cmp.Ordered] interface {
	Contains(value T) bool
	Overlaps(start T, end T) bool
	Ranges() []interval.Range[T]
	sized
	Range(fn func(r interval.Range[T]) bool)
}

func NewRangeSet[T cmp.Ordered]() RangeSet[T] {
	return interval.NewRangeSet[T]()
}

type RangeEntry[T cmp.Ordered, V any] = interval.RangeEntry[T, V]

type RangeMap[T cmp.Ordered, V any] interface {
	rangeMapWritable[T, V]
	rangeMapReadable[T, V]
	jsonStringer
}

type rangeMapWritable[T cmp.Ordered, V any] interface {
	Put(start T, end T, value V) bool
	DeleteRange(start T, end T) bool
	clearable
}

type rangeMapReadable[T cmp.Ordered, V any] interface {
	Get(point T) (V, bool)
	GetOption(point T) mo.Option[V]
	Entries() []interval.RangeEntry[T, V]
	sized
	Range(fn func(entry interval.RangeEntry[T, V]) bool)
}

func NewRangeMap[T cmp.Ordered, V any]() RangeMap[T, V] {
	return interval.NewRangeMap[T, V]()
}
