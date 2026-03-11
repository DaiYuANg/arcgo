package list

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPriorityQueue_MinHeap(t *testing.T) {
	t.Parallel()

	pq, err := NewPriorityQueue(func(a int, b int) bool { return a < b }, 5, 1, 3, 2)
	require.NoError(t, err)
	require.Equal(t, []int{1, 2, 3, 5}, pq.ValuesSorted())

	v, ok := pq.Pop()
	require.True(t, ok)
	require.Equal(t, 1, v)
}

func TestPriorityQueue_MaxHeap(t *testing.T) {
	t.Parallel()

	pq, err := NewPriorityQueue(func(a int, b int) bool { return a > b })
	require.NoError(t, err)
	pq.Push(10)
	pq.Push(2)
	pq.Push(8)

	v, ok := pq.Peek()
	require.True(t, ok)
	require.Equal(t, 10, v)
}

func TestPriorityQueue_ErrorOnNilComparator(t *testing.T) {
	t.Parallel()

	pq, err := NewPriorityQueue[int](nil)
	require.Nil(t, pq)
	require.ErrorIs(t, err, ErrNilPriorityQueueComparator)
}
