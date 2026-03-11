package list

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConcurrentDeque_Basic(t *testing.T) {
	t.Parallel()

	d := NewConcurrentDeque[int]()
	d.PushBack(2, 3)
	d.PushFront(1)

	require.Equal(t, []int{1, 2, 3}, d.Values())

	value, ok := d.PopFront()
	require.True(t, ok)
	require.Equal(t, 1, value)

	value, ok = d.PopBack()
	require.True(t, ok)
	require.Equal(t, 3, value)

	require.Equal(t, []int{2}, d.Values())
}

func TestConcurrentDeque_ParallelPushBack(t *testing.T) {
	t.Parallel()

	d := NewConcurrentDeque[int]()

	const workers = 16
	const each = 120

	var wg sync.WaitGroup
	wg.Add(workers)

	for worker := 0; worker < workers; worker++ {
		worker := worker
		go func() {
			defer wg.Done()
			base := worker * each
			for i := 0; i < each; i++ {
				d.PushBack(base + i)
			}
		}()
	}

	wg.Wait()
	require.Equal(t, workers*each, d.Len())
}

func TestConcurrentDeque_SnapshotIsolation(t *testing.T) {
	t.Parallel()

	d := NewConcurrentDeque[int](1, 2)
	snapshot := d.Snapshot()

	d.PushBack(3)
	require.Equal(t, []int{1, 2}, snapshot.Values())
}
