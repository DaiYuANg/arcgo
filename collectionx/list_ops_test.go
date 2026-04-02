package collectionx_test

import (
	"fmt"
	"testing"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListOps(t *testing.T) {
	source := collectionx.NewList(1, 2, 3, 4)

	mapped := collectionx.MapList(source, func(index int, item int) string {
		return fmt.Sprintf("%d:%d", index, item)
	})
	assert.Equal(t, []string{"0:1", "1:2", "2:3", "3:4"}, mapped.Values())

	filtered := collectionx.FilterList(source, func(_ int, item int) bool {
		return item%2 == 0
	})
	assert.Equal(t, []int{2, 4}, filtered.Values())

	rejected := collectionx.RejectList(source, func(_ int, item int) bool {
		return item%2 == 0
	})
	assert.Equal(t, []int{1, 3}, rejected.Values())

	filterMapped := collectionx.FilterMapList(source, func(_ int, item int) (string, bool) {
		if item%2 != 0 {
			return "", false
		}
		return fmt.Sprintf("v%d", item), true
	})
	assert.Equal(t, []string{"v2", "v4"}, filterMapped.Values())

	flatMapped := collectionx.FlatMapList(source, func(_ int, item int) []string {
		return []string{fmt.Sprintf("x%d", item), fmt.Sprintf("y%d", item)}
	})
	assert.Equal(t, []string{"x1", "y1", "x2", "y2", "x3", "y3", "x4", "y4"}, flatMapped.Values())

	found, ok := collectionx.FindList(source, func(_ int, item int) bool {
		return item > 2
	})
	assert.True(t, ok)
	assert.Equal(t, 3, found)

	reduced := collectionx.ReduceList(source, 0, func(acc int, _ int, item int) int {
		return acc + item
	})
	assert.Equal(t, 10, reduced)

	reducedErr, err := collectionx.ReduceErrList(source, 0, func(acc int, _ int, item int) (int, error) {
		if item == 4 {
			return 0, fmt.Errorf("stop at %d", item)
		}
		return acc + item, nil
	})
	require.Error(t, err)
	assert.Equal(t, 6, reducedErr)
	assert.ErrorContains(t, err, "stop at 4")

	grouped := collectionx.GroupByList(source, func(_ int, item int) string {
		if item%2 == 0 {
			return "even"
		}
		return "odd"
	})
	assert.Equal(t, []int{1, 3}, grouped.Get("odd"))
	assert.Equal(t, []int{2, 4}, grouped.Get("even"))

	associated := collectionx.AssociateList(source, func(index int, item int) (string, string) {
		return fmt.Sprintf("k%d", index), fmt.Sprintf("v%d", item)
	})
	value, ok := associated.Get("k2")
	require.True(t, ok)
	assert.Equal(t, "v3", value)
}

func TestListChainMethods(t *testing.T) {
	values := collectionx.NewList(1, 2, 3, 4).
		Where(func(_ int, item int) bool { return item >= 2 }).
		Reject(func(_ int, item int) bool { return item == 3 }).
		Take(2)
	assert.Equal(t, []int{2, 4}, values.Values())

	visited := collectionx.NewList[string]()
	first, ok := collectionx.NewList(1, 2, 3, 4).
		Each(func(index int, item int) {
			visited.Add(fmt.Sprintf("%d:%d", index, item))
		}).
		FirstWhere(func(_ int, item int) bool { return item > 2 }).Get()
	require.True(t, ok)
	assert.Equal(t, 3, first)
	assert.Equal(t, []string{"0:1", "1:2", "2:3", "3:4"}, visited.Values())

	assert.True(t, collectionx.NewList(2, 4, 6).AllMatch(func(_ int, item int) bool { return item%2 == 0 }))
	assert.True(t, collectionx.NewList(1, 2, 3).AnyMatch(func(_ int, item int) bool { return item == 2 }))
}
