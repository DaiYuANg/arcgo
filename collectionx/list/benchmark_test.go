package list

import "testing"

const (
	benchListKeySpace         = 1 << 12
	benchRingBufferCapacity   = 1 << 10
	benchPriorityQueueSeedLen = 1 << 10
)

func newBenchPriorityQueue(b testing.TB) *PriorityQueue[int] {
	b.Helper()
	pq, err := NewPriorityQueue(func(a, c int) bool {
		return a < c
	})
	if err != nil {
		b.Fatalf("NewPriorityQueue() error = %v", err)
	}
	return pq
}

func BenchmarkListAppend(b *testing.B) {
	l := NewListWithCapacity[int](b.N)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Add(i)
	}
}

func BenchmarkListSetGet(b *testing.B) {
	l := NewListWithCapacity[int](benchListKeySpace)
	for i := 0; i < benchListKeySpace; i++ {
		l.Add(i)
	}

	mask := benchListKeySpace - 1
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index := i & mask
		l.Set(index, i)
		_, _ = l.Get(index)
	}
}

func BenchmarkListRemoveAtMiddle(b *testing.B) {
	l := NewListWithCapacity[int](benchListKeySpace)
	for i := 0; i < benchListKeySpace; i++ {
		l.Add(i)
	}
	mid := benchListKeySpace / 2

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = l.RemoveAt(mid)
		_ = l.AddAt(mid, i)
	}
}

func BenchmarkListClone(b *testing.B) {
	l := NewListWithCapacity[int](benchListKeySpace)
	for i := 0; i < benchListKeySpace; i++ {
		l.Add(i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clone := l.Clone()
		if clone.Len() != benchListKeySpace {
			b.Fatalf("unexpected clone length: %d", clone.Len())
		}
	}
}

func BenchmarkDequePushPop(b *testing.B) {
	d := NewDeque[int]()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.PushBack(i)
		_, _ = d.PopFront()
	}
}

func BenchmarkDequeGet(b *testing.B) {
	d := NewDeque[int]()
	for i := 0; i < benchListKeySpace; i++ {
		d.PushBack(i)
	}
	mask := benchListKeySpace - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = d.Get(i & mask)
	}
}

func BenchmarkConcurrentDequePushPopParallel(b *testing.B) {
	d := NewConcurrentDeque[int]()

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
	r := NewRingBuffer[int](benchRingBufferCapacity)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Push(i)
		_, _ = r.Pop()
	}
}

func BenchmarkRingBufferOverwrite(b *testing.B) {
	r := NewRingBuffer[int](benchRingBufferCapacity)
	for i := 0; i < benchRingBufferCapacity; i++ {
		_ = r.Push(i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Push(i)
	}
}

func BenchmarkConcurrentRingBufferPushParallel(b *testing.B) {
	r := NewConcurrentRingBuffer[int](benchRingBufferCapacity)

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
	for i := 0; i < b.N; i++ {
		pq.Push(i)
		_, _ = pq.Pop()
	}
}

func BenchmarkPriorityQueuePeek(b *testing.B) {
	pq := newBenchPriorityQueue(b)
	for i := 0; i < benchPriorityQueueSeedLen; i++ {
		pq.Push(i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pq.Peek()
	}
}

func BenchmarkConcurrentListGetParallel(b *testing.B) {
	l := NewConcurrentList[int]()
	for i := 0; i < benchListKeySpace; i++ {
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
	l := NewConcurrentList[int]()
	for i := 0; i < benchListKeySpace; i++ {
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
	l := NewListWithCapacity[int](benchListKeySpace)
	for i := 0; i < benchListKeySpace; i++ {
		l.Add(i)
	}
	mid := benchListKeySpace / 2

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = l.AddAt(mid, i)
		_, _ = l.RemoveAt(mid)
	}
}

func BenchmarkListRemoveIf(b *testing.B) {
	l := NewListWithCapacity[int](benchListKeySpace)
	for i := 0; i < benchListKeySpace; i++ {
		l.Add(i)
	}
	half := benchListKeySpace / 2

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.RemoveIf(func(x int) bool { return x%2 == 0 })
		for j := 0; j < half; j++ {
			l.Add(j * 2)
		}
	}
}

func BenchmarkDequePushFrontPopBack(b *testing.B) {
	d := NewDeque[int]()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.PushFront(i)
		_, _ = d.PopBack()
	}
}

func BenchmarkListRange(b *testing.B) {
	l := NewListWithCapacity[int](benchListKeySpace)
	for i := 0; i < benchListKeySpace; i++ {
		l.Add(i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Range(func(idx int, item int) bool {
			_ = item
			return true
		})
	}
}

func BenchmarkConcurrentListAddParallel(b *testing.B) {
	l := NewConcurrentList[int]()
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
	r := NewRopeList[int]()
	for i := 0; i < benchListKeySpace; i++ {
		r.Add(i)
	}
	mid := benchListKeySpace / 2

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.InsertAt(mid, i)
		_, _ = r.RemoveAt(mid)
	}
}

func BenchmarkRopeListGet(b *testing.B) {
	r := NewRopeList[int]()
	for i := 0; i < benchListKeySpace; i++ {
		r.Add(i)
	}
	mask := benchListKeySpace - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = r.Get(i & mask)
	}
}

func BenchmarkListAddAtLarge(b *testing.B) {
	const largeSize = 50_000
	l := NewListWithCapacity[int](largeSize)
	for i := 0; i < largeSize; i++ {
		l.Add(i)
	}
	mid := largeSize / 2

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = l.AddAt(mid, i)
		_, _ = l.RemoveAt(mid)
	}
}

func BenchmarkRopeListAddAtLarge(b *testing.B) {
	const largeSize = 50_000
	r := NewRopeList[int]()
	for i := 0; i < largeSize; i++ {
		r.Add(i)
	}
	mid := largeSize / 2

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.InsertAt(mid, i)
		_, _ = r.RemoveAt(mid)
	}
}
