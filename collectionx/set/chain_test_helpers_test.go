package set_test

import (
	"strconv"
	"testing"

	set "github.com/DaiYuANg/arcgo/collectionx/set"
	"github.com/stretchr/testify/require"
)

func assertSetChainMethods(
	t *testing.T,
	values *set.Set[int],
	first int,
	ok bool,
	visited *set.Set[string],
	allEven bool,
	anyTwo bool,
) {
	t.Helper()
	require.ElementsMatch(t, []int{2, 4}, values.Values())
	require.True(t, ok)
	require.Contains(t, []int{3, 4}, first)
	require.ElementsMatch(t, []string{"1", "2", "3", "4"}, visited.Values())
	require.True(t, allEven)
	require.True(t, anyTwo)
}

func collectVisited(values *set.Set[int]) (*set.Set[string], int, bool) {
	visited := set.NewSet[string]()
	first, ok := values.
		Each(func(item int) { visited.Add(strconv.Itoa(item)) }).
		FirstWhere(func(item int) bool { return item > 2 }).Get()
	return visited, first, ok
}
