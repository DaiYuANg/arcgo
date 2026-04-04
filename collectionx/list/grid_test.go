package list

import "testing"

func TestGridAddValuesAndClone(t *testing.T) {
	g := NewGrid[int]([]int{1, 2}, []int{3})
	g.AddRow(4, 5)

	if g.RowCount() != 3 {
		t.Fatalf("expected row count 3, got %d", g.RowCount())
	}
	if g.Len() != 5 {
		t.Fatalf("expected cell count 5, got %d", g.Len())
	}

	row, ok := g.GetRow(0)
	if !ok || len(row) != 2 || row[0] != 1 || row[1] != 2 {
		t.Fatalf("unexpected first row: %#v, ok=%v", row, ok)
	}
	row[0] = 99

	values := g.Values()
	values[1][0] = 88
	if value, ok := g.Get(0, 0); !ok || value != 1 {
		t.Fatalf("unexpected cell after row mutation: %v, ok=%v", value, ok)
	}
	if value, ok := g.Get(1, 0); !ok || value != 3 {
		t.Fatalf("unexpected cell after values mutation: %v, ok=%v", value, ok)
	}

	cloned := g.Clone()
	_ = cloned.Set(0, 0, 7)
	_ = cloned.SetRow(1, 8, 9)
	if value, _ := g.Get(0, 0); value != 1 {
		t.Fatalf("unexpected original value after clone mutation: %v", value)
	}
	if row, _ := g.GetRow(1); len(row) != 1 || row[0] != 3 {
		t.Fatalf("unexpected original row after clone mutation: %#v", row)
	}
}

func TestGridRemoveAndMerge(t *testing.T) {
	g := NewGrid[int]()
	g.AddRow(1)
	g.AddRows([]int{2, 3}, []int{})

	removed, ok := g.RemoveRow(1)
	if !ok || len(removed) != 2 || removed[0] != 2 || removed[1] != 3 {
		t.Fatalf("unexpected removed row: %#v, ok=%v", removed, ok)
	}

	other := NewGrid[int]([]int{4, 5})
	g.Merge(other)
	if g.RowCount() != 3 {
		t.Fatalf("expected row count 3 after merge, got %d", g.RowCount())
	}
	if g.Len() != 3 {
		t.Fatalf("expected cell count 3 after merge, got %d", g.Len())
	}

	otherValues := other.Values()
	otherValues[0][0] = 99
	value, _ := g.Get(2, 0)
	if value != 4 {
		t.Fatalf("unexpected merged cell value: %v", value)
	}
}

func TestGridListRowHelpers(t *testing.T) {
	g := NewGrid[int]()
	g.AddRowList(NewList(1, 2))
	g.AddRowsList(NewList(NewList(3), NewList(4, 5)))

	if g.RowCount() != 3 {
		t.Fatalf("expected row count 3, got %d", g.RowCount())
	}

	row, ok := g.GetRowList(0)
	if !ok || row.Len() != 2 {
		t.Fatalf("unexpected row list: %#v ok=%v", row, ok)
	}
	_ = row.Set(0, 99)
	value, _ := g.Get(0, 0)
	if value != 1 {
		t.Fatalf("expected copied row list, got %v", value)
	}
}

func TestGridRowFluentHelpers(t *testing.T) {
	g := NewGrid[int]([]int{1}, []int{2, 3}, []int{4, 5, 6})

	filtered := g.WhereRows(func(_ int, row []int) bool {
		return len(row) >= 2
	})
	if filtered.RowCount() != 2 {
		t.Fatalf("expected 2 filtered rows, got %d", filtered.RowCount())
	}
	if row, _ := filtered.GetRow(0); len(row) != 2 || row[0] != 2 || row[1] != 3 {
		t.Fatalf("unexpected filtered first row: %#v", row)
	}

	rejected := g.RejectRows(func(index int, _ []int) bool {
		return index == 0
	})
	if rejected.RowCount() != 2 {
		t.Fatalf("expected 2 rejected rows, got %d", rejected.RowCount())
	}

	taken := g.TakeRows(2)
	if taken.RowCount() != 2 {
		t.Fatalf("expected 2 taken rows, got %d", taken.RowCount())
	}

	dropped := g.DropRows(1)
	if dropped.RowCount() != 2 {
		t.Fatalf("expected 2 dropped rows, got %d", dropped.RowCount())
	}
	if row, _ := dropped.GetRow(0); len(row) != 2 || row[0] != 2 {
		t.Fatalf("unexpected dropped first row: %#v", row)
	}

	rowCount := 0
	cellCount := 0
	returned := g.EachRow(func(_ int, row []int) {
		rowCount++
		cellCount += len(row)
		row[0] = 99
	})
	if returned != g {
		t.Fatal("expected EachRow to return receiver")
	}
	if rowCount != 3 || cellCount != 6 {
		t.Fatalf("unexpected EachRow counts: rows=%d cells=%d", rowCount, cellCount)
	}
	if value, _ := g.Get(0, 0); value != 1 {
		t.Fatalf("expected EachRow callback row to be copied, got %v", value)
	}

	first, ok := g.FirstRowWhere(func(_ int, row []int) bool {
		return len(row) == 3
	}).Get()
	if !ok || len(first) != 3 || first[0] != 4 {
		t.Fatalf("unexpected first matching row: %#v, ok=%v", first, ok)
	}
	first[0] = 77
	if value, _ := g.Get(2, 0); value != 4 {
		t.Fatalf("expected FirstRowWhere result row to be copied, got %v", value)
	}

	if !g.AnyRowMatch(func(_ int, row []int) bool { return len(row) == 1 }) {
		t.Fatal("expected AnyRowMatch to find row")
	}
	if g.AnyRowMatch(func(_ int, row []int) bool { return len(row) == 4 }) {
		t.Fatal("did not expect AnyRowMatch to find row")
	}
	if !g.AllRowsMatch(func(_ int, row []int) bool { return len(row) >= 1 }) {
		t.Fatal("expected AllRowsMatch to pass")
	}
	if g.AllRowsMatch(func(_ int, row []int) bool { return len(row) >= 2 }) {
		t.Fatal("did not expect AllRowsMatch to pass")
	}
}

func TestGridCellFluentHelpers(t *testing.T) {
	g := NewGrid[int]([]int{1, 2}, []int{3, 4, 5}, []int{6})

	filtered := g.WhereCells(func(_ int, _ int, value int) bool {
		return value%2 == 0
	})
	if filtered.RowCount() != 3 {
		t.Fatalf("expected 3 filtered rows, got %d", filtered.RowCount())
	}
	if row, _ := filtered.GetRow(1); len(row) != 1 || row[0] != 4 {
		t.Fatalf("unexpected filtered second row: %#v", row)
	}

	rejected := g.RejectCells(func(rowIndex int, _ int, _ int) bool {
		return rowIndex == 0
	})
	if rejected.RowCount() != 2 {
		t.Fatalf("expected 2 rejected rows, got %d", rejected.RowCount())
	}
	if row, _ := rejected.GetRow(0); len(row) != 3 || row[0] != 3 {
		t.Fatalf("unexpected rejected first row: %#v", row)
	}

	cellCount := 0
	sum := 0
	returned := g.EachCell(func(rowIndex int, columnIndex int, value int) {
		cellCount++
		sum += value + rowIndex + columnIndex
	})
	if returned != g {
		t.Fatal("expected EachCell to return receiver")
	}
	if cellCount != 6 {
		t.Fatalf("unexpected cell count from EachCell: %d", cellCount)
	}
	if sum != 30 {
		t.Fatalf("unexpected accumulated sum from EachCell: %d", sum)
	}

	rowIndex, columnIndex, value, ok := g.FirstCellWhere(func(_ int, _ int, value int) bool {
		return value == 4
	})
	if !ok || rowIndex != 1 || columnIndex != 1 || value != 4 {
		t.Fatalf("unexpected first matching cell: row=%d col=%d value=%v ok=%v", rowIndex, columnIndex, value, ok)
	}

	if !g.AnyCellMatch(func(_ int, _ int, value int) bool { return value == 6 }) {
		t.Fatal("expected AnyCellMatch to find cell")
	}
	if g.AnyCellMatch(func(_ int, _ int, value int) bool { return value == 7 }) {
		t.Fatal("did not expect AnyCellMatch to find cell")
	}
	if !g.AllCellsMatch(func(_ int, _ int, value int) bool { return value >= 1 }) {
		t.Fatal("expected AllCellsMatch to pass")
	}
	if g.AllCellsMatch(func(_ int, _ int, value int) bool { return value%2 == 0 }) {
		t.Fatal("did not expect AllCellsMatch to pass")
	}
}
