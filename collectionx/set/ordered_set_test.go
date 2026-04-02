package set_test

import (
	"testing"

	set "github.com/DaiYuANg/arcgo/collectionx/set"
	"github.com/stretchr/testify/require"
)

func TestOrderedSet_OrderAndDedupe(t *testing.T) {
	t.Parallel()

	var s set.OrderedSet[int]
	s.Add(1, 2, 2, 3, 1)

	require.Equal(t, []int{1, 2, 3}, s.Values())
	require.True(t, s.Contains(2))
}

func TestOrderedSet_RemoveReindex(t *testing.T) {
	t.Parallel()

	s := set.NewOrderedSet("a", "b", "c")
	require.True(t, s.Remove("b"))
	require.Equal(t, []string{"a", "c"}, s.Values())

	item, ok := s.At(1)
	require.True(t, ok)
	require.Equal(t, "c", item)
}

func TestNewOrderedSetWithCapacity(t *testing.T) {
	t.Parallel()

	s := set.NewOrderedSetWithCapacity(8, "a", "b", "a", "c")

	require.Equal(t, []string{"a", "b", "c"}, s.Values())
}

func TestOrderedSet_ChainMethods(t *testing.T) {
	t.Parallel()

	values := set.NewOrderedSet(1, 2, 3, 4).
		Where(func(item int) bool { return item >= 2 }).
		Reject(func(item int) bool { return item == 3 }).
		Take(2)
	require.Equal(t, []int{2, 4}, values.Values())

	dropped := set.NewOrderedSet(1, 2, 3, 4).Drop(2)
	require.Equal(t, []int{3, 4}, dropped.Values())

	visited := make([]int, 0, 4)
	first, ok := set.NewOrderedSet(1, 2, 3, 4).
		Each(func(item int) { visited = append(visited, item) }).
		FirstWhere(func(item int) bool { return item > 2 }).Get()
	require.True(t, ok)
	require.Equal(t, 3, first)
	require.Equal(t, []int{1, 2, 3, 4}, visited)

	require.True(t, set.NewOrderedSet(2, 4, 6).AllMatch(func(item int) bool { return item%2 == 0 }))
	require.True(t, set.NewOrderedSet(1, 2, 3).AnyMatch(func(item int) bool { return item == 2 }))
}
