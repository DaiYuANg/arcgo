package list

// Grid is an ordered two-dimensional container backed by row lists.
// Zero value is ready to use.
type Grid[T any] struct {
	rows List[*List[T]]
	size int
}

// NewGrid creates a grid and copies optional rows.
func NewGrid[T any](rows ...[]T) *Grid[T] {
	return NewGridWithCapacity(len(rows), rows...)
}

// NewGridWithCapacity creates a grid with preallocated row capacity and optional rows.
func NewGridWithCapacity[T any](capacity int, rows ...[]T) *Grid[T] {
	if capacity < len(rows) {
		capacity = len(rows)
	}

	g := &Grid[T]{}
	if capacity > 0 {
		g.rows = *NewListWithCapacity[*List[T]](capacity)
	}
	g.AddRows(rows...)
	return g
}

// AddRow appends one row.
func (g *Grid[T]) AddRow(items ...T) {
	if g == nil {
		return
	}
	g.appendOwnedRow(NewList(items...))
}

// AddRowList appends one row from a list copy.
func (g *Grid[T]) AddRowList(items *List[T]) {
	if g == nil {
		return
	}
	g.appendOwnedRow(cloneGridRowList(items))
}

// AddRows appends multiple rows.
func (g *Grid[T]) AddRows(rows ...[]T) {
	if g == nil || len(rows) == 0 {
		return
	}
	for _, row := range rows {
		g.appendOwnedRow(NewList(row...))
	}
}

// AddRowsList appends multiple rows from copied row lists.
func (g *Grid[T]) AddRowsList(rows *List[*List[T]]) {
	if g == nil || rows == nil || rows.Len() == 0 {
		return
	}
	rows.Range(func(_ int, row *List[T]) bool {
		g.appendOwnedRow(cloneGridRowList(row))
		return true
	})
}

// Merge appends copied rows from other.
func (g *Grid[T]) Merge(other *Grid[T]) *Grid[T] {
	if g == nil {
		return nil
	}
	if other == nil || other.rows.Len() == 0 {
		return g
	}
	other.rows.Range(func(_ int, row *List[T]) bool {
		g.appendOwnedRow(cloneGridRowList(row))
		return true
	})
	return g
}

// Get returns value at row and column indexes.
func (g *Grid[T]) Get(rowIndex, columnIndex int) (T, bool) {
	var zero T
	if g == nil {
		return zero, false
	}
	row, ok := g.rows.Get(rowIndex)
	if !ok || row == nil {
		return zero, false
	}
	return row.Get(columnIndex)
}

// GetRow returns a copied row at index.
func (g *Grid[T]) GetRow(index int) ([]T, bool) {
	if g == nil {
		return nil, false
	}
	row, ok := g.rows.Get(index)
	if !ok || row == nil {
		return nil, false
	}
	return row.Values(), true
}

// GetRowList returns a copied row list at index.
func (g *Grid[T]) GetRowList(index int) (*List[T], bool) {
	if g == nil {
		return nil, false
	}
	row, ok := g.rows.Get(index)
	if !ok {
		return nil, false
	}
	return cloneGridRowList(row), true
}

// Set replaces value at row and column indexes.
func (g *Grid[T]) Set(rowIndex, columnIndex int, item T) bool {
	if g == nil {
		return false
	}
	row, ok := g.rows.Get(rowIndex)
	if !ok || row == nil {
		return false
	}
	return row.Set(columnIndex, item)
}

// SetRow replaces one row.
func (g *Grid[T]) SetRow(index int, items ...T) bool {
	if g == nil {
		return false
	}
	current, ok := g.rows.Get(index)
	if !ok {
		return false
	}
	next := NewList(items...)
	currentLen := 0
	if current != nil {
		currentLen = current.Len()
	}
	g.size += next.Len() - currentLen
	return g.rows.Set(index, next)
}

// RemoveRow removes and returns one copied row.
func (g *Grid[T]) RemoveRow(index int) ([]T, bool) {
	if g == nil {
		return nil, false
	}
	removed, ok := g.rows.RemoveAt(index)
	if !ok {
		return nil, false
	}
	if removed == nil {
		return nil, true
	}
	g.size -= removed.Len()
	return removed.Values(), true
}

// RowCount returns total row count.
func (g *Grid[T]) RowCount() int {
	if g == nil {
		return 0
	}
	return g.rows.Len()
}

// Len returns total cell count across all rows.
func (g *Grid[T]) Len() int {
	if g == nil {
		return 0
	}
	return g.size
}

// IsEmpty reports whether grid has no rows.
func (g *Grid[T]) IsEmpty() bool {
	return g.RowCount() == 0
}

// Clear removes all rows.
func (g *Grid[T]) Clear() {
	if g == nil {
		return
	}
	g.rows.Clear()
	g.size = 0
}

// Values returns copied rows.
func (g *Grid[T]) Values() [][]T {
	if g == nil || g.rows.Len() == 0 {
		return nil
	}
	rows := make([][]T, 0, g.rows.Len())
	g.rows.Range(func(_ int, row *List[T]) bool {
		rows = append(rows, gridRowValues(row))
		return true
	})
	return rows
}

// Range iterates rows from top to bottom until fn returns false.
func (g *Grid[T]) Range(fn func(index int, row []T) bool) {
	if g == nil || fn == nil {
		return
	}
	g.rows.Range(func(index int, row *List[T]) bool {
		return fn(index, gridRowItems(row))
	})
}

// Clone returns a deep copy.
func (g *Grid[T]) Clone() *Grid[T] {
	if g == nil || g.rows.Len() == 0 {
		return &Grid[T]{}
	}
	rows := NewListWithCapacity[*List[T]](g.rows.Len())
	g.rows.Range(func(_ int, row *List[T]) bool {
		rows.Add(cloneGridRowList(row))
		return true
	})
	return &Grid[T]{
		rows: *rows,
		size: g.size,
	}
}

func (g *Grid[T]) appendOwnedRow(row *List[T]) {
	if row == nil {
		row = NewList[T]()
	}
	g.rows.Add(row)
	g.size += row.Len()
}

func newGridFromRowLists[T any](rows *List[*List[T]]) *Grid[T] {
	if rows == nil || rows.Len() == 0 {
		return &Grid[T]{}
	}
	cloned := NewListWithCapacity[*List[T]](rows.Len())
	size := 0
	rows.Range(func(_ int, row *List[T]) bool {
		rowCopy := cloneGridRowList(row)
		cloned.Add(rowCopy)
		size += rowCopy.Len()
		return true
	})
	return &Grid[T]{
		rows: *cloned,
		size: size,
	}
}

func cloneGridRowList[T any](row *List[T]) *List[T] {
	if row == nil {
		return NewList[T]()
	}
	return row.Clone()
}

func gridRowItems[T any](row *List[T]) []T {
	if row == nil {
		return nil
	}
	return row.items
}

func gridRowValues[T any](row *List[T]) []T {
	if row == nil {
		return nil
	}
	return row.Values()
}
