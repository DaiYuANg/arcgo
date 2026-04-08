package mapping

import (
	"sync"

	"github.com/samber/mo"
)

// ConcurrentTable is a goroutine-safe 2D key-value structure.
// Zero value is ready to use.
type ConcurrentTable[R comparable, C comparable, V any] struct {
	mu   sync.RWMutex
	core *Table[R, C, V]

	jsonCache   []byte
	stringCache string
	jsonDirty   bool
}

// NewConcurrentTable creates an empty concurrent table.
func NewConcurrentTable[R comparable, C comparable, V any]() *ConcurrentTable[R, C, V] {
	return &ConcurrentTable[R, C, V]{
		core: NewTable[R, C, V](),
	}
}

// Put sets value at (rowKey, columnKey).
func (t *ConcurrentTable[R, C, V]) Put(rowKey R, columnKey C, value V) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ensureInitLocked()
	t.core.Put(rowKey, columnKey, value)
	t.invalidateSerializationCacheLocked()
}

// Get returns value at (rowKey, columnKey).
func (t *ConcurrentTable[R, C, V]) Get(rowKey R, columnKey C) (V, bool) {
	var zero V
	if t == nil {
		return zero, false
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.core == nil {
		return zero, false
	}
	return t.core.Get(rowKey, columnKey)
}

// GetOption returns value at (rowKey, columnKey) as mo.Option.
func (t *ConcurrentTable[R, C, V]) GetOption(rowKey R, columnKey C) mo.Option[V] {
	value, ok := t.Get(rowKey, columnKey)
	if !ok {
		return mo.None[V]()
	}
	return mo.Some(value)
}

// SetRow replaces one entire row.
// Empty rowValues removes the row.
func (t *ConcurrentTable[R, C, V]) SetRow(rowKey R, rowValues map[C]V) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ensureInitLocked()
	t.core.SetRow(rowKey, rowValues)
	t.invalidateSerializationCacheLocked()
}

// Row returns one row as a copied map.
func (t *ConcurrentTable[R, C, V]) Row(rowKey R) map[C]V {
	if t == nil {
		return map[C]V{}
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.core == nil {
		return map[C]V{}
	}
	return t.core.Row(rowKey)
}

// Column returns one column as a copied map[row]value.
func (t *ConcurrentTable[R, C, V]) Column(columnKey C) map[R]V {
	if t == nil {
		return map[R]V{}
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.core == nil {
		return map[R]V{}
	}
	return t.core.Column(columnKey)
}

// Delete removes one cell and reports whether it existed.
func (t *ConcurrentTable[R, C, V]) Delete(rowKey R, columnKey C) bool {
	if t == nil {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.core == nil {
		return false
	}
	removed := t.core.Delete(rowKey, columnKey)
	if removed {
		t.invalidateSerializationCacheLocked()
	}
	return removed
}

// DeleteRow removes one row and reports whether it existed.
func (t *ConcurrentTable[R, C, V]) DeleteRow(rowKey R) bool {
	if t == nil {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.core == nil {
		return false
	}
	removed := t.core.DeleteRow(rowKey)
	if removed {
		t.invalidateSerializationCacheLocked()
	}
	return removed
}

// DeleteColumn removes one column from all rows and returns removed cell count.
func (t *ConcurrentTable[R, C, V]) DeleteColumn(columnKey C) int {
	if t == nil {
		return 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.core == nil {
		return 0
	}
	removed := t.core.DeleteColumn(columnKey)
	if removed > 0 {
		t.invalidateSerializationCacheLocked()
	}
	return removed
}

// Has reports whether cell exists.
func (t *ConcurrentTable[R, C, V]) Has(rowKey R, columnKey C) bool {
	_, ok := t.Get(rowKey, columnKey)
	return ok
}

// RowCount returns total row count.
func (t *ConcurrentTable[R, C, V]) RowCount() int {
	if t == nil {
		return 0
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.core == nil {
		return 0
	}
	return t.core.RowCount()
}

// Len returns total cell count.
func (t *ConcurrentTable[R, C, V]) Len() int {
	if t == nil {
		return 0
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.core == nil {
		return 0
	}
	return t.core.Len()
}

// IsEmpty reports whether table has no cells.
func (t *ConcurrentTable[R, C, V]) IsEmpty() bool {
	return t.Len() == 0
}

// Clear removes all cells.
func (t *ConcurrentTable[R, C, V]) Clear() {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.core == nil {
		return
	}
	t.core.Clear()
	t.jsonCache = nil
	t.stringCache = ""
	t.jsonDirty = false
}

// RowKeys returns all row keys.
func (t *ConcurrentTable[R, C, V]) RowKeys() []R {
	if t == nil {
		return nil
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.core == nil {
		return nil
	}
	return t.core.RowKeys()
}

// ColumnKeys returns all unique column keys.
func (t *ConcurrentTable[R, C, V]) ColumnKeys() []C {
	if t == nil {
		return nil
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.core == nil {
		return nil
	}
	return t.core.ColumnKeys()
}

// All returns a deep-copied built-in map.
func (t *ConcurrentTable[R, C, V]) All() map[R]map[C]V {
	if t == nil {
		return map[R]map[C]V{}
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.core == nil {
		return map[R]map[C]V{}
	}
	return t.core.All()
}

// Snapshot returns an immutable-style copy in a normal Table.
func (t *ConcurrentTable[R, C, V]) Snapshot() *Table[R, C, V] {
	out := NewTable[R, C, V]()
	for rowKey, row := range t.All() {
		out.SetRow(rowKey, row)
	}
	return out
}

// Range iterates all cells snapshots until fn returns false.
func (t *ConcurrentTable[R, C, V]) Range(fn func(rowKey R, columnKey C, value V) bool) {
	if t == nil || fn == nil {
		return
	}
	for rowKey, row := range t.All() {
		for columnKey, value := range row {
			if !fn(rowKey, columnKey, value) {
				return
			}
		}
	}
}

func (t *ConcurrentTable[R, C, V]) ensureInitLocked() {
	if t.core == nil {
		t.core = NewTable[R, C, V]()
	}
}

func (t *ConcurrentTable[R, C, V]) invalidateSerializationCacheLocked() {
	t.jsonCache = nil
	t.stringCache = ""
	t.jsonDirty = true
}
