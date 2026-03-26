package interval_test

import (
	"testing"

	"github.com/DaiYuANg/arcgo/collectionx/interval"
)

const benchRangeSetSize = 1 << 10

func BenchmarkRangeContains(b *testing.B) {
	r := interval.Range[int]{Start: 128, End: 2048}
	mask := 4095

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		_ = r.Contains(i & mask)
	}
}

func BenchmarkRangeMerge(b *testing.B) {
	left := interval.Range[int]{Start: 100, End: 200}
	right := interval.Range[int]{Start: 150, End: 250}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		merged, ok := left.Merge(right)
		if !ok || merged.Start != 100 || merged.End != 250 {
			b.Fatalf("unexpected merge result: %+v ok=%v", merged, ok)
		}
	}
}

func BenchmarkRangeSetAdd(b *testing.B) {
	s := interval.NewRangeSet[int]()
	sizeMask := benchRangeSetSize - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		slot := i & sizeMask
		start := slot * 4
		s.Add(start, start+2)
	}
}

func BenchmarkRangeSetContains(b *testing.B) {
	s := interval.NewRangeSet[int]()
	for i := range benchRangeSetSize {
		start := i * 2
		s.Add(start, start+1)
	}

	mask := (benchRangeSetSize * 2) - 1
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		_ = s.Contains(i & mask)
	}
}

func BenchmarkRangeMapPutGet(b *testing.B) {
	m := interval.NewRangeMap[int, int]()
	sizeMask := benchRangeSetSize - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		slot := i & sizeMask
		start := slot * 2
		end := start + 2
		m.Put(start, end, i)
		_, _ = m.Get(start)
	}
}

func BenchmarkRangeMapGet(b *testing.B) {
	m := interval.NewRangeMap[int, int]()
	for i := range benchRangeSetSize {
		start := i * 4
		m.Put(start, start+3, i)
	}
	mask := benchRangeSetSize - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		point := (i & mask) * 4
		_, _ = m.Get(point)
	}
}

func BenchmarkRangeSetRemove(b *testing.B) {
	s := interval.NewRangeSet[int]()
	for i := range benchRangeSetSize {
		start := i * 2
		s.Add(start, start+1)
	}
	sizeMask := benchRangeSetSize - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		slot := i & sizeMask
		start := slot * 2
		s.Remove(start, start+1)
		s.Add(start, start+1)
	}
}

func BenchmarkRangeSetRanges(b *testing.B) {
	s := interval.NewRangeSet[int]()
	for i := range benchRangeSetSize {
		start := i * 2
		s.Add(start, start+1)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = s.Ranges()
	}
}

func BenchmarkRangeMapEntries(b *testing.B) {
	m := interval.NewRangeMap[int, int]()
	for i := range benchRangeSetSize {
		start := i * 4
		m.Put(start, start+3, i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = m.Entries()
	}
}

func BenchmarkRangeMapDeleteRange(b *testing.B) {
	m := interval.NewRangeMap[int, int]()
	for i := range benchRangeSetSize {
		start := i * 4
		m.Put(start, start+3, i)
	}
	sizeMask := benchRangeSetSize - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		slot := i & sizeMask
		start := slot * 4
		m.DeleteRange(start, start+3)
		m.Put(start, start+3, slot)
	}
}
