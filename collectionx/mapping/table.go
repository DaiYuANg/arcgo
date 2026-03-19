package mapping

import (
	"github.com/samber/mo"
)

// Table is a 2D key-value structure: (rowKey, columnKey) -> value.
// Similar to Guava Table and backed by map[row]map[column]value.
// Zero value is ready to use.
type Table[R comparable, C comparable, V any] struct {
	data Map[R, map[C]V]
	size int
}

// NewTable creates an empty table.
func NewTable[R comparable, C comparable, V any]() *Table[R, C, V] {
	return &Table[R, C, V]{}
}

// Put sets value at (rowKey, columnKey).
func (t *Table[R, C, V]) Put(rowKey R, columnKey C, value V) {
	if t == nil {
		return
	}
	t.ensureInit()
	row := t.ensureRow(rowKey)
	if _, existed := row[columnKey]; !existed {
		t.size++
	}
	row[columnKey] = value
}

// Get returns value at (rowKey, columnKey).
func (t *Table[R, C, V]) Get(rowKey R, columnKey C) (V, bool) {
	var zero V
	if t == nil {
		return zero, false
	}
	row, ok := t.data.Get(rowKey)
	if !ok {
		return zero, false
	}
	value, ok := row[columnKey]
	return value, ok
}

// GetOption returns value at (rowKey, columnKey) as mo.Option.
func (t *Table[R, C, V]) GetOption(rowKey R, columnKey C) mo.Option[V] {
	value, ok := t.Get(rowKey, columnKey)
	if !ok {
		return mo.None[V]()
	}
	return mo.Some(value)
}

// SetRow replaces one entire row.
// Empty rowValues removes the row.
func (t *Table[R, C, V]) SetRow(rowKey R, rowValues map[C]V) {
	if t == nil {
		return
	}
	t.ensureInit()
	oldRow, _ := t.data.Get(rowKey)
	oldSize := len(oldRow)
	if len(rowValues) == 0 {
		t.data.Delete(rowKey)
		t.size -= oldSize
		return
	}
	rowCopy := make(map[C]V, len(rowValues))
	for columnKey, value := range rowValues {
		rowCopy[columnKey] = value
	}
	t.data.Set(rowKey, rowCopy)
	t.size += len(rowValues) - oldSize
}

// Row returns one row as a copied map.
func (t *Table[R, C, V]) Row(rowKey R) map[C]V {
	if t == nil {
		return map[C]V{}
	}
	row, ok := t.data.Get(rowKey)
	if !ok || len(row) == 0 {
		return map[C]V{}
	}
	out := make(map[C]V, len(row))
	for columnKey, value := range row {
		out[columnKey] = value
	}
	return out
}

// Column returns one column as a copied map[row]value.
func (t *Table[R, C, V]) Column(columnKey C) map[R]V {
	if t == nil || t.data.Len() == 0 {
		return map[R]V{}
	}
	out := make(map[R]V)
	t.data.Range(func(rowKey R, row map[C]V) bool {
		if value, ok := row[columnKey]; ok {
			out[rowKey] = value
		}
		return true
	})
	return out
}

// Delete removes one cell and reports whether it existed.
func (t *Table[R, C, V]) Delete(rowKey R, columnKey C) bool {
	if t == nil {
		return false
	}
	row, ok := t.data.Get(rowKey)
	if !ok {
		return false
	}
	_, existed := row[columnKey]
	if !existed {
		return false
	}

	delete(row, columnKey)
	t.size--
	if len(row) == 0 {
		t.data.Delete(rowKey)
	}
	return true
}

// DeleteRow removes one row and reports whether it existed.
func (t *Table[R, C, V]) DeleteRow(rowKey R) bool {
	if t == nil {
		return false
	}
	row, existed := t.data.Get(rowKey)
	if existed {
		t.size -= len(row)
		t.data.Delete(rowKey)
	}
	return existed
}

// DeleteColumn removes one column from all rows and returns removed cell count.
func (t *Table[R, C, V]) DeleteColumn(columnKey C) int {
	if t == nil || t.data.Len() == 0 {
		return 0
	}
	removed := 0
	rowsToDelete := make([]R, 0)
	t.data.Range(func(rowKey R, row map[C]V) bool {
		if _, ok := row[columnKey]; ok {
			delete(row, columnKey)
			removed++
			t.size--
		}
		if len(row) == 0 {
			rowsToDelete = append(rowsToDelete, rowKey)
		}
		return true
	})
	for _, rowKey := range rowsToDelete {
		t.data.Delete(rowKey)
	}
	return removed
}

// Has reports whether cell exists.
func (t *Table[R, C, V]) Has(rowKey R, columnKey C) bool {
	_, ok := t.Get(rowKey, columnKey)
	return ok
}

// RowCount returns total row count.
func (t *Table[R, C, V]) RowCount() int {
	if t == nil {
		return 0
	}
	return t.data.Len()
}

// Len returns total cell count.
func (t *Table[R, C, V]) Len() int {
	if t == nil {
		return 0
	}
	return t.size
}

// IsEmpty reports whether table has no cells.
func (t *Table[R, C, V]) IsEmpty() bool {
	return t.Len() == 0
}

// Clear removes all cells.
func (t *Table[R, C, V]) Clear() {
	if t == nil {
		return
	}
	t.data.Clear()
	t.size = 0
}

// RowKeys returns all row keys.
func (t *Table[R, C, V]) RowKeys() []R {
	if t == nil || t.data.Len() == 0 {
		return nil
	}
	return t.data.Keys()
}

// ColumnKeys returns all unique column keys.
func (t *Table[R, C, V]) ColumnKeys() []C {
	if t == nil || t.data.Len() == 0 {
		return nil
	}
	set := make(map[C]struct{})
	t.data.Range(func(_ R, row map[C]V) bool {
		for columnKey := range row {
			set[columnKey] = struct{}{}
		}
		return true
	})
	out := make([]C, 0, len(set))
	for columnKey := range set {
		out = append(out, columnKey)
	}
	return out
}

// All returns a deep-copied built-in map.
func (t *Table[R, C, V]) All() map[R]map[C]V {
	if t == nil || t.data.Len() == 0 {
		return map[R]map[C]V{}
	}
	out := make(map[R]map[C]V, t.data.Len())
	t.data.Range(func(rowKey R, row map[C]V) bool {
		rowCopy := make(map[C]V, len(row))
		for columnKey, value := range row {
			rowCopy[columnKey] = value
		}
		out[rowKey] = rowCopy
		return true
	})
	return out
}

// Range iterates all cells until fn returns false.
func (t *Table[R, C, V]) Range(fn func(rowKey R, columnKey C, value V) bool) {
	if t == nil || fn == nil {
		return
	}
	t.data.Range(func(rowKey R, row map[C]V) bool {
		for columnKey, value := range row {
			if !fn(rowKey, columnKey, value) {
				return false
			}
		}
		return true
	})
}

func (t *Table[R, C, V]) ensureInit() {
	t.data.ensureInit()
}

func (t *Table[R, C, V]) ensureRow(rowKey R) map[C]V {
	row, ok := t.data.Get(rowKey)
	if !ok {
		row = make(map[C]V)
		t.data.Set(rowKey, row)
	}
	return row
}
