package list_test

import (
	"strconv"
	"testing"

	list "github.com/DaiYuANg/arcgo/collectionx/list"
	"github.com/stretchr/testify/require"
)

func TestList_ZeroValueAndBasicOps(t *testing.T) {
	t.Parallel()

	var l list.List[string]

	l.Add("a", "b")
	require.Equal(t, 2, l.Len())

	value, ok := l.Get(1)
	require.True(t, ok)
	require.Equal(t, "b", value)

	require.True(t, l.Set(1, "B"))
	value, ok = l.Get(1)
	require.True(t, ok)
	require.Equal(t, "B", value)

	removed, ok := l.RemoveAt(0)
	require.True(t, ok)
	require.Equal(t, "a", removed)
	require.Equal(t, 1, l.Len())
}

func TestList_AddAllAt(t *testing.T) {
	t.Parallel()

	l := list.NewList(1, 4)
	require.True(t, l.AddAllAt(1, 2, 3))
	require.Equal(t, []int{1, 2, 3, 4}, l.Values())

	require.True(t, l.AddAt(4, 5))
	require.Equal(t, []int{1, 2, 3, 4, 5}, l.Values())
	require.False(t, l.AddAt(6, 6))
}

func TestList_RemoveIfAndCopySemantics(t *testing.T) {
	t.Parallel()

	l := list.NewList(1, 2, 3, 4, 5, 6)
	removed := l.RemoveIf(func(item int) bool {
		return item%2 == 0
	})

	require.Equal(t, 3, removed)
	require.Equal(t, []int{1, 3, 5}, l.Values())

	values := l.Values()
	values[0] = 99
	require.Equal(t, []int{1, 3, 5}, l.Values())
}

func TestList_OptionAPIs(t *testing.T) {
	t.Parallel()

	l := list.NewList("a", "b")

	value, ok := l.GetFirst()
	require.True(t, ok)
	require.Equal(t, "a", value)

	opt := l.GetFirstOption()
	require.True(t, opt.IsPresent())
	value, ok = opt.Get()
	require.True(t, ok)
	require.Equal(t, "a", value)

	last, ok := l.GetLast()
	require.True(t, ok)
	require.Equal(t, "b", last)

	lastOpt := l.GetLastOption()
	require.True(t, lastOpt.IsPresent())
	last, ok = lastOpt.Get()
	require.True(t, ok)
	require.Equal(t, "b", last)

	removedOpt := l.RemoveAtOption(1)
	require.True(t, removedOpt.IsPresent())
	removedValue, ok := removedOpt.Get()
	require.True(t, ok)
	require.Equal(t, "b", removedValue)

	require.True(t, l.GetOption(10).IsAbsent())
	require.True(t, l.RemoveAtOption(10).IsAbsent())
	require.True(t, list.NewList[string]().GetFirstOption().IsAbsent())
	require.True(t, list.NewList[string]().GetLastOption().IsAbsent())
}

func TestList_Merge(t *testing.T) {
	t.Parallel()

	left := list.NewList(1, 2)
	right := list.NewList(3, 4)

	left.Merge(right).MergeSlice([]int{5, 6})
	require.Equal(t, []int{1, 2, 3, 4, 5, 6}, left.Values())
}

func TestNewListWithCapacity(t *testing.T) {
	t.Parallel()

	l := list.NewListWithCapacity[int](8, 1, 2, 3)

	require.Equal(t, []int{1, 2, 3}, l.Values())
	l.Add(4, 5, 6, 7, 8)
	require.Equal(t, []int{1, 2, 3, 4, 5, 6, 7, 8}, l.Values())
}

func TestList_Sort(t *testing.T) {
	t.Parallel()

	l := list.NewList(3, 1, 2)

	sorted := l.Sort(func(left, right int) int {
		switch {
		case left < right:
			return -1
		case left > right:
			return 1
		default:
			return 0
		}
	})

	require.Same(t, l, sorted)
	require.Equal(t, []int{1, 2, 3}, l.Values())
}

func TestList_Join(t *testing.T) {
	t.Parallel()

	require.Equal(t, "a,b,c", list.NewList("a", "b", "c").Join(","))
	require.Equal(t, "1,2,3", list.NewList(1, 2, 3).Join(","))
	require.Equal(t, "u1 | u2", list.NewList(1, 2).Join(" | ", func(_ int, item int) string {
		return "u" + strconv.Itoa(item)
	}))
	require.Equal(t, "", list.NewList[string]().Join(","))
}
