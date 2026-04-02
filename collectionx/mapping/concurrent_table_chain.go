package mapping

// WhereRows returns a filtered table snapshot.
func (t *ConcurrentTable[R, C, V]) WhereRows(predicate func(rowKey R, row map[C]V) bool) *Table[R, C, V] {
	return t.Snapshot().WhereRows(predicate)
}

// RejectRows returns a filtered table snapshot that excludes matching rows.
func (t *ConcurrentTable[R, C, V]) RejectRows(predicate func(rowKey R, row map[C]V) bool) *Table[R, C, V] {
	return t.Snapshot().RejectRows(predicate)
}

// WhereCells returns a filtered table snapshot containing only matching cells.
func (t *ConcurrentTable[R, C, V]) WhereCells(predicate func(rowKey R, columnKey C, value V) bool) *Table[R, C, V] {
	return t.Snapshot().WhereCells(predicate)
}

// RejectCells returns a filtered table snapshot excluding matching cells.
func (t *ConcurrentTable[R, C, V]) RejectCells(predicate func(rowKey R, columnKey C, value V) bool) *Table[R, C, V] {
	return t.Snapshot().RejectCells(predicate)
}

// EachRow iterates a stable snapshot and returns it for chaining.
func (t *ConcurrentTable[R, C, V]) EachRow(fn func(rowKey R, row map[C]V)) *Table[R, C, V] {
	return t.Snapshot().EachRow(fn)
}

// EachCell iterates a stable snapshot and returns it for chaining.
func (t *ConcurrentTable[R, C, V]) EachCell(fn func(rowKey R, columnKey C, value V)) *Table[R, C, V] {
	return t.Snapshot().EachCell(fn)
}

// FirstCellWhere returns the first cell matching predicate from a stable snapshot.
func (t *ConcurrentTable[R, C, V]) FirstCellWhere(predicate func(rowKey R, columnKey C, value V) bool) (R, C, V, bool) {
	return t.Snapshot().FirstCellWhere(predicate)
}

// AnyCellMatch reports whether any cell in a stable snapshot matches predicate.
func (t *ConcurrentTable[R, C, V]) AnyCellMatch(predicate func(rowKey R, columnKey C, value V) bool) bool {
	return t.Snapshot().AnyCellMatch(predicate)
}

// AllCellsMatch reports whether all cells in a stable snapshot match predicate.
func (t *ConcurrentTable[R, C, V]) AllCellsMatch(predicate func(rowKey R, columnKey C, value V) bool) bool {
	return t.Snapshot().AllCellsMatch(predicate)
}
