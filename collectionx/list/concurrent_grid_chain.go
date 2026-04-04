package list

import "github.com/samber/mo"

// WhereRows returns a filtered grid snapshot.
func (g *ConcurrentGrid[T]) WhereRows(predicate func(index int, row []T) bool) *Grid[T] {
	return g.Snapshot().WhereRows(predicate)
}

// RejectRows returns a filtered grid snapshot that excludes matching rows.
func (g *ConcurrentGrid[T]) RejectRows(predicate func(index int, row []T) bool) *Grid[T] {
	return g.Snapshot().RejectRows(predicate)
}

// TakeRows returns the first n rows from a stable snapshot.
func (g *ConcurrentGrid[T]) TakeRows(n int) *Grid[T] {
	return g.Snapshot().TakeRows(n)
}

// DropRows returns a stable snapshot without the first n rows.
func (g *ConcurrentGrid[T]) DropRows(n int) *Grid[T] {
	return g.Snapshot().DropRows(n)
}

// EachRow iterates a stable snapshot and returns it for chaining.
func (g *ConcurrentGrid[T]) EachRow(fn func(index int, row []T)) *Grid[T] {
	return g.Snapshot().EachRow(fn)
}

// FirstRowWhere returns the first row matching predicate from a stable snapshot.
func (g *ConcurrentGrid[T]) FirstRowWhere(predicate func(index int, row []T) bool) mo.Option[[]T] {
	return g.Snapshot().FirstRowWhere(predicate)
}

// AnyRowMatch reports whether any row in a stable snapshot matches predicate.
func (g *ConcurrentGrid[T]) AnyRowMatch(predicate func(index int, row []T) bool) bool {
	return g.Snapshot().AnyRowMatch(predicate)
}

// AllRowsMatch reports whether all rows in a stable snapshot match predicate.
func (g *ConcurrentGrid[T]) AllRowsMatch(predicate func(index int, row []T) bool) bool {
	return g.Snapshot().AllRowsMatch(predicate)
}

// WhereCells returns a filtered grid snapshot containing only matching cells.
func (g *ConcurrentGrid[T]) WhereCells(predicate func(rowIndex int, columnIndex int, value T) bool) *Grid[T] {
	return g.Snapshot().WhereCells(predicate)
}

// RejectCells returns a filtered grid snapshot excluding matching cells.
func (g *ConcurrentGrid[T]) RejectCells(predicate func(rowIndex int, columnIndex int, value T) bool) *Grid[T] {
	return g.Snapshot().RejectCells(predicate)
}

// EachCell iterates a stable snapshot and returns it for chaining.
func (g *ConcurrentGrid[T]) EachCell(fn func(rowIndex int, columnIndex int, value T)) *Grid[T] {
	return g.Snapshot().EachCell(fn)
}

// FirstCellWhere returns the first cell matching predicate from a stable snapshot.
func (g *ConcurrentGrid[T]) FirstCellWhere(predicate func(rowIndex int, columnIndex int, value T) bool) (int, int, T, bool) {
	return g.Snapshot().FirstCellWhere(predicate)
}

// AnyCellMatch reports whether any cell in a stable snapshot matches predicate.
func (g *ConcurrentGrid[T]) AnyCellMatch(predicate func(rowIndex int, columnIndex int, value T) bool) bool {
	return g.Snapshot().AnyCellMatch(predicate)
}

// AllCellsMatch reports whether all cells in a stable snapshot match predicate.
func (g *ConcurrentGrid[T]) AllCellsMatch(predicate func(rowIndex int, columnIndex int, value T) bool) bool {
	return g.Snapshot().AllCellsMatch(predicate)
}
