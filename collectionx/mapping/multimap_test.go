package mapping_test

import (
	"testing"

	mapping "github.com/DaiYuANg/arcgo/collectionx/mapping"
	"github.com/stretchr/testify/require"
)

func TestMultiMap_BasicOps(t *testing.T) {
	t.Parallel()

	var m mapping.MultiMap[string, int]

	m.Put("a", 1)
	m.PutAll("a", 2, 3)
	m.Put("b", 10)

	require.Equal(t, 2, m.Len())
	require.Equal(t, 4, m.ValueCount())
	require.Equal(t, []int{1, 2, 3}, m.Get("a"))

	m.Set("a", 9)
	require.Equal(t, []int{9}, m.Get("a"))

	removed := m.DeleteValueIf("a", func(value int) bool { return value == 9 })
	require.Equal(t, 1, removed)
	require.False(t, m.ContainsKey("a"))
}

func TestMultiMap_ViewStaysStableAfterWriteAndCopy(t *testing.T) {
	t.Parallel()

	m := mapping.NewMultiMap[string, int]()
	m.PutAll("k", 1, 2)

	view := m.Get("k")
	m.Put("k", 3)
	require.Equal(t, []int{1, 2}, view)
	require.Equal(t, []int{1, 2, 3}, m.Get("k"))

	copyValues := m.GetCopy("k")
	copyValues[0] = 99
	require.Equal(t, []int{1, 2, 3}, m.Get("k"))

	opt := m.GetOption("k")
	require.True(t, opt.IsPresent())
	got, ok := opt.Get()
	require.True(t, ok)
	require.Equal(t, []int{1, 2, 3}, got)

	require.True(t, m.GetOption("missing").IsAbsent())
}

func TestNewMultiMapWithCapacity(t *testing.T) {
	t.Parallel()

	m := mapping.NewMultiMapWithCapacity[string, int](8)
	m.PutAll("k", 1, 2)

	require.Equal(t, 1, m.Len())
	require.Equal(t, 2, m.ValueCount())
	require.Equal(t, []int{1, 2}, m.Get("k"))
}

func TestMultiMap_CloneAndFromAll(t *testing.T) {
	t.Parallel()

	source := mapping.NewMultiMapFromAll(map[string][]int{
		"a": {1, 2},
		"b": {3},
	})
	cloned := source.Clone()

	cloned.Put("a", 4)
	cloned.Delete("b")

	require.Equal(t, []int{1, 2}, source.Get("a"))
	require.Equal(t, []int{1, 2, 4}, cloned.Get("a"))
	require.True(t, source.ContainsKey("b"))
	require.False(t, cloned.ContainsKey("b"))
	require.Equal(t, 2, source.Len())
	require.Equal(t, 3, source.ValueCount())
	require.Equal(t, 3, cloned.ValueCount())
}

func TestMultiMap_FluentOps(t *testing.T) {
	t.Parallel()

	values := mapping.NewMultiMap[string, int]()
	values.PutAll("a", 1, 2)
	values.PutAll("b", 3, 4)
	values.Put("c", 5)

	filtered := values.
		WhereKeys(func(key string, _ []int) bool { return key != "c" }).
		RejectValues(func(_ string, value int) bool { return value%2 == 0 })

	require.Equal(t, []int{1}, filtered.Get("a"))
	require.Equal(t, []int{3}, filtered.Get("b"))
	require.False(t, filtered.ContainsKey("c"))

	flattened := filtered.FlattenValues()
	require.ElementsMatch(t, []int{1, 3}, flattened.Values())

	visited := mapping.NewMultiMap[string, int]()
	foundKey, foundValue, ok := values.
		EachKey(func(key string, entries []int) { visited.Set(key, entries...) }).
		EachValue(func(key string, value int) { visited.Put(key, value*10) }).
		FirstValueWhere(func(_ string, value int) bool { return value > 3 })

	require.True(t, ok)
	require.True(t, foundKey == "b" || foundKey == "c")
	require.True(t, foundValue > 3)
	require.Equal(t, []int{1, 2, 10, 20}, visited.Get("a"))
	require.True(t, values.AnyValueMatch(func(_ string, value int) bool { return value == 5 }))
	require.True(t, values.AllValuesMatch(func(_ string, value int) bool { return value > 0 }))
	require.False(t, values.AllValuesMatch(func(_ string, value int) bool { return value%2 == 0 }))
}
