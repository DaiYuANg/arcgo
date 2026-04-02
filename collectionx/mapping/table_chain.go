package mapping

// WhereRows returns a new table containing only rows that match predicate.
func (t *Table[R, C, V]) WhereRows(predicate func(rowKey R, row map[C]V) bool) *Table[R, C, V] {
	if t == nil || predicate == nil || t.data.Len() == 0 {
		return NewTable[R, C, V]()
	}
	filtered := NewTable[R, C, V]()
	t.data.Range(func(rowKey R, row map[C]V) bool {
		if predicate(rowKey, row) {
			filtered.SetRow(rowKey, row)
		}
		return true
	})
	return filtered
}

// RejectRows returns a new table excluding rows that match predicate.
func (t *Table[R, C, V]) RejectRows(predicate func(rowKey R, row map[C]V) bool) *Table[R, C, V] {
	if t == nil || predicate == nil || t.data.Len() == 0 {
		return NewTable[R, C, V]()
	}
	rejected := NewTable[R, C, V]()
	t.data.Range(func(rowKey R, row map[C]V) bool {
		if !predicate(rowKey, row) {
			rejected.SetRow(rowKey, row)
		}
		return true
	})
	return rejected
}

// WhereCells returns a new table containing only cells that match predicate.
func (t *Table[R, C, V]) WhereCells(predicate func(rowKey R, columnKey C, value V) bool) *Table[R, C, V] {
	if t == nil || predicate == nil || t.size == 0 {
		return NewTable[R, C, V]()
	}
	filtered := NewTable[R, C, V]()
	t.Range(func(rowKey R, columnKey C, value V) bool {
		if predicate(rowKey, columnKey, value) {
			filtered.Put(rowKey, columnKey, value)
		}
		return true
	})
	return filtered
}

// RejectCells returns a new table excluding cells that match predicate.
func (t *Table[R, C, V]) RejectCells(predicate func(rowKey R, columnKey C, value V) bool) *Table[R, C, V] {
	if t == nil || predicate == nil || t.size == 0 {
		return NewTable[R, C, V]()
	}
	rejected := NewTable[R, C, V]()
	t.Range(func(rowKey R, columnKey C, value V) bool {
		if !predicate(rowKey, columnKey, value) {
			rejected.Put(rowKey, columnKey, value)
		}
		return true
	})
	return rejected
}

// EachRow invokes fn for each row and returns the receiver for chaining.
func (t *Table[R, C, V]) EachRow(fn func(rowKey R, row map[C]V)) *Table[R, C, V] {
	if t == nil {
		return NewTable[R, C, V]()
	}
	if fn == nil {
		return t
	}
	t.data.Range(func(rowKey R, row map[C]V) bool {
		fn(rowKey, row)
		return true
	})
	return t
}

// EachCell invokes fn for each cell and returns the receiver for chaining.
func (t *Table[R, C, V]) EachCell(fn func(rowKey R, columnKey C, value V)) *Table[R, C, V] {
	if t == nil {
		return NewTable[R, C, V]()
	}
	if fn == nil {
		return t
	}
	t.Range(func(rowKey R, columnKey C, value V) bool {
		fn(rowKey, columnKey, value)
		return true
	})
	return t
}

// FirstCellWhere returns the first cell matching predicate.
func (t *Table[R, C, V]) FirstCellWhere(predicate func(rowKey R, columnKey C, value V) bool) (R, C, V, bool) {
	var zeroR R
	var zeroC C
	var zeroV V
	if t == nil || predicate == nil || t.size == 0 {
		return zeroR, zeroC, zeroV, false
	}
	foundR, foundC, foundV := zeroR, zeroC, zeroV
	ok := false
	t.Range(func(rowKey R, columnKey C, value V) bool {
		if !predicate(rowKey, columnKey, value) {
			return true
		}
		foundR, foundC, foundV = rowKey, columnKey, value
		ok = true
		return false
	})
	return foundR, foundC, foundV, ok
}

// AnyCellMatch reports whether any cell matches predicate.
func (t *Table[R, C, V]) AnyCellMatch(predicate func(rowKey R, columnKey C, value V) bool) bool {
	_, _, _, ok := t.FirstCellWhere(predicate)
	return ok
}

// AllCellsMatch reports whether all cells match predicate.
func (t *Table[R, C, V]) AllCellsMatch(predicate func(rowKey R, columnKey C, value V) bool) bool {
	if t == nil || t.size == 0 || predicate == nil {
		return false
	}
	matched := true
	t.Range(func(rowKey R, columnKey C, value V) bool {
		if predicate(rowKey, columnKey, value) {
			return true
		}
		matched = false
		return false
	})
	return matched
}
