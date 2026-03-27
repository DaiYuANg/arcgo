package mapping_test

import (
	"strconv"
	"sync"
	"testing"

	mapping "github.com/DaiYuANg/arcgo/collectionx/mapping"
	"github.com/stretchr/testify/require"
)

func TestConcurrentMap_ParallelSet(t *testing.T) {
	t.Parallel()

	var m mapping.ConcurrentMap[int, int]

	const workers = 20
	const each = 200

	var wg sync.WaitGroup
	wg.Add(workers)

	for worker := range workers {
		go func() {
			defer wg.Done()
			base := worker * each
			for i := range each {
				m.Set(base+i, i)
			}
		}()
	}

	wg.Wait()
	require.Equal(t, workers*each, m.Len())
}

func TestConcurrentMap_GetOrStore(t *testing.T) {
	t.Parallel()

	var m mapping.ConcurrentMap[string, int]

	value, loaded := m.GetOrStore("a", 1)
	require.False(t, loaded)
	require.Equal(t, 1, value)

	value, loaded = m.GetOrStore("a", 9)
	require.True(t, loaded)
	require.Equal(t, 1, value)
}

func TestConcurrentMap_LoadAndDelete(t *testing.T) {
	t.Parallel()

	var m mapping.ConcurrentMap[string, string]
	m.Set("k", "v")

	value, ok := m.LoadAndDelete("k")
	require.True(t, ok)
	require.Equal(t, "v", value)

	_, ok = m.Get("k")
	require.False(t, ok)
}

func TestConcurrentMap_OptionAPIs(t *testing.T) {
	t.Parallel()

	var m mapping.ConcurrentMap[string, int]
	m.Set("x", 42)

	opt := m.GetOption("x")
	require.True(t, opt.IsPresent())
	value, ok := opt.Get()
	require.True(t, ok)
	require.Equal(t, 42, value)

	deleted := m.LoadAndDeleteOption("x")
	require.True(t, deleted.IsPresent())
	deletedValue, ok := deleted.Get()
	require.True(t, ok)
	require.Equal(t, 42, deletedValue)

	require.True(t, m.GetOption("x").IsAbsent())
}

func TestConcurrentMap_Range(t *testing.T) {
	t.Parallel()

	m := mapping.NewConcurrentMap[string, int]()
	for i := range 10 {
		m.Set(strconv.Itoa(i), i)
	}

	visited := 0
	m.Range(func(_ string, _ int) bool {
		visited++
		return visited < 3
	})
	require.Equal(t, 3, visited)
}

func TestNewConcurrentMapWithCapacity(t *testing.T) {
	t.Parallel()

	m := mapping.NewConcurrentMapWithCapacity[string, int](8)
	m.Set("a", 1)

	value, ok := m.Get("a")
	require.True(t, ok)
	require.Equal(t, 1, value)
}
