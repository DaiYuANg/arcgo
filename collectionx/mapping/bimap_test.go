package mapping_test

import (
	"testing"

	mapping "github.com/DaiYuANg/arcgo/collectionx/mapping"
	"github.com/stretchr/testify/require"
)

func TestBiMap_BasicAndReplace(t *testing.T) {
	t.Parallel()

	var m mapping.BiMap[string, int]
	m.Put("a", 1)
	m.Put("b", 2)

	value, ok := m.GetByKey("a")
	require.True(t, ok)
	require.Equal(t, 1, value)

	key, ok := m.GetByValue(2)
	require.True(t, ok)
	require.Equal(t, "b", key)

	// Replace by existing value: value=1 moves from key a to key c.
	m.Put("c", 1)
	_, ok = m.GetByKey("a")
	require.False(t, ok)
	key, ok = m.GetByValue(1)
	require.True(t, ok)
	require.Equal(t, "c", key)
}

func TestBiMap_OptionAndDelete(t *testing.T) {
	t.Parallel()

	m := mapping.NewBiMap[int, string]()
	m.Put(10, "x")

	opt := m.GetValueOption(10)
	require.True(t, opt.IsPresent())
	value, ok := opt.Get()
	require.True(t, ok)
	require.Equal(t, "x", value)

	require.True(t, m.DeleteByValue("x"))
	require.True(t, m.GetValueOption(10).IsAbsent())
}
