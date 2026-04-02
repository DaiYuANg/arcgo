package set_test

import (
	"strconv"
	"testing"

	set "github.com/DaiYuANg/arcgo/collectionx/set"
	"github.com/stretchr/testify/require"
)

func TestSet_ZeroValueAndBasicOps(t *testing.T) {
	t.Parallel()

	var s set.Set[int]

	s.Add(1, 2, 2, 3)

	require.Equal(t, 3, s.Len())
	require.True(t, s.Contains(1))
	require.False(t, s.Contains(9))

	require.True(t, s.Remove(2))
	require.False(t, s.Remove(2))
	require.Equal(t, 2, s.Len())

	s.Clear()
	require.True(t, s.IsEmpty())
}

func TestSet_MathOperations(t *testing.T) {
	t.Parallel()

	left := set.NewSet(1, 2, 3)
	right := set.NewSet(3, 4, 5)

	require.ElementsMatch(t, []int{1, 2, 3, 4, 5}, left.Union(right).Values())
	require.ElementsMatch(t, []int{3}, left.Intersect(right).Values())
	require.ElementsMatch(t, []int{1, 2}, left.Difference(right).Values())
}

func TestSet_RangeStop(t *testing.T) {
	t.Parallel()

	s := set.NewSet("a", "b", "c")
	visited := 0

	s.Range(func(item string) bool {
		visited++
		return item != ""
	})

	require.Equal(t, 3, visited)

	visited = 0
	s.Range(func(_ string) bool {
		visited++
		return false
	})
	require.Equal(t, 1, visited)
}

func TestSet_Merge(t *testing.T) {
	t.Parallel()

	left := set.NewSet(1, 2)
	right := set.NewSet(2, 3)

	left.Merge(right).MergeSlice([]int{3, 4, 5})
	require.ElementsMatch(t, []int{1, 2, 3, 4, 5}, left.Values())
}

func TestNewSetWithCapacity(t *testing.T) {
	t.Parallel()

	s := set.NewSetWithCapacity(8, 1, 2, 2, 3)

	require.Equal(t, 3, s.Len())
	require.True(t, s.Contains(1))
	require.True(t, s.Contains(2))
	require.True(t, s.Contains(3))
}

func TestSet_ChainMethods(t *testing.T) {
	t.Parallel()

	values := set.NewSet(1, 2, 3, 4).
		Where(func(item int) bool { return item >= 2 }).
		Reject(func(item int) bool { return item == 3 })
	require.ElementsMatch(t, []int{2, 4}, values.Values())

	visited := set.NewSet[string]()
	first, ok := set.NewSet(1, 2, 3, 4).
		Each(func(item int) { visited.Add(strconv.Itoa(item)) }).
		FirstWhere(func(item int) bool { return item > 2 }).Get()
	require.True(t, ok)
	require.Contains(t, []int{3, 4}, first)
	require.ElementsMatch(t, []string{"1", "2", "3", "4"}, visited.Values())

	require.True(t, set.NewSet(2, 4, 6).AllMatch(func(item int) bool { return item%2 == 0 }))
	require.True(t, set.NewSet(1, 2, 3).AnyMatch(func(item int) bool { return item == 2 }))
}
