package mapping_test

import (
	"sync"
	"testing"

	mapping "github.com/DaiYuANg/arcgo/collectionx/mapping"
	"github.com/stretchr/testify/require"
)

func TestConcurrentTable_ParallelPut(t *testing.T) {
	t.Parallel()

	var tb mapping.ConcurrentTable[int, int, int]

	const workers = 12
	const each = 80

	var wg sync.WaitGroup
	wg.Add(workers)

	for worker := range workers {
		go func() {
			defer wg.Done()
			for i := range each {
				tb.Put(worker, i, i)
			}
		}()
	}

	wg.Wait()
	require.Equal(t, workers, tb.RowCount())
	require.Equal(t, workers*each, tb.Len())
}

func TestConcurrentTable_OptionDeleteAndSnapshot(t *testing.T) {
	t.Parallel()

	var tb mapping.ConcurrentTable[string, string, int]
	tb.Put("u1", "score", 10)
	tb.Put("u1", "level", 2)
	tb.Put("u2", "score", 20)

	opt := tb.GetOption("u1", "score")
	require.True(t, opt.IsPresent())
	value, ok := opt.Get()
	require.True(t, ok)
	require.Equal(t, 10, value)

	removed := tb.DeleteColumn("score")
	require.Equal(t, 2, removed)

	snapshot := tb.Snapshot()
	tb.Put("u3", "score", 99)
	_, ok = snapshot.Get("u3", "score")
	require.False(t, ok)
}

func TestConcurrentTable_FluentOps(t *testing.T) {
	t.Parallel()

	var tb mapping.ConcurrentTable[string, string, int]
	tb.Put("r1", "c1", 1)
	tb.Put("r1", "c2", 2)
	tb.Put("r2", "c1", 3)
	tb.Put("r2", "c2", 4)

	filtered := tb.
		RejectRows(func(rowKey string, _ map[string]int) bool { return rowKey == "r1" }).
		RejectCells(func(_ string, columnKey string, _ int) bool { return columnKey == "c1" })

	require.False(t, filtered.Has("r1", "c2"))
	require.Equal(t, map[string]int{"c2": 4}, filtered.Row("r2"))

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
		FirstCellWhere(func(_ string, _ string, value int) bool { return value > 3 })

	require.True(t, ok)
	require.Equal(t, "r2", foundRow)
	require.Equal(t, "c2", foundColumn)
	require.Equal(t, 4, foundValue)
	require.Equal(t, 8, visited.Len())
	require.True(t, tb.AnyCellMatch(func(_ string, columnKey string, _ int) bool { return columnKey == "c1" }))
	require.True(t, tb.AllCellsMatch(func(_ string, _ string, value int) bool { return value > 0 }))
}

func TestConcurrentTable_JSONCacheReturnsDefensiveCopy(t *testing.T) {
	t.Parallel()

	tb := mapping.NewConcurrentTable[string, string, int]()
	tb.Put("r1", "c1", 1)

	data, err := tb.ToJSON()
	require.NoError(t, err)
	require.Equal(t, `{"r1":{"c1":1}}`, string(data))
	require.Equal(t, `{"r1":{"c1":1}}`, tb.String())

	data[0] = '['
	fresh, err := tb.ToJSON()
	require.NoError(t, err)
	require.Equal(t, `{"r1":{"c1":1}}`, string(fresh))

	tb.Put("r1", "c2", 2)
	require.Contains(t, tb.String(), `"c1":1`)
	require.Contains(t, tb.String(), `"c2":2`)
}
