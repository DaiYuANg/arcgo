package list

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConcurrentRingBuffer_Basic(t *testing.T) {
	t.Parallel()

	r := NewConcurrentRingBuffer[int](3)
	require.True(t, r.Push(1).IsAbsent())
	require.True(t, r.Push(2).IsAbsent())
	require.True(t, r.Push(3).IsAbsent())

	evicted := r.Push(4)
	require.True(t, evicted.IsPresent())
	value, ok := evicted.Get()
	require.True(t, ok)
	require.Equal(t, 1, value)
	require.Equal(t, []int{2, 3, 4}, r.Values())
}

func TestConcurrentRingBuffer_ParallelPush(t *testing.T) {
	t.Parallel()

	const workers = 12
	const each = 50
	r := NewConcurrentRingBuffer[int](workers * each)

	var wg sync.WaitGroup
	wg.Add(workers)

	for worker := 0; worker < workers; worker++ {
		worker := worker
		go func() {
			defer wg.Done()
			base := worker * each
			for i := 0; i < each; i++ {
				r.Push(base + i)
			}
		}()
	}

	wg.Wait()
	require.Equal(t, workers*each, r.Len())
}

func TestConcurrentRingBuffer_SnapshotIsolation(t *testing.T) {
	t.Parallel()

	r := NewConcurrentRingBuffer[string](2)
	r.Push("a")
	r.Push("b")
	snapshot := r.Snapshot()

	r.Push("c")
	require.Equal(t, []string{"a", "b"}, snapshot.Values())
}
