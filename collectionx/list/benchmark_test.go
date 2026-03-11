package list

import "testing"

const benchListKeySpace = 1 << 12

func BenchmarkListSetGet(b *testing.B) {
	l := NewList[int]()
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

func BenchmarkDequePushPop(b *testing.B) {
	d := NewDeque[int]()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.PushBack(i)
		_, _ = d.PopFront()
	}
}

func BenchmarkPriorityQueuePushPop(b *testing.B) {
	pq, err := NewPriorityQueue(func(a, c int) bool {
		return a < c
	})
	if err != nil {
		b.Fatalf("NewPriorityQueue() error = %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pq.Push(i)
		_, _ = pq.Pop()
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
