package list_test

import (
	"sync"
	"testing"

	list "github.com/DaiYuANg/arcgo/collectionx/list"
	"github.com/stretchr/testify/require"
)

func TestConcurrentGrid_ParallelAddRow(t *testing.T) {
	t.Parallel()

	var g list.ConcurrentGrid[int]

	const workers = 16
	const each = 40

	var wg sync.WaitGroup
	wg.Add(workers)

	for worker := range workers {
		go func() {
			defer wg.Done()
			base := worker * each
			for i := range each {
				g.AddRow(base+i, base+i+1)
			}
		}()
	}

	wg.Wait()
	require.Equal(t, workers*each, g.RowCount())
	require.Equal(t, workers*each*2, g.Len())
}

func TestConcurrentGrid_RowMutationAndSnapshot(t *testing.T) {
	t.Parallel()

	g := list.NewConcurrentGrid([]int{1, 2}, []int{3})
	require.True(t, g.Set(0, 1, 9))

	row, ok := g.GetRow(0)
	require.True(t, ok)
	require.Equal(t, []int{1, 9}, row)

	snapshot := g.Snapshot()
	g.AddRow(4, 5)
	require.Equal(t, 2, snapshot.RowCount())

	removed, ok := g.RemoveRow(1)
	require.True(t, ok)
	require.Equal(t, []int{3}, removed)
}

func TestConcurrentGrid_MergeAndFluentOps(t *testing.T) {
	t.Parallel()

	left := list.NewConcurrentGrid([]int{1}, []int{2, 3})
	right := list.NewGrid([]int{4, 5})
	otherConcurrent := list.NewConcurrentGrid([]int{6})

	left.Merge(right).MergeConcurrent(otherConcurrent)
	require.Equal(t, 4, left.RowCount())
	require.Equal(t, 6, left.Len())

	filtered := left.
		RejectRows(func(index int, _ []int) bool { return index == 0 }).
		RejectCells(func(_ int, _ int, value int) bool { return value%2 == 0 })

	require.Equal(t, 2, filtered.RowCount())
	require.Equal(t, []int{3}, filtered.Values()[0])
	require.Equal(t, []int{5}, filtered.Values()[1])

	visited := list.NewGrid[int]()
	firstRow, ok := left.
		EachRow(func(_ int, row []int) {
			visited.AddRow(row...)
		}).
		FirstRowWhere(func(_ int, row []int) bool { return len(row) == 2 }).
		Get()
	require.True(t, ok)
	require.Equal(t, []int{2, 3}, firstRow)
	require.Equal(t, left.RowCount(), visited.RowCount())

	rowIndex, columnIndex, value, ok := left.FirstCellWhere(func(_ int, _ int, value int) bool { return value == 5 })
	require.True(t, ok)
	require.Equal(t, 2, rowIndex)
	require.Equal(t, 1, columnIndex)
	require.Equal(t, 5, value)
	require.True(t, left.AnyCellMatch(func(_ int, _ int, value int) bool { return value == 6 }))
	require.True(t, left.AllCellsMatch(func(_ int, _ int, value int) bool { return value > 0 }))
}

func TestNewConcurrentGridWithCapacity(t *testing.T) {
	t.Parallel()

	g := list.NewConcurrentGridWithCapacity[int](4, []int{1}, []int{2, 3})
	require.Equal(t, [][]int{{1}, {2, 3}}, g.Values())
}
