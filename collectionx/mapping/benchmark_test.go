package mapping_test

import (
	"testing"

	mapping "github.com/DaiYuANg/arcgo/collectionx/mapping"
)

const (
	benchMapKeySpace       = 1 << 12
	benchTableDim          = 1 << 6
	benchMultiMapValueSeed = 8
)

func BenchmarkMapSetGet(b *testing.B) {
	m := mapping.NewMap[int, int]()
	mask := benchMapKeySpace - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		k := i & mask
		m.Set(k, i)
		_, _ = m.Get(k)
	}
}

func BenchmarkMapClone(b *testing.B) {
	m := mapping.NewMapWithCapacity[int, int](benchMapKeySpace)
	for i := range benchMapKeySpace {
		m.Set(i, i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		clone := m.Clone()
		if clone.Len() != benchMapKeySpace {
			b.Fatalf("unexpected clone length: %d", clone.Len())
		}
	}
}

func BenchmarkOrderedMapSetGet(b *testing.B) {
	m := mapping.NewOrderedMapWithCapacity[int, int](benchMapKeySpace)
	mask := benchMapKeySpace - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		k := i & mask
		m.Set(k, i)
		_, _ = m.Get(k)
	}
}

func BenchmarkOrderedMapValues(b *testing.B) {
	m := mapping.NewOrderedMapWithCapacity[int, int](benchMapKeySpace)
	for i := range benchMapKeySpace {
		m.Set(i, i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = m.Values()
	}
}

func BenchmarkBiMapPutGetByValue(b *testing.B) {
	m := mapping.NewBiMap[int, int]()
	mask := benchMapKeySpace - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		v := i & mask
		m.Put(v, v)
		_, _ = m.GetByKey(v)
		_, _ = m.GetByValue(v)
	}
}

func BenchmarkConcurrentMapGetParallel(b *testing.B) {
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
			_, _ = m.Get(i & mask)
			i++
		}
	})
}

func BenchmarkConcurrentMapGetOrStoreParallel(b *testing.B) {
	m := mapping.NewConcurrentMap[int, int]()
	mask := benchMapKeySpace - 1

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = m.GetOrStore(i&mask, i)
			i++
		}
	})
}

func BenchmarkMultiMapPutGet(b *testing.B) {
	m := mapping.NewMultiMap[int, int]()
	mask := benchMapKeySpace - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		k := i & mask
		m.Put(k, i)
		_ = m.Get(k)
	}
}

func BenchmarkMultiMapDeleteValueIf(b *testing.B) {
	m := mapping.NewMultiMapWithCapacity[int, int](benchMapKeySpace)
	for key := range benchMapKeySpace {
		for value := range benchMultiMapValueSeed {
			m.Put(key, value)
		}
	}
	mask := benchMapKeySpace - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		key := i & mask
		removed := m.DeleteValueIf(key, func(value int) bool { return value%2 == 0 })
		if removed > 0 {
			m.PutAll(key, 0, 2, 4, 6)
		}
	}
}

func BenchmarkConcurrentMultiMapGetParallel(b *testing.B) {
	m := mapping.NewConcurrentMultiMapWithCapacity[int, int](benchMapKeySpace)
	for key := range benchMapKeySpace {
		for value := range benchMultiMapValueSeed {
			m.Put(key, value)
		}
	}
	mask := benchMapKeySpace - 1

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = m.Get(i & mask)
			i++
		}
	})
}

func BenchmarkTablePutGet(b *testing.B) {
	t := mapping.NewTable[int, int, int]()
	rowMask := benchTableDim - 1
	colMask := benchTableDim - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		row := i & rowMask
		col := (i >> 6) & colMask
		t.Put(row, col, i)
		_, _ = t.Get(row, col)
	}
}

func BenchmarkTableRow(b *testing.B) {
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
		_ = t.Row(i & rowMask)
	}
}

func BenchmarkConcurrentTableGetParallel(b *testing.B) {
	t := mapping.NewConcurrentTable[int, int, int]()
	for row := range benchTableDim {
		for col := range benchTableDim {
			t.Put(row, col, row+col)
		}
	}

	rowMask := benchTableDim - 1
	colMask := benchTableDim - 1
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			row := i & rowMask
			col := (i >> 6) & colMask
			_, _ = t.Get(row, col)
			i++
		}
	})
}

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
