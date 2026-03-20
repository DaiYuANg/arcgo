package list

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRopeList_Basic(t *testing.T) {
	t.Parallel()

	r := NewRopeList[int](1, 2, 3)
	require.Equal(t, 3, r.Len())
	v, ok := r.Get(1)
	require.True(t, ok)
	require.Equal(t, 2, v)

	require.True(t, r.AddAt(1, 99))
	require.Equal(t, 4, r.Len())
	require.Equal(t, []int{1, 99, 2, 3}, r.Values())

	removed, ok := r.RemoveAt(1)
	require.True(t, ok)
	require.Equal(t, 99, removed)
	require.Equal(t, []int{1, 2, 3}, r.Values())
}

func TestRopeList_InsertAt(t *testing.T) {
	t.Parallel()

	r := NewRopeList[int]()
	r.Add(1, 2, 3)
	require.True(t, r.InsertAt(0, 0))
	require.Equal(t, []int{0, 1, 2, 3}, r.Values())
	require.True(t, r.InsertAt(4, 4))
	require.Equal(t, []int{0, 1, 2, 3, 4}, r.Values())
	require.True(t, r.InsertAt(2, 99, 98))
	require.Equal(t, []int{0, 1, 99, 98, 2, 3, 4}, r.Values())
}

func TestRopeList_RemoveIf(t *testing.T) {
	t.Parallel()

	r := NewRopeList[int](1, 2, 3, 4, 5)
	n := r.RemoveIf(func(x int) bool { return x%2 == 0 })
	require.Equal(t, 2, n)
	require.Equal(t, []int{1, 3, 5}, r.Values())
}

func TestRopeList_Clone(t *testing.T) {
	t.Parallel()

	r := NewRopeList[int](1, 2, 3)
	c := r.Clone()
	require.Equal(t, r.Values(), c.Values())
	c.Add(4)
	require.Equal(t, 3, r.Len())
	require.Equal(t, 4, c.Len())
}
