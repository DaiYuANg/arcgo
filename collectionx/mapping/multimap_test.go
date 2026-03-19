package mapping

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMultiMap_BasicOps(t *testing.T) {
	t.Parallel()

	var m MultiMap[string, int]

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

	m := NewMultiMap[string, int]()
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

	m := NewMultiMapWithCapacity[string, int](8)
	m.PutAll("k", 1, 2)

	require.Equal(t, 1, m.Len())
	require.Equal(t, 2, m.ValueCount())
	require.Equal(t, []int{1, 2}, m.Get("k"))
}
