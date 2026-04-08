package mapping_test

import (
	"sync"
	"testing"

	mapping "github.com/DaiYuANg/arcgo/collectionx/mapping"
	"github.com/stretchr/testify/require"
)

func TestConcurrentMultiMap_ParallelPut(t *testing.T) {
	t.Parallel()

	var m mapping.ConcurrentMultiMap[int, int]

	const workers = 16
	const each = 120

	var wg sync.WaitGroup
	wg.Add(workers)

	for worker := range workers {
		go func() {
			defer wg.Done()
			for i := range each {
				m.Put(worker, i)
			}
		}()
	}

	wg.Wait()
	require.Equal(t, workers, m.Len())
	require.Equal(t, workers*each, m.ValueCount())
}

func TestConcurrentMultiMap_OptionAndSnapshot(t *testing.T) {
	t.Parallel()

	var m mapping.ConcurrentMultiMap[string, int]
	m.PutAll("a", 1, 2, 3)

	view := m.Get("a")
	m.Put("a", 4)
	require.Equal(t, []int{1, 2, 3}, view)
	require.Equal(t, []int{1, 2, 3, 4}, m.Get("a"))

	copyValues := m.GetCopy("a")
	copyValues[0] = 99
	require.Equal(t, []int{1, 2, 3, 4}, m.Get("a"))

	opt := m.GetOption("a")
	require.True(t, opt.IsPresent())
	values, ok := opt.Get()
	require.True(t, ok)
	require.Equal(t, []int{1, 2, 3, 4}, values)

	snapshot := m.Snapshot()
	m.Put("a", 5)
	require.Equal(t, []int{1, 2, 3, 4}, snapshot.Get("a"))
}

func TestNewConcurrentMultiMapWithCapacity(t *testing.T) {
	t.Parallel()

	m := mapping.NewConcurrentMultiMapWithCapacity[string, int](8)
	m.PutAll("a", 1, 2)

	require.Equal(t, 1, m.Len())
	require.Equal(t, 2, m.ValueCount())
}

func TestConcurrentMultiMap_SnapshotCountsAndIsolation(t *testing.T) {
	t.Parallel()

	var m mapping.ConcurrentMultiMap[string, int]
	m.PutAll("a", 1, 2)
	m.Put("b", 3)

	snapshot := m.Snapshot()
	require.Equal(t, 2, snapshot.Len())
	require.Equal(t, 3, snapshot.ValueCount())

	m.Delete("b")
	m.Put("a", 4)

	require.Equal(t, []int{1, 2}, snapshot.Get("a"))
	require.Equal(t, []int{1, 2, 4}, m.Get("a"))
	require.True(t, snapshot.ContainsKey("b"))
	require.False(t, m.ContainsKey("b"))
}

func TestConcurrentMultiMap_FluentOps(t *testing.T) {
	t.Parallel()

	var values mapping.ConcurrentMultiMap[string, int]
	values.PutAll("a", 1, 2)
	values.PutAll("b", 3, 4)
	values.Put("c", 5)

	filtered := values.
		WhereKeys(func(key string, _ []int) bool { return key != "c" }).
		WhereValues(func(_ string, value int) bool { return value >= 2 })

	require.Equal(t, []int{2}, filtered.Get("a"))
	require.Equal(t, []int{3, 4}, filtered.Get("b"))
	require.False(t, filtered.ContainsKey("c"))

	flattened := values.FlattenValues()
	require.ElementsMatch(t, []int{1, 2, 3, 4, 5}, flattened.Values())

	visited := mapping.NewMultiMap[string, int]()
	foundKey, foundValue, ok := values.
		EachKey(func(key string, entries []int) { visited.Set(key, entries...) }).
		EachValue(func(key string, value int) { visited.Put(key, value*10) }).
		FirstValueWhere(func(_ string, value int) bool { return value > 4 })

	require.True(t, ok)
	require.Equal(t, "c", foundKey)
	require.Equal(t, 5, foundValue)
	require.Equal(t, []int{1, 2, 10, 20}, visited.Get("a"))
	require.True(t, values.AnyValueMatch(func(_ string, value int) bool { return value == 4 }))
	require.True(t, values.AllValuesMatch(func(_ string, value int) bool { return value > 0 }))
}

func TestConcurrentMultiMap_JSONCacheReturnsDefensiveCopy(t *testing.T) {
	t.Parallel()

	m := mapping.NewConcurrentMultiMap[string, int]()
	m.Put("a", 1)

	data, err := m.ToJSON()
	require.NoError(t, err)
	require.Equal(t, `{"a":[1]}`, string(data))
	require.Equal(t, `{"a":[1]}`, m.String())

	data[0] = '['
	fresh, err := m.ToJSON()
	require.NoError(t, err)
	require.Equal(t, `{"a":[1]}`, string(fresh))

	m.Put("a", 2)
	require.Equal(t, `{"a":[1,2]}`, m.String())
}
