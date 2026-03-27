package mapping_test

import (
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
