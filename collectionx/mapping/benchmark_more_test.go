package mapping_test

import (
	"testing"

	mapping "github.com/DaiYuANg/arcgo/collectionx/mapping"
)

func BenchmarkMapDelete(b *testing.B) {
	m := mapping.NewMapWithCapacity[int, int](benchMapKeySpace)
	for i := range benchMapKeySpace {
		m.Set(i, i)
	}
	mask := benchMapKeySpace - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		k := i & mask
		m.Delete(k)
		m.Set(k, i)
	}
}

func BenchmarkMapKeys(b *testing.B) {
	m := mapping.NewMapWithCapacity[int, int](benchMapKeySpace)
	for i := range benchMapKeySpace {
		m.Set(i, i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = m.Keys()
	}
}

func BenchmarkMapValues(b *testing.B) {
	m := mapping.NewMapWithCapacity[int, int](benchMapKeySpace)
	for i := range benchMapKeySpace {
		m.Set(i, i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = m.Values()
	}
}

func BenchmarkMapAll(b *testing.B) {
	m := mapping.NewMapWithCapacity[int, int](benchMapKeySpace)
	for i := range benchMapKeySpace {
		m.Set(i, i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = m.All()
	}
}

func BenchmarkConcurrentMapSetParallel(b *testing.B) {
	m := mapping.NewConcurrentMap[int, int]()
	mask := benchMapKeySpace - 1

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			m.Set(i&mask, i)
			i++
		}
	})
}

func BenchmarkConcurrentMapDeleteParallel(b *testing.B) {
	m := mapping.NewConcurrentMap[int, int]()
	for i := range benchMapKeySpace {
		m.Set(i, i)
	}
	mask := benchMapKeySpace - 1

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			k := i & mask
			m.Delete(k)
			m.Set(k, i)
			i++
		}
	})
}

func BenchmarkTableColumn(b *testing.B) {
	t := mapping.NewTable[int, int, int]()
	for row := range benchTableDim {
		for col := range benchTableDim {
			t.Put(row, col, row+col)
		}
	}
	colMask := benchTableDim - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		_ = t.Column(i & colMask)
	}
}

func BenchmarkTableColumnKeys(b *testing.B) {
	t := mapping.NewTable[int, int, int]()
	for row := range benchTableDim {
		for col := range benchTableDim {
			t.Put(row, col, row+col)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = t.ColumnKeys()
	}
}

func BenchmarkOrderedMapToJSON(b *testing.B) {
	m := mapping.NewOrderedMapWithCapacity[int, int](benchMapKeySpace)
	for i := range benchMapKeySpace {
		m.Set(i, i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := m.ToJSON(); err != nil {
			b.Fatalf("ordered map to json failed: %v", err)
		}
	}
}

func BenchmarkMultiMapToJSON(b *testing.B) {
	m := mapping.NewMultiMapWithCapacity[int, int](benchMapKeySpace)
	for key := range benchMapKeySpace {
		for value := range benchMultiMapValueSeed {
			m.Put(key, value)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := m.ToJSON(); err != nil {
			b.Fatalf("multimap to json failed: %v", err)
		}
	}
}

func BenchmarkTableToJSON(b *testing.B) {
	t := mapping.NewTable[int, int, int]()
	for row := range benchTableDim {
		for col := range benchTableDim {
			t.Put(row, col, row+col)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := t.ToJSON(); err != nil {
			b.Fatalf("table to json failed: %v", err)
		}
	}
}

func BenchmarkConcurrentMapToJSON(b *testing.B) {
	m := mapping.NewConcurrentMap[int, int]()
	for i := range benchMapKeySpace {
		m.Set(i, i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := m.ToJSON(); err != nil {
			b.Fatalf("concurrent map to json failed: %v", err)
		}
	}
}

func BenchmarkConcurrentTableToJSON(b *testing.B) {
	t := mapping.NewConcurrentTable[int, int, int]()
	for row := range benchTableDim {
		for col := range benchTableDim {
			t.Put(row, col, row+col)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := t.ToJSON(); err != nil {
			b.Fatalf("concurrent table to json failed: %v", err)
		}
	}
}

func BenchmarkTableDeleteRow(b *testing.B) {
	t := mapping.NewTable[int, int, int]()
	for row := range benchTableDim {
		for col := range benchTableDim {
			t.Put(row, col, row+col)
		}
	}
	rowMask := benchTableDim - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		row := i & rowMask
		t.DeleteRow(row)
		for col := range benchTableDim {
			t.Put(row, col, row+col)
		}
	}
}

func BenchmarkTableDeleteColumn(b *testing.B) {
	t := mapping.NewTable[int, int, int]()
	for row := range benchTableDim {
		for col := range benchTableDim {
			t.Put(row, col, row+col)
		}
	}
	colMask := benchTableDim - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		col := i & colMask
		t.DeleteColumn(col)
		for row := range benchTableDim {
			t.Put(row, col, row+col)
		}
	}
}

func BenchmarkOrderedMapKeys(b *testing.B) {
	m := mapping.NewOrderedMapWithCapacity[int, int](benchMapKeySpace)
	for i := range benchMapKeySpace {
		m.Set(i, i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = m.Keys()
	}
}

func BenchmarkShardedConcurrentMapGetParallel(b *testing.B) {
	m := mapping.NewShardedConcurrentMap[int, int](32, mapping.HashInt)
	for i := range benchMapKeySpace {
		m.Set(i, i)
	}
	mask := benchMapKeySpace - 1

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = m.Get(i & mask)
			i++
		}
	})
}

func BenchmarkShardedConcurrentMapSetParallel(b *testing.B) {
	m := mapping.NewShardedConcurrentMap[int, int](32, mapping.HashInt)
	mask := benchMapKeySpace - 1

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			m.Set(i&mask, i)
			i++
		}
	})
}
