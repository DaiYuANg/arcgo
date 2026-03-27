package list_test

import (
	"testing"

	list "github.com/DaiYuANg/arcgo/collectionx/list"
)

const (
	benchListKeySpace         = 1 << 12
	benchRingBufferCapacity   = 1 << 10
	benchPriorityQueueSeedLen = 1 << 10
)

func newBenchPriorityQueue(tb testing.TB) *list.PriorityQueue[int] {
	tb.Helper()
	pq, err := list.NewPriorityQueue(func(a, c int) bool {
		return a < c
	})
	if err != nil {
		tb.Fatalf("NewPriorityQueue() error = %v", err)
	}
	return pq
}

func BenchmarkListAppend(b *testing.B) {
	l := list.NewListWithCapacity[int](b.N)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		l.Add(i)
	}
}

func BenchmarkListSetGet(b *testing.B) {
	l := list.NewListWithCapacity[int](benchListKeySpace)
	for i := range benchListKeySpace {
		l.Add(i)
	}

	mask := benchListKeySpace - 1
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		index := i & mask
		l.Set(index, i)
		_, _ = l.Get(index)
	}
}

func BenchmarkListRemoveAtMiddle(b *testing.B) {
	l := list.NewListWithCapacity[int](benchListKeySpace)
	for i := range benchListKeySpace {
		l.Add(i)
	}
	mid := benchListKeySpace / 2

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		_, _ = l.RemoveAt(mid)
		_ = l.AddAt(mid, i)
	}
}

func BenchmarkListClone(b *testing.B) {
	l := list.NewListWithCapacity[int](benchListKeySpace)
	for i := range benchListKeySpace {
		l.Add(i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		clone := l.Clone()
		if clone.Len() != benchListKeySpace {
			b.Fatalf("unexpected clone length: %d", clone.Len())
		}
	}
}

func BenchmarkDequePushPop(b *testing.B) {
	d := list.NewDeque[int]()

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		d.PushBack(i)
		_, _ = d.PopFront()
	}
}

func BenchmarkDequeGet(b *testing.B) {
	d := list.NewDeque[int]()
	for i := range benchListKeySpace {
		d.PushBack(i)
	}
	mask := benchListKeySpace - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		_, _ = d.Get(i & mask)
	}
}

func BenchmarkConcurrentDequePushPopParallel(b *testing.B) {
	d := list.NewConcurrentDeque[int]()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			d.PushBack(i)
			_, _ = d.PopFront()
			i++
		}
	})
}

func BenchmarkRingBufferPushPop(b *testing.B) {
	r := list.NewRingBuffer[int](benchRingBufferCapacity)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		_ = r.Push(i)
		_, _ = r.Pop()
	}
}

func BenchmarkRingBufferOverwrite(b *testing.B) {
	r := list.NewRingBuffer[int](benchRingBufferCapacity)
	for i := range benchRingBufferCapacity {
		_ = r.Push(i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		_ = r.Push(i)
	}
}

func BenchmarkConcurrentRingBufferPushParallel(b *testing.B) {
	r := list.NewConcurrentRingBuffer[int](benchRingBufferCapacity)

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = r.Push(i)
			i++
		}
	})
}

func BenchmarkPriorityQueuePushPop(b *testing.B) {
	pq := newBenchPriorityQueue(b)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		pq.Push(i)
		_, _ = pq.Pop()
	}
}

func BenchmarkPriorityQueuePeek(b *testing.B) {
	pq := newBenchPriorityQueue(b)
	for i := range benchPriorityQueueSeedLen {
		pq.Push(i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _ = pq.Peek()
	}
}

func BenchmarkConcurrentListGetParallel(b *testing.B) {
	l := list.NewConcurrentList[int]()
	for i := range benchListKeySpace {
		l.Add(i)
	}

	mask := benchListKeySpace - 1
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = l.Get(i & mask)
			i++
		}
	})
}

func BenchmarkConcurrentListSetParallel(b *testing.B) {
	l := list.NewConcurrentList[int]()
	for i := range benchListKeySpace {
		l.Add(i)
	}

	mask := benchListKeySpace - 1
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = l.Set(i&mask, i)
			i++
		}
	})
}

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
