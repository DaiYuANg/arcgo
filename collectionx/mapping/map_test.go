package mapping_test

import (
	"strconv"
	"testing"

	mapping "github.com/DaiYuANg/arcgo/collectionx/mapping"
	"github.com/stretchr/testify/require"
)

func TestMap_ZeroValueAndClone(t *testing.T) {
	t.Parallel()

	var m mapping.Map[string, int]
	m.Set("a", 1)
	m.Set("b", 2)

	value, ok := m.Get("a")
	require.True(t, ok)
	require.Equal(t, 1, value)
	require.Equal(t, 2, m.Len())

	clone := m.Clone()
	clone.Set("a", 9)

	originalValue, ok := m.Get("a")
	require.True(t, ok)
	require.Equal(t, 1, originalValue)
}

func TestMap_AllReturnsCopy(t *testing.T) {
	t.Parallel()

	m := mapping.NewMapFrom(map[string]int{
		"a": 1,
		"b": 2,
	})

	all := m.All()
	all["a"] = 99

	value, ok := m.Get("a")
	require.True(t, ok)
	require.Equal(t, 1, value)
}

func TestMap_GetOption(t *testing.T) {
	t.Parallel()

	m := mapping.NewMapFrom(map[string]int{
		"a": 1,
	})

	opt := m.GetOption("a")
	require.True(t, opt.IsPresent())
	value, ok := opt.Get()
	require.True(t, ok)
	require.Equal(t, 1, value)

	require.True(t, m.GetOption("missing").IsAbsent())
}

func TestMap_RangeStop(t *testing.T) {
	t.Parallel()

	m := mapping.NewMapFrom(map[int]int{
		1: 10,
		2: 20,
		3: 30,
	})

	visited := 0
	m.Range(func(_ int, _ int) bool {
		visited++
		return false
	})
	require.Equal(t, 1, visited)
}

func TestNewMapWithCapacity(t *testing.T) {
	t.Parallel()

	m := mapping.NewMapWithCapacity[string, int](8)
	m.Set("a", 1)
	m.Set("b", 2)

	require.Equal(t, 2, m.Len())
	require.Equal(t, 1, m.GetOrDefault("a", 0))
}

func TestMap_ChainMethods(t *testing.T) {
	t.Parallel()

	values := mapping.NewMapFrom(map[string]int{
		"a": 1,
		"b": 2,
		"c": 3,
	}).
		WhereEntries(func(key string, value int) bool { return value >= 2 }).
		RejectEntries(func(key string, _ int) bool { return key == "c" })
	require.Equal(t, map[string]int{"b": 2}, values.All())

	visited := mapping.NewMap[string, string]()
	key, value, ok := mapping.NewMapFrom(map[string]int{
		"a": 1,
		"b": 2,
		"c": 3,
	}).EachEntry(func(key string, value int) {
		visited.Set(key, strconv.Itoa(value))
	}).FirstEntryWhere(func(_ string, value int) bool { return value > 1 })
	require.True(t, ok)
	require.Contains(t, []string{"b", "c"}, key)
	require.Contains(t, []int{2, 3}, value)
	require.Equal(t, 3, visited.Len())

	require.True(t, mapping.NewMapFrom(map[string]int{"a": 2, "b": 4}).AllEntryMatch(func(_ string, value int) bool { return value%2 == 0 }))
	require.True(t, mapping.NewMapFrom(map[string]int{"a": 1, "b": 2}).AnyEntryMatch(func(key string, _ int) bool { return key == "b" }))
}

func TestMap_JSONCacheReturnsDefensiveCopy(t *testing.T) {
	t.Parallel()

	m := mapping.NewMap[string, int]()
	m.Set("a", 1)

	data, err := m.ToJSON()
	require.NoError(t, err)
	require.Equal(t, `{"a":1}`, string(data))
	require.Equal(t, `{"a":1}`, m.String())

	data[0] = '['

	fresh, err := m.ToJSON()
	require.NoError(t, err)
	require.Equal(t, `{"a":1}`, string(fresh))
	require.Equal(t, `{"a":1}`, m.String())

	m.Set("b", 2)
	require.Contains(t, m.String(), `"b":2`)
}
