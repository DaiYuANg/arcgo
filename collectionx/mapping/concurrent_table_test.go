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
