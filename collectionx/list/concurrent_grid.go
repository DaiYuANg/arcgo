package list

import "sync"

// ConcurrentGrid is a goroutine-safe ordered two-dimensional container.
// Zero value is ready to use.
type ConcurrentGrid[T any] struct {
	mu   sync.RWMutex
	core *Grid[T]
}

// NewConcurrentGrid creates a grid and copies optional rows.
func NewConcurrentGrid[T any](rows ...[]T) *ConcurrentGrid[T] {
	return NewConcurrentGridWithCapacity(len(rows), rows...)
}

// NewConcurrentGridWithCapacity creates a grid with preallocated row capacity and optional rows.
func NewConcurrentGridWithCapacity[T any](capacity int, rows ...[]T) *ConcurrentGrid[T] {
	if capacity < len(rows) {
		capacity = len(rows)
	}
	if capacity <= 0 {
		return &ConcurrentGrid[T]{}
	}
	return &ConcurrentGrid[T]{
		core: NewGridWithCapacity(capacity, rows...),
	}
}

// AddRow appends one row.
func (g *ConcurrentGrid[T]) AddRow(items ...T) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.ensureInitLocked()
	g.core.AddRow(items...)
}

// AddRowList appends one row from a list copy.
func (g *ConcurrentGrid[T]) AddRowList(items *List[T]) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.ensureInitLocked()
	g.core.AddRowList(items)
}

// AddRows appends multiple rows.
func (g *ConcurrentGrid[T]) AddRows(rows ...[]T) {
	if len(rows) == 0 {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.ensureInitLocked()
	g.core.AddRows(rows...)
}

// AddRowsList appends multiple rows from copied row lists.
func (g *ConcurrentGrid[T]) AddRowsList(rows *List[*List[T]]) {
	if rows == nil || rows.Len() == 0 {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.ensureInitLocked()
	g.core.AddRowsList(rows)
}

// Merge appends copied rows from a normal grid.
func (g *ConcurrentGrid[T]) Merge(other *Grid[T]) *ConcurrentGrid[T] {
	if other == nil {
		return g
	}
	g.AddRows(other.Values()...)
	return g
}

// MergeConcurrent appends copied rows from another concurrent grid snapshot.
func (g *ConcurrentGrid[T]) MergeConcurrent(other *ConcurrentGrid[T]) *ConcurrentGrid[T] {
	if other == nil {
		return g
	}
	g.AddRows(other.Values()...)
	return g
}

// Get returns value at row and column indexes.
func (g *ConcurrentGrid[T]) Get(rowIndex, columnIndex int) (T, bool) {
	var zero T
	if g == nil {
		return zero, false
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.core == nil {
		return zero, false
	}
	return g.core.Get(rowIndex, columnIndex)
}

// GetRow returns a copied row at index.
func (g *ConcurrentGrid[T]) GetRow(index int) ([]T, bool) {
	if g == nil {
		return nil, false
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.core == nil {
		return nil, false
	}
	return g.core.GetRow(index)
}

// GetRowList returns a copied row list at index.
func (g *ConcurrentGrid[T]) GetRowList(index int) (*List[T], bool) {
	if g == nil {
		return nil, false
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.core == nil {
		return nil, false
	}
	return g.core.GetRowList(index)
}

// Set replaces value at row and column indexes.
func (g *ConcurrentGrid[T]) Set(rowIndex, columnIndex int, item T) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.core == nil {
		return false
	}
	return g.core.Set(rowIndex, columnIndex, item)
}

// SetRow replaces one row.
func (g *ConcurrentGrid[T]) SetRow(index int, items ...T) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.core == nil {
		return false
	}
	return g.core.SetRow(index, items...)
}

// RemoveRow removes and returns one copied row.
func (g *ConcurrentGrid[T]) RemoveRow(index int) ([]T, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.core == nil {
		return nil, false
	}
	return g.core.RemoveRow(index)
}

// RowCount returns total row count.
func (g *ConcurrentGrid[T]) RowCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.core == nil {
		return 0
	}
	return g.core.RowCount()
}

// Len returns total cell count across all rows.
func (g *ConcurrentGrid[T]) Len() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.core == nil {
		return 0
	}
	return g.core.Len()
}

// IsEmpty reports whether grid has no rows.
func (g *ConcurrentGrid[T]) IsEmpty() bool {
	return g.RowCount() == 0
}

// Clear removes all rows.
func (g *ConcurrentGrid[T]) Clear() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.core == nil {
		return
	}
	g.core.Clear()
}

// Values returns copied rows.
func (g *ConcurrentGrid[T]) Values() [][]T {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.core == nil {
		return nil
	}
	return g.core.Values()
}

// Range iterates a stable snapshot from top to bottom until fn returns false.
func (g *ConcurrentGrid[T]) Range(fn func(index int, row []T) bool) {
	if fn == nil {
		return
	}
	g.Snapshot().Range(fn)
}

// Snapshot returns an immutable-style copy in a normal Grid.
func (g *ConcurrentGrid[T]) Snapshot() *Grid[T] {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.core == nil {
		return NewGrid[T]()
	}
	return g.core.Clone()
}

func (g *ConcurrentGrid[T]) ensureInitLocked() {
	if g.core == nil {
		g.core = NewGrid[T]()
	}
}
