package list

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPriorityQueue_MinHeap(t *testing.T) {
	t.Parallel()

	pq := NewPriorityQueue(func(a int, b int) bool { return a < b }, 5, 1, 3, 2)
	require.Equal(t, []int{1, 2, 3, 5}, pq.ValuesSorted())

	v, ok := pq.Pop()
	require.True(t, ok)
	require.Equal(t, 1, v)
}

func TestPriorityQueue_MaxHeap(t *testing.T) {
	t.Parallel()

	pq := NewPriorityQueue(func(a int, b int) bool { return a > b })
	pq.Push(10)
	pq.Push(2)
	pq.Push(8)

	v, ok := pq.Peek()
	require.True(t, ok)
	require.Equal(t, 10, v)
}

func TestPriorityQueue_PanicOnNilComparator(t *testing.T) {
	t.Parallel()

	require.PanicsWithValue(t, "list: priority queue comparator cannot be nil", func() {
		NewPriorityQueue[int](nil)
	})
}
