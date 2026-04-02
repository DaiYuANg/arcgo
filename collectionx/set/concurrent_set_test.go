package set_test

import (
	"sync"
	"testing"

	set "github.com/DaiYuANg/arcgo/collectionx/set"
	"github.com/stretchr/testify/require"
)

func TestConcurrentSet_ParallelAdd(t *testing.T) {
	t.Parallel()

	var s set.ConcurrentSet[int]

	const workers = 24
	const each = 200

	var wg sync.WaitGroup
	wg.Add(workers)

	for worker := range workers {
		go func() {
			defer wg.Done()
			base := worker * each
			for i := range each {
				s.Add(base + i)
			}
		}()
	}

	wg.Wait()

	require.Equal(t, workers*each, s.Len())
	require.True(t, s.Contains(0))
	require.True(t, s.Contains(workers*each-1))
}

func TestConcurrentSet_SnapshotIsIndependent(t *testing.T) {
	t.Parallel()

	s := set.NewConcurrentSet(1, 2, 3)
	snap := s.Snapshot()

	require.True(t, snap.Contains(1))

	s.Add(9)
	require.False(t, snap.Contains(9))
}

func TestConcurrentSet_Merge(t *testing.T) {
	t.Parallel()

	left := set.NewConcurrentSet(1, 2)
	right := set.NewSet(2, 3)
	otherConcurrent := set.NewConcurrentSet(4, 5)

	left.Merge(right).MergeConcurrent(otherConcurrent).MergeSlice([]int{5, 6})
	require.ElementsMatch(t, []int{1, 2, 3, 4, 5, 6}, left.Values())
}

func TestNewConcurrentSetWithCapacity(t *testing.T) {
	t.Parallel()

	s := set.NewConcurrentSetWithCapacity(8, 1, 2, 2, 3)

	require.Equal(t, 3, s.Len())
	require.True(t, s.Contains(1))
	require.True(t, s.Contains(2))
	require.True(t, s.Contains(3))
}

func TestConcurrentSet_ChainMethods(t *testing.T) {
	t.Parallel()

	values := set.NewConcurrentSet(1, 2, 3, 4).
		Where(func(item int) bool { return item >= 2 }).
		Reject(func(item int) bool { return item == 3 })
	visited, first, ok := collectVisited(set.NewConcurrentSet(1, 2, 3, 4).Snapshot())
	assertSetChainMethods(
		t,
		values,
		first,
		ok,
		visited,
		set.NewConcurrentSet(2, 4, 6).AllMatch(func(item int) bool { return item%2 == 0 }),
		set.NewConcurrentSet(1, 2, 3).AnyMatch(func(item int) bool { return item == 2 }),
	)
}
