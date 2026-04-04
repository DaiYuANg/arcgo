package list_test

import (
	"sync"
	"testing"

	list "github.com/DaiYuANg/arcgo/collectionx/list"
	"github.com/stretchr/testify/require"
)

func TestConcurrentList_ParallelAdd(t *testing.T) {
	t.Parallel()

	var l list.ConcurrentList[int]

	const workers = 24
	const each = 150

	var wg sync.WaitGroup
	wg.Add(workers)

	for worker := range workers {
		go func() {
			defer wg.Done()
			base := worker * each
			for i := range each {
				l.Add(base + i)
			}
		}()
	}

	wg.Wait()
	require.Equal(t, workers*each, l.Len())
}

func TestConcurrentList_InsertRemoveAndSnapshot(t *testing.T) {
	t.Parallel()

	l := list.NewConcurrentList(1, 3)
	require.True(t, l.AddAt(1, 2))
	require.Equal(t, []int{1, 2, 3}, l.Values())

	removed, ok := l.RemoveAt(1)
	require.True(t, ok)
	require.Equal(t, 2, removed)

	snapshot := l.Snapshot()
	l.Add(9)
	require.Equal(t, []int{1, 3}, snapshot.Values())
}

func TestConcurrentList_OptionAPIs(t *testing.T) {
	t.Parallel()

	var l list.ConcurrentList[string]
	l.Add("a", "b")

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

	removed := l.RemoveAtOption(1)
	require.True(t, removed.IsPresent())
	removedValue, ok := removed.Get()
	require.True(t, ok)
	require.Equal(t, "b", removedValue)

	require.True(t, l.GetOption(99).IsAbsent())
	require.True(t, new(list.ConcurrentList[string]).GetFirstOption().IsAbsent())
	require.True(t, new(list.ConcurrentList[string]).GetLastOption().IsAbsent())
}

func TestConcurrentList_Merge(t *testing.T) {
	t.Parallel()

	left := list.NewConcurrentList(1, 2)
	right := list.NewList(3, 4)
	otherConcurrent := list.NewConcurrentList(5, 6)

	left.Merge(right).MergeConcurrent(otherConcurrent).MergeSlice([]int{7, 8})
	require.Equal(t, []int{1, 2, 3, 4, 5, 6, 7, 8}, left.Values())
}

func TestNewConcurrentListWithCapacity(t *testing.T) {
	t.Parallel()

	l := list.NewConcurrentListWithCapacity[int](8, 1, 2, 3)

	require.Equal(t, []int{1, 2, 3}, l.Values())
	l.Add(4, 5, 6, 7, 8)
	require.Equal(t, []int{1, 2, 3, 4, 5, 6, 7, 8}, l.Values())
}

func TestConcurrentList_Join(t *testing.T) {
	t.Parallel()

	l := list.NewConcurrentList("a", "b", "c")
	require.Equal(t, "a,b,c", l.Join(","))
}
