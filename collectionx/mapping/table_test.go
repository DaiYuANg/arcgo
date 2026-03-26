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
