package list

import "github.com/samber/mo"

// WhereRows returns a new grid containing only rows that match predicate.
func (g *Grid[T]) WhereRows(predicate func(index int, row []T) bool) *Grid[T] {
	if g == nil || predicate == nil || g.rows.Len() == 0 {
		return NewGrid[T]()
	}
	filtered := NewGridWithCapacity[T](g.rows.Len())
	g.rows.Range(func(index int, row *List[T]) bool {
		rowCopy := cloneGridRowList(row)
		if predicate(index, gridRowItems(rowCopy)) {
			filtered.appendOwnedRow(rowCopy)
		}
		return true
	})
	return filtered
}

// RejectRows returns a new grid excluding rows that match predicate.
func (g *Grid[T]) RejectRows(predicate func(index int, row []T) bool) *Grid[T] {
	if g == nil || predicate == nil || g.rows.Len() == 0 {
		return NewGrid[T]()
	}
	rejected := NewGridWithCapacity[T](g.rows.Len())
	g.rows.Range(func(index int, row *List[T]) bool {
		rowCopy := cloneGridRowList(row)
		if !predicate(index, gridRowItems(rowCopy)) {
			rejected.appendOwnedRow(rowCopy)
		}
		return true
	})
	return rejected
}

// TakeRows returns the first n rows as a new grid.
func (g *Grid[T]) TakeRows(n int) *Grid[T] {
	if g == nil || n <= 0 || g.rows.Len() == 0 {
		return NewGrid[T]()
	}
	if n >= g.rows.Len() {
		return g.Clone()
	}
	return newGridFromRowLists(g.rows.Take(n))
}

// DropRows returns a new grid without the first n rows.
func (g *Grid[T]) DropRows(n int) *Grid[T] {
	if g == nil || g.rows.Len() == 0 {
		return NewGrid[T]()
	}
	if n <= 0 {
		return g.Clone()
	}
	if n >= g.rows.Len() {
		return NewGrid[T]()
	}
	return newGridFromRowLists(g.rows.Drop(n))
}

// EachRow invokes fn for every row and returns the receiver for chaining.
func (g *Grid[T]) EachRow(fn func(index int, row []T)) *Grid[T] {
	if g == nil {
		return NewGrid[T]()
	}
	if fn == nil {
		return g
	}
	g.rows.Range(func(index int, row *List[T]) bool {
		fn(index, gridRowValues(row))
		return true
	})
	return g
}

// FirstRowWhere returns the first row matching predicate.
func (g *Grid[T]) FirstRowWhere(predicate func(index int, row []T) bool) mo.Option[[]T] {
	if g == nil || predicate == nil || g.rows.Len() == 0 {
		return mo.None[[]T]()
	}
	found := mo.None[[]T]()
	g.rows.Range(func(index int, row *List[T]) bool {
		rowCopy := cloneGridRowList(row)
		if !predicate(index, gridRowItems(rowCopy)) {
			return true
		}
		found = mo.Some(gridRowItems(rowCopy))
		return false
	})
	return found
}

// AnyRowMatch reports whether any row matches predicate.
func (g *Grid[T]) AnyRowMatch(predicate func(index int, row []T) bool) bool {
	_, ok := g.FirstRowWhere(predicate).Get()
	return ok
}

// AllRowsMatch reports whether all rows match predicate.
func (g *Grid[T]) AllRowsMatch(predicate func(index int, row []T) bool) bool {
	if g == nil || g.rows.Len() == 0 || predicate == nil {
		return false
	}
	matched := true
	g.rows.Range(func(index int, row *List[T]) bool {
		if predicate(index, gridRowValues(row)) {
			return true
		}
		matched = false
		return false
	})
	return matched
}

// WhereCells returns a new grid containing only cells that match predicate.
// Rows without matching cells are dropped.
func (g *Grid[T]) WhereCells(predicate func(rowIndex int, columnIndex int, value T) bool) *Grid[T] {
	if g == nil || predicate == nil || g.rows.Len() == 0 {
		return NewGrid[T]()
	}
	filtered := NewGridWithCapacity[T](g.rows.Len())
	g.rows.Range(func(rowIndex int, row *List[T]) bool {
		nextRow := NewListWithCapacity[T](row.Len())
		row.Range(func(columnIndex int, value T) bool {
			if predicate(rowIndex, columnIndex, value) {
				nextRow.Add(value)
			}
			return true
		})
		if nextRow.Len() > 0 {
			filtered.appendOwnedRow(nextRow)
		}
		return true
	})
	return filtered
}

// RejectCells returns a new grid excluding cells that match predicate.
// Rows without remaining cells are dropped.
func (g *Grid[T]) RejectCells(predicate func(rowIndex int, columnIndex int, value T) bool) *Grid[T] {
	if g == nil || predicate == nil || g.rows.Len() == 0 {
		return NewGrid[T]()
	}
	rejected := NewGridWithCapacity[T](g.rows.Len())
	g.rows.Range(func(rowIndex int, row *List[T]) bool {
		nextRow := NewListWithCapacity[T](row.Len())
		row.Range(func(columnIndex int, value T) bool {
			if !predicate(rowIndex, columnIndex, value) {
				nextRow.Add(value)
			}
			return true
		})
		if nextRow.Len() > 0 {
			rejected.appendOwnedRow(nextRow)
		}
		return true
	})
	return rejected
}

// EachCell invokes fn for every cell and returns the receiver for chaining.
func (g *Grid[T]) EachCell(fn func(rowIndex int, columnIndex int, value T)) *Grid[T] {
	if g == nil {
		return NewGrid[T]()
	}
	if fn == nil {
		return g
	}
	g.rows.Range(func(rowIndex int, row *List[T]) bool {
		row.Range(func(columnIndex int, value T) bool {
			fn(rowIndex, columnIndex, value)
			return true
		})
		return true
	})
	return g
}

// FirstCellWhere returns the first cell matching predicate.
func (g *Grid[T]) FirstCellWhere(predicate func(rowIndex int, columnIndex int, value T) bool) (int, int, T, bool) {
	var zero T
	if g == nil || predicate == nil || g.rows.Len() == 0 {
		return 0, 0, zero, false
	}
	foundRowIndex := 0
	foundColumnIndex := 0
	foundValue := zero
	found := false
	g.rows.Range(func(rowIndex int, row *List[T]) bool {
		row.Range(func(columnIndex int, value T) bool {
			if !predicate(rowIndex, columnIndex, value) {
				return true
			}
			foundRowIndex = rowIndex
			foundColumnIndex = columnIndex
			foundValue = value
			found = true
			return false
		})
		return !found
	})
	return foundRowIndex, foundColumnIndex, foundValue, found
}

// AnyCellMatch reports whether any cell matches predicate.
func (g *Grid[T]) AnyCellMatch(predicate func(rowIndex int, columnIndex int, value T) bool) bool {
	_, _, _, ok := g.FirstCellWhere(predicate)
	return ok
}

// AllCellsMatch reports whether all cells match predicate.
func (g *Grid[T]) AllCellsMatch(predicate func(rowIndex int, columnIndex int, value T) bool) bool {
	if g == nil || g.size == 0 || predicate == nil {
		return false
	}
	matched := true
	g.rows.Range(func(rowIndex int, row *List[T]) bool {
		row.Range(func(columnIndex int, value T) bool {
			if predicate(rowIndex, columnIndex, value) {
				return true
			}
			matched = false
			return false
		})
		return matched
	})
	return matched
}
