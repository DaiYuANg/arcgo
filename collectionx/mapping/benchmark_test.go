package mapping

import "testing"

const (
	benchMapKeySpace       = 1 << 12
	benchTableDim          = 1 << 6
	benchMultiMapValueSeed = 8
)

func BenchmarkMapSetGet(b *testing.B) {
	m := NewMap[int, int]()
	mask := benchMapKeySpace - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		k := i & mask
		m.Set(k, i)
		_, _ = m.Get(k)
	}
}

func BenchmarkMapClone(b *testing.B) {
	m := NewMapWithCapacity[int, int](benchMapKeySpace)
	for i := 0; i < benchMapKeySpace; i++ {
		m.Set(i, i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clone := m.Clone()
		if clone.Len() != benchMapKeySpace {
			b.Fatalf("unexpected clone length: %d", clone.Len())
		}
	}
}

func BenchmarkOrderedMapSetGet(b *testing.B) {
	m := NewOrderedMapWithCapacity[int, int](benchMapKeySpace)
	mask := benchMapKeySpace - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		k := i & mask
		m.Set(k, i)
		_, _ = m.Get(k)
	}
}

func BenchmarkOrderedMapValues(b *testing.B) {
	m := NewOrderedMapWithCapacity[int, int](benchMapKeySpace)
	for i := 0; i < benchMapKeySpace; i++ {
		m.Set(i, i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.Values()
	}
}

func BenchmarkBiMapPutGetByValue(b *testing.B) {
	m := NewBiMap[int, int]()
	mask := benchMapKeySpace - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := i & mask
		m.Put(v, v)
		_, _ = m.GetByKey(v)
		_, _ = m.GetByValue(v)
	}
}

func BenchmarkConcurrentMapGetParallel(b *testing.B) {
	m := NewConcurrentMap[int, int]()
	for i := 0; i < benchMapKeySpace; i++ {
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
	m := NewConcurrentMap[int, int]()
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
	m := NewMultiMap[int, int]()
	mask := benchMapKeySpace - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		k := i & mask
		m.Put(k, i)
		_ = m.Get(k)
	}
}

func BenchmarkMultiMapDeleteValueIf(b *testing.B) {
	m := NewMultiMapWithCapacity[int, int](benchMapKeySpace)
	for key := 0; key < benchMapKeySpace; key++ {
		for value := 0; value < benchMultiMapValueSeed; value++ {
			m.Put(key, value)
		}
	}
	mask := benchMapKeySpace - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := i & mask
		removed := m.DeleteValueIf(key, func(value int) bool { return value%2 == 0 })
		if removed > 0 {
			m.PutAll(key, 0, 2, 4, 6)
		}
	}
}

func BenchmarkConcurrentMultiMapGetParallel(b *testing.B) {
	m := NewConcurrentMultiMapWithCapacity[int, int](benchMapKeySpace)
	for key := 0; key < benchMapKeySpace; key++ {
		for value := 0; value < benchMultiMapValueSeed; value++ {
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
	t := NewTable[int, int, int]()
	rowMask := benchTableDim - 1
	colMask := benchTableDim - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		row := i & rowMask
		col := (i >> 6) & colMask
		t.Put(row, col, i)
		_, _ = t.Get(row, col)
	}
}

func BenchmarkTableRow(b *testing.B) {
	t := NewTable[int, int, int]()
	for row := 0; row < benchTableDim; row++ {
		for col := 0; col < benchTableDim; col++ {
			t.Put(row, col, row+col)
		}
	}
	rowMask := benchTableDim - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = t.Row(i & rowMask)
	}
}

func BenchmarkConcurrentTableGetParallel(b *testing.B) {
	t := NewConcurrentTable[int, int, int]()
	for row := 0; row < benchTableDim; row++ {
		for col := 0; col < benchTableDim; col++ {
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
	m := NewMapWithCapacity[int, int](benchMapKeySpace)
	for i := 0; i < benchMapKeySpace; i++ {
		m.Set(i, i)
	}
	mask := benchMapKeySpace - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		k := i & mask
		m.Delete(k)
		m.Set(k, i)
	}
}

func BenchmarkMapKeys(b *testing.B) {
	m := NewMapWithCapacity[int, int](benchMapKeySpace)
	for i := 0; i < benchMapKeySpace; i++ {
		m.Set(i, i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.Keys()
	}
}

func BenchmarkMapValues(b *testing.B) {
	m := NewMapWithCapacity[int, int](benchMapKeySpace)
	for i := 0; i < benchMapKeySpace; i++ {
		m.Set(i, i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.Values()
	}
}

func BenchmarkMapAll(b *testing.B) {
	m := NewMapWithCapacity[int, int](benchMapKeySpace)
	for i := 0; i < benchMapKeySpace; i++ {
		m.Set(i, i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.All()
	}
}

func BenchmarkConcurrentMapSetParallel(b *testing.B) {
	m := NewConcurrentMap[int, int]()
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
	m := NewConcurrentMap[int, int]()
	for i := 0; i < benchMapKeySpace; i++ {
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
	t := NewTable[int, int, int]()
	for row := 0; row < benchTableDim; row++ {
		for col := 0; col < benchTableDim; col++ {
			t.Put(row, col, row+col)
		}
	}
	colMask := benchTableDim - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = t.Column(i & colMask)
	}
}

func BenchmarkTableDeleteRow(b *testing.B) {
	t := NewTable[int, int, int]()
	for row := 0; row < benchTableDim; row++ {
		for col := 0; col < benchTableDim; col++ {
			t.Put(row, col, row+col)
		}
	}
	rowMask := benchTableDim - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		row := i & rowMask
		t.DeleteRow(row)
		for col := 0; col < benchTableDim; col++ {
			t.Put(row, col, row+col)
		}
	}
}

func BenchmarkTableDeleteColumn(b *testing.B) {
	t := NewTable[int, int, int]()
	for row := 0; row < benchTableDim; row++ {
		for col := 0; col < benchTableDim; col++ {
			t.Put(row, col, row+col)
		}
	}
	colMask := benchTableDim - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		col := i & colMask
		t.DeleteColumn(col)
		for row := 0; row < benchTableDim; row++ {
			t.Put(row, col, row+col)
		}
	}
}

func BenchmarkOrderedMapKeys(b *testing.B) {
	m := NewOrderedMapWithCapacity[int, int](benchMapKeySpace)
	for i := 0; i < benchMapKeySpace; i++ {
		m.Set(i, i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.Keys()
	}
}

func BenchmarkShardedConcurrentMapGetParallel(b *testing.B) {
	m := NewShardedConcurrentMap[int, int](32, HashInt)
	for i := 0; i < benchMapKeySpace; i++ {
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
	m := NewShardedConcurrentMap[int, int](32, HashInt)
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
