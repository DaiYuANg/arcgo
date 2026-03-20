package mapping

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShardedConcurrentMap_Basic(t *testing.T) {
	t.Parallel()

	m := NewShardedConcurrentMap[int, int](16, HashInt)
	m.Set(1, 10)
	m.Set(2, 20)

	v, ok := m.Get(1)
	require.True(t, ok)
	require.Equal(t, 10, v)

	v, ok = m.GetOrStore(3, 30)
	require.False(t, ok)
	require.Equal(t, 30, v)

	ok = m.Delete(2)
	require.True(t, ok)
	_, ok = m.Get(2)
	require.False(t, ok)
}

func TestShardedConcurrentMap_Parallel(t *testing.T) {
	t.Parallel()

	m := NewShardedConcurrentMap[int, int](32, HashInt)
	const workers = 20
	const each = 200

	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		w := w
		go func() {
			defer wg.Done()
			for i := 0; i < each; i++ {
				m.Set(w*each+i, i)
			}
		}()
	}
	wg.Wait()
	require.Equal(t, workers*each, m.Len())
}

func TestShardedConcurrentMap_StringKeys(t *testing.T) {
	t.Parallel()

	m := NewShardedConcurrentMap[string, int](8, HashString)
	m.Set("a", 1)
	m.Set("b", 2)
	v, ok := m.Get("a")
	require.True(t, ok)
	require.Equal(t, 1, v)
}
