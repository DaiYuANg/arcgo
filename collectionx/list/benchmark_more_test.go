package list_test

import (
	"testing"

	list "github.com/DaiYuANg/arcgo/collectionx/list"
)

func BenchmarkListAddAt(b *testing.B) {
	l := list.NewListWithCapacity[int](benchListKeySpace)
	for i := range benchListKeySpace {
		l.Add(i)
	}
	mid := benchListKeySpace / 2

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		_ = l.AddAt(mid, i)
		_, _ = l.RemoveAt(mid)
	}
}

func BenchmarkListRemoveIf(b *testing.B) {
	l := list.NewListWithCapacity[int](benchListKeySpace)
	for i := range benchListKeySpace {
		l.Add(i)
	}
	half := benchListKeySpace / 2

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		l.RemoveIf(func(x int) bool { return x%2 == 0 })
		for j := range half {
			l.Add(j * 2)
		}
	}
}

func BenchmarkDequePushFrontPopBack(b *testing.B) {
	d := list.NewDeque[int]()

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		d.PushFront(i)
		_, _ = d.PopBack()
	}
}

func BenchmarkListRange(b *testing.B) {
	l := list.NewListWithCapacity[int](benchListKeySpace)
	for i := range benchListKeySpace {
		l.Add(i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		l.Range(func(_ int, item int) bool {
			_ = item
			return true
		})
	}
}

func BenchmarkConcurrentListAddParallel(b *testing.B) {
	l := list.NewConcurrentList[int]()
	mask := benchListKeySpace - 1

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			l.Add(i & mask)
			i++
		}
	})
}

func BenchmarkRopeListAddAt(b *testing.B) {
	r := list.NewRopeList[int]()
	for i := range benchListKeySpace {
		r.Add(i)
	}
	mid := benchListKeySpace / 2

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		_ = r.InsertAt(mid, i)
		_, _ = r.RemoveAt(mid)
	}
}

func BenchmarkRopeListGet(b *testing.B) {
	r := list.NewRopeList[int]()
	for i := range benchListKeySpace {
		r.Add(i)
	}
	mask := benchListKeySpace - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		_, _ = r.Get(i & mask)
	}
}

func BenchmarkListAddAtLarge(b *testing.B) {
	const largeSize = 50_000
	l := list.NewListWithCapacity[int](largeSize)
	for i := range largeSize {
		l.Add(i)
	}
	mid := largeSize / 2

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		_ = l.AddAt(mid, i)
		_, _ = l.RemoveAt(mid)
	}
}

func BenchmarkRopeListAddAtLarge(b *testing.B) {
	const largeSize = 50_000
	r := list.NewRopeList[int]()
	for i := range largeSize {
		r.Add(i)
	}
	mid := largeSize / 2

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		_ = r.InsertAt(mid, i)
		_, _ = r.RemoveAt(mid)
	}
}
