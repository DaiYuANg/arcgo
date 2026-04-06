package list_test

import (
	"testing"

	list "github.com/DaiYuANg/arcgo/collectionx/list"
	"github.com/stretchr/testify/require"
)

func TestGridAddValuesAndClone(t *testing.T) {
	t.Parallel()

	g := list.NewGrid[int]([]int{1, 2}, []int{3})
	g.AddRow(4, 5)

	require.Equal(t, 3, g.RowCount())
	require.Equal(t, 5, g.Len())

	firstRow, ok := g.GetRow(0)
	require.True(t, ok)
	require.Equal(t, []int{1, 2}, firstRow)

	firstRow[0] = 99
	values := g.Values()
	values[1][0] = 88

	assertGridCell(t, g, 0, 0, 1)
	assertGridCell(t, g, 1, 0, 3)

	cloned := g.Clone()
	require.True(t, cloned.Set(0, 0, 7))
	require.True(t, cloned.SetRow(1, 8, 9))

	assertGridCell(t, g, 0, 0, 1)
	assertGridRow(t, g, 1, []int{3})
}

func TestGridRemoveAndMerge(t *testing.T) {
	t.Parallel()

	g := list.NewGrid[int]()
	g.AddRow(1)
	g.AddRows([]int{2, 3}, []int{})

	removed, ok := g.RemoveRow(1)
	require.True(t, ok)
	require.Equal(t, []int{2, 3}, removed)

	other := list.NewGrid[int]([]int{4, 5})
	g.Merge(other)

	require.Equal(t, 3, g.RowCount())
	require.Equal(t, 3, g.Len())

	otherValues := other.Values()
	otherValues[0][0] = 99
	assertGridCell(t, g, 2, 0, 4)
}

func TestGridListRowHelpers(t *testing.T) {
	t.Parallel()

	g := list.NewGrid[int]()
	g.AddRowList(list.NewList(1, 2))
	g.AddRowsList(list.NewList(list.NewList(3), list.NewList(4, 5)))

	require.Equal(t, 3, g.RowCount())

	row, ok := g.GetRowList(0)
	require.True(t, ok)
	require.Equal(t, 2, row.Len())

	require.True(t, row.Set(0, 99))
	assertGridCell(t, g, 0, 0, 1)
}

func TestGridRowSelectors(t *testing.T) {
	t.Parallel()

	g := list.NewGrid[int]([]int{1}, []int{2, 3}, []int{4, 5, 6})

	filtered := g.WhereRows(func(_ int, row []int) bool { return len(row) >= 2 })
	require.Equal(t, 2, filtered.RowCount())
	assertGridRow(t, filtered, 0, []int{2, 3})

	rejected := g.RejectRows(func(index int, _ []int) bool { return index == 0 })
	require.Equal(t, 2, rejected.RowCount())

	taken := g.TakeRows(2)
	require.Equal(t, 2, taken.RowCount())

	dropped := g.DropRows(1)
	require.Equal(t, 2, dropped.RowCount())
	assertGridRow(t, dropped, 0, []int{2, 3})
}

func TestGridEachRowAndFirstRowWhere(t *testing.T) {
	t.Parallel()

	g := list.NewGrid[int]([]int{1}, []int{2, 3}, []int{4, 5, 6})

	rowCount := 0
	cellCount := 0
	returned := g.EachRow(func(_ int, row []int) {
		rowCount++
		cellCount += len(row)
		row[0] = 99
	})

	require.Same(t, g, returned)
	require.Equal(t, 3, rowCount)
	require.Equal(t, 6, cellCount)
	assertGridCell(t, g, 0, 0, 1)

	first, ok := g.FirstRowWhere(func(_ int, row []int) bool { return len(row) == 3 }).Get()
	require.True(t, ok)
	require.Equal(t, []int{4, 5, 6}, first)

	first[0] = 77
	assertGridCell(t, g, 2, 0, 4)
}

func TestGridRowPredicates(t *testing.T) {
	t.Parallel()

	g := list.NewGrid[int]([]int{1}, []int{2, 3}, []int{4, 5, 6})

	require.True(t, g.AnyRowMatch(func(_ int, row []int) bool { return len(row) == 1 }))
	require.False(t, g.AnyRowMatch(func(_ int, row []int) bool { return len(row) == 4 }))
	require.True(t, g.AllRowsMatch(func(_ int, row []int) bool { return len(row) >= 1 }))
	require.False(t, g.AllRowsMatch(func(_ int, row []int) bool { return len(row) >= 2 }))
}

func TestGridCellSelectors(t *testing.T) {
	t.Parallel()

	g := list.NewGrid[int]([]int{1, 2}, []int{3, 4, 5}, []int{6})

	filtered := g.WhereCells(func(_ int, _ int, value int) bool { return value%2 == 0 })
	require.Equal(t, 3, filtered.RowCount())
	assertGridRow(t, filtered, 1, []int{4})

	rejected := g.RejectCells(func(rowIndex int, _ int, _ int) bool { return rowIndex == 0 })
	require.Equal(t, 2, rejected.RowCount())
	assertGridRow(t, rejected, 0, []int{3, 4, 5})
}

func TestGridEachCellAndFirstCellWhere(t *testing.T) {
	t.Parallel()

	g := list.NewGrid[int]([]int{1, 2}, []int{3, 4, 5}, []int{6})

	cellCount := 0
	sum := 0
	returned := g.EachCell(func(rowIndex int, columnIndex int, value int) {
		cellCount++
		sum += value + rowIndex + columnIndex
	})

	require.Same(t, g, returned)
	require.Equal(t, 6, cellCount)
	require.Equal(t, 30, sum)

	rowIndex, columnIndex, value, ok := g.FirstCellWhere(func(_ int, _ int, current int) bool { return current == 4 })
	require.True(t, ok)
	require.Equal(t, 1, rowIndex)
	require.Equal(t, 1, columnIndex)
	require.Equal(t, 4, value)
}

func TestGridCellPredicates(t *testing.T) {
	t.Parallel()

	g := list.NewGrid[int]([]int{1, 2}, []int{3, 4, 5}, []int{6})

	require.True(t, g.AnyCellMatch(func(_ int, _ int, value int) bool { return value == 6 }))
	require.False(t, g.AnyCellMatch(func(_ int, _ int, value int) bool { return value == 7 }))
	require.True(t, g.AllCellsMatch(func(_ int, _ int, value int) bool { return value >= 1 }))
	require.False(t, g.AllCellsMatch(func(_ int, _ int, value int) bool { return value%2 == 0 }))
}

func assertGridCell[T comparable](t *testing.T, g *list.Grid[T], rowIndex, columnIndex int, expected T) {
	t.Helper()

	value, ok := g.Get(rowIndex, columnIndex)
	require.True(t, ok)
	require.Equal(t, expected, value)
}

func assertGridRow[T comparable](t *testing.T, g *list.Grid[T], rowIndex int, expected []T) {
	t.Helper()

	row, ok := g.GetRow(rowIndex)
	require.True(t, ok)
	require.Equal(t, expected, row)
}
