package mapping

import (
	"github.com/samber/lo"
	"github.com/samber/mo"
)

// Table is a 2D key-value structure: (rowKey, columnKey) -> value.
// Similar to Guava Table and backed by map[row]map[column]value.
// Zero value is ready to use.
type Table[R comparable, C comparable, V any] struct {
	data map[R]map[C]V
	size int
}

// NewTable creates an empty table.
func NewTable[R comparable, C comparable, V any]() *Table[R, C, V] {
	return &Table[R, C, V]{
		data: make(map[R]map[C]V),
	}
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
	if t == nil || t.data == nil {
		return zero, false
	}
	row, ok := t.data[rowKey]
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
	oldSize := len(t.data[rowKey])
	if len(rowValues) == 0 {
		delete(t.data, rowKey)
		t.size -= oldSize
		return
	}
	t.data[rowKey] = lo.Assign(map[C]V{}, rowValues)
	t.size += len(rowValues) - oldSize
}

// Row returns one row as a copied map.
func (t *Table[R, C, V]) Row(rowKey R) map[C]V {
	if t == nil || t.data == nil {
		return map[C]V{}
	}
	row, ok := t.data[rowKey]
	if !ok || len(row) == 0 {
		return map[C]V{}
	}
	return lo.Assign(map[C]V{}, row)
}

// Column returns one column as a copied map[row]value.
func (t *Table[R, C, V]) Column(columnKey C) map[R]V {
	if t == nil || len(t.data) == 0 {
		return map[R]V{}
	}
	out := make(map[R]V)
	for rowKey, row := range t.data {
		if value, ok := row[columnKey]; ok {
			out[rowKey] = value
		}
	}
	return out
}

// Delete removes one cell and reports whether it existed.
func (t *Table[R, C, V]) Delete(rowKey R, columnKey C) bool {
	if t == nil || t.data == nil {
		return false
	}
	row, ok := t.data[rowKey]
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
		delete(t.data, rowKey)
	}
	return true
}

// DeleteRow removes one row and reports whether it existed.
func (t *Table[R, C, V]) DeleteRow(rowKey R) bool {
	if t == nil || t.data == nil {
		return false
	}
	_, existed := t.data[rowKey]
	if existed {
		t.size -= len(t.data[rowKey])
		delete(t.data, rowKey)
	}
	return existed
}

// DeleteColumn removes one column from all rows and returns removed cell count.
func (t *Table[R, C, V]) DeleteColumn(columnKey C) int {
	if t == nil || len(t.data) == 0 {
		return 0
	}
	removed := 0
	for rowKey, row := range t.data {
		if _, ok := row[columnKey]; ok {
			delete(row, columnKey)
			removed++
			t.size--
		}
		if len(row) == 0 {
			delete(t.data, rowKey)
		}
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
	return len(t.data)
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
	clear(t.data)
	t.size = 0
}

// RowKeys returns all row keys.
func (t *Table[R, C, V]) RowKeys() []R {
	if t == nil || len(t.data) == 0 {
		return nil
	}
	return lo.Keys(t.data)
}

// ColumnKeys returns all unique column keys.
func (t *Table[R, C, V]) ColumnKeys() []C {
	if t == nil || len(t.data) == 0 {
		return nil
	}
	set := make(map[C]struct{})
	for _, row := range t.data {
		for columnKey := range row {
			set[columnKey] = struct{}{}
		}
	}
	return lo.Keys(set)
}

// All returns a deep-copied built-in map.
func (t *Table[R, C, V]) All() map[R]map[C]V {
	if t == nil || len(t.data) == 0 {
		return map[R]map[C]V{}
	}
	out := make(map[R]map[C]V, len(t.data))
	for rowKey, row := range t.data {
		out[rowKey] = lo.Assign(map[C]V{}, row)
	}
	return out
}

// Range iterates all cells until fn returns false.
func (t *Table[R, C, V]) Range(fn func(rowKey R, columnKey C, value V) bool) {
	if t == nil || fn == nil {
		return
	}
	for rowKey, row := range t.data {
		for columnKey, value := range row {
			if !fn(rowKey, columnKey, value) {
				return
			}
		}
	}
}

func (t *Table[R, C, V]) ensureInit() {
	if t.data == nil {
		t.data = make(map[R]map[C]V)
	}
}

func (t *Table[R, C, V]) ensureRow(rowKey R) map[C]V {
	row, ok := t.data[rowKey]
	if !ok {
		row = make(map[C]V)
		t.data[rowKey] = row
	}
	return row
}
