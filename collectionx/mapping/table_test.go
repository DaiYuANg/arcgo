package mapping_test

import (
	"testing"

	mapping "github.com/DaiYuANg/arcgo/collectionx/mapping"
	"github.com/stretchr/testify/require"
)

func TestTable_BasicOps(t *testing.T) {
	t.Parallel()

	var tb mapping.Table[string, string, int]

	tb.Put("u1", "score", 100)
	tb.Put("u1", "level", 8)
	tb.Put("u2", "score", 90)

	value, ok := tb.Get("u1", "score")
	require.True(t, ok)
	require.Equal(t, 100, value)
	require.Equal(t, 3, tb.Len())
	require.Equal(t, 2, tb.RowCount())

	require.True(t, tb.Has("u2", "score"))
	require.True(t, tb.Delete("u2", "score"))
	require.False(t, tb.Has("u2", "score"))
	require.Equal(t, 2, tb.Len())
}

func TestTable_RowColumnAndOption(t *testing.T) {
	t.Parallel()

	tb := mapping.NewTable[string, string, int]()
	tb.Put("r1", "c1", 1)
	tb.Put("r1", "c2", 2)
	tb.Put("r2", "c1", 3)

	row := tb.Row("r1")
	require.Equal(t, map[string]int{"c1": 1, "c2": 2}, row)
	row["c1"] = 99
	require.Equal(t, 1, tb.Row("r1")["c1"])

	col := tb.Column("c1")
	require.Equal(t, map[string]int{"r1": 1, "r2": 3}, col)

	opt := tb.GetOption("r2", "c1")
	require.True(t, opt.IsPresent())
	value, ok := opt.Get()
	require.True(t, ok)
	require.Equal(t, 3, value)

	require.True(t, tb.GetOption("missing", "c1").IsAbsent())
}

func TestTable_DeleteColumn(t *testing.T) {
	t.Parallel()

	tb := mapping.NewTable[string, string, int]()
	tb.Put("r1", "c1", 1)
	tb.Put("r1", "c2", 2)
	tb.Put("r2", "c2", 3)

	removed := tb.DeleteColumn("c2")
	require.Equal(t, 2, removed)
	require.Equal(t, 1, tb.Len())
	require.Equal(t, []string{"r1"}, tb.RowKeys())
}

func TestTable_FluentOps(t *testing.T) {
	t.Parallel()

	tb := mapping.NewTable[string, string, int]()
	tb.Put("r1", "c1", 1)
	tb.Put("r1", "c2", 2)
	tb.Put("r2", "c1", 3)
	tb.Put("r2", "c2", 4)

	filtered := tb.
		WhereRows(func(rowKey string, _ map[string]int) bool { return rowKey != "r2" }).
		WhereCells(func(_ string, columnKey string, value int) bool { return columnKey == "c2" || value >= 2 })

	require.Equal(t, map[string]int{"c2": 2}, filtered.Row("r1"))
	require.False(t, filtered.Has("r2", "c1"))

	visited := mapping.NewTable[string, string, int]()
	foundRow, foundColumn, foundValue, ok := tb.
		EachRow(func(rowKey string, row map[string]int) {
			for columnKey, value := range row {
				visited.Put(rowKey, columnKey, value)
			}
		}).
		EachCell(func(rowKey string, columnKey string, value int) {
			visited.Put(rowKey, columnKey+"x", value*10)
		}).
		FirstCellWhere(func(_ string, columnKey string, _ int) bool { return columnKey == "c2" })

	require.True(t, ok)
	require.True(t, foundRow == "r1" || foundRow == "r2")
	require.Equal(t, "c2", foundColumn)
	require.True(t, foundValue == 2 || foundValue == 4)
	require.Equal(t, 8, visited.Len())
	require.True(t, tb.AnyCellMatch(func(_ string, _ string, value int) bool { return value == 4 }))
	require.True(t, tb.AllCellsMatch(func(_ string, _ string, value int) bool { return value > 0 }))
	require.False(t, tb.AllCellsMatch(func(_ string, columnKey string, _ int) bool { return columnKey == "c1" }))
}
