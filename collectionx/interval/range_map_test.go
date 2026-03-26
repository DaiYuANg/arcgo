package interval_test

import (
	"testing"

	"github.com/DaiYuANg/arcgo/collectionx/interval"
	"github.com/stretchr/testify/require"
)

func TestRangeMap_PutOverride(t *testing.T) {
	t.Parallel()

	m := interval.NewRangeMap[int, string]()
	require.True(t, m.Put(0, 10, "A"))
	require.True(t, m.Put(3, 6, "B"))

	entries := m.Entries()
	require.Equal(
		t,
		[]interval.RangeEntry[int, string]{
			{Range: interval.Range[int]{Start: 0, End: 3}, Value: "A"},
			{Range: interval.Range[int]{Start: 3, End: 6}, Value: "B"},
			{Range: interval.Range[int]{Start: 6, End: 10}, Value: "A"},
		},
		entries,
	)

	value, ok := m.Get(4)
	require.True(t, ok)
	require.Equal(t, "B", value)
}

func TestRangeMap_DeleteRangeAndOption(t *testing.T) {
	t.Parallel()

	m := interval.NewRangeMap[int, int]()
	m.Put(0, 5, 1)
	m.Put(5, 10, 2)
	require.True(t, m.DeleteRange(2, 8))

	require.Equal(
		t,
		[]interval.RangeEntry[int, int]{
			{Range: interval.Range[int]{Start: 0, End: 2}, Value: 1},
			{Range: interval.Range[int]{Start: 8, End: 10}, Value: 2},
		},
		m.Entries(),
	)

	require.True(t, m.GetOption(4).IsAbsent())
	require.True(t, m.GetOption(9).IsPresent())
}

func TestRangeMap_PutKeepsEntriesSorted(t *testing.T) {
	t.Parallel()

	m := interval.NewRangeMap[int, string]()
	require.True(t, m.Put(10, 20, "A"))
	require.True(t, m.Put(0, 5, "B"))
	require.True(t, m.Put(5, 10, "C"))
	require.True(t, m.Put(3, 12, "D"))

	require.Equal(
		t,
		[]interval.RangeEntry[int, string]{
			{Range: interval.Range[int]{Start: 0, End: 3}, Value: "B"},
			{Range: interval.Range[int]{Start: 3, End: 12}, Value: "D"},
			{Range: interval.Range[int]{Start: 12, End: 20}, Value: "A"},
		},
		m.Entries(),
	)
}
