package mapping

import (
	"hash/maphash"
	"strconv"
	"sync"

	"github.com/samber/mo"
)

// HashStringSeed is a seed for HashString; create once per map for consistency.
var hashStringSeed = maphash.MakeSeed()

const defaultShardCount = 32

// ShardedConcurrentMap is a goroutine-safe map with per-shard locks
// to reduce contention under concurrent access.
// Requires a hash function for key distribution.
type ShardedConcurrentMap[K comparable, V any] struct {
	shards []struct {
		mu   sync.RWMutex
		core *Map[K, V]
	}
	hash func(K) uint64
	mask uint64
}

// NewShardedConcurrentMap creates a sharded concurrent map.
// shardCount should be a power of 2 (e.g. 16, 32, 64); otherwise it is rounded up.
// hash distributes keys across shards; use HashInt, HashString, etc. for common types.
func NewShardedConcurrentMap[K comparable, V any](shardCount int, hash func(K) uint64) *ShardedConcurrentMap[K, V] {
	if shardCount <= 0 {
		shardCount = defaultShardCount
	}
	// round up to power of 2
	n := 1
	for n < shardCount {
		n *= 2
	}
	shardCount = n

	shards := make([]struct {
		mu   sync.RWMutex
		core *Map[K, V]
	}, shardCount)
	for i := range shards {
		shards[i].core = NewMap[K, V]()
	}
	return &ShardedConcurrentMap[K, V]{
		shards: shards,
		hash:   hash,
		mask:   shardMask(shardCount),
	}
}

// HashInt returns a hash for int keys.
func HashInt(k int) uint64 {
	return hashSignedInt64(int64(k))
}

// HashInt64 returns a hash for int64 keys.
func HashInt64(k int64) uint64 {
	return hashSignedInt64(k)
}

// HashUint64 returns a hash for uint64 keys.
func HashUint64(k uint64) uint64 {
	return k
}

// HashString returns a hash for string keys.
func HashString(k string) uint64 {
	var h maphash.Hash
	h.SetSeed(hashStringSeed)
	mustWriteHashString(&h, k)
	return h.Sum64()
}

func (m *ShardedConcurrentMap[K, V]) shard(key K) *struct {
	mu   sync.RWMutex
	core *Map[K, V]
} {
	if m == nil || len(m.shards) == 0 {
		return nil
	}
	i := m.hash(key) & m.mask
	return &m.shards[i]
}

func shardMask(shardCount int) uint64 {
	var mask uint64
	for range shardCount - 1 {
		mask++
	}
	return mask
}

func hashSignedInt64(value int64) uint64 {
	var h maphash.Hash
	h.SetSeed(hashStringSeed)
	buffer := strconv.AppendInt(nil, value, 10)
	mustWriteHashBytes(&h, buffer)
	return h.Sum64()
}

func mustWriteHashString(h *maphash.Hash, value string) {
	written, err := h.WriteString(value)
	if err != nil || written != len(value) {
		panic("maphash.WriteString failed")
	}
}

func mustWriteHashBytes(h *maphash.Hash, value []byte) {
	written, err := h.Write(value)
	if err != nil || written != len(value) {
		panic("maphash.Write failed")
	}
}

// Set puts a key-value pair.
func (m *ShardedConcurrentMap[K, V]) Set(key K, value V) {
	if m == nil {
		return
	}
	s := m.shard(key)
	s.mu.Lock()
	s.core.Set(key, value)
	s.mu.Unlock()
}

// SetAll copies all entries from source into the map.
func (m *ShardedConcurrentMap[K, V]) SetAll(source map[K]V) {
	if m == nil || len(source) == 0 {
		return
	}
	for k, v := range source {
		m.Set(k, v)
	}
}

// Get returns the value for key.
func (m *ShardedConcurrentMap[K, V]) Get(key K) (V, bool) {
	var zero V
	if m == nil {
		return zero, false
	}
	s := m.shard(key)
	s.mu.RLock()
	v, ok := s.core.Get(key)
	s.mu.RUnlock()
	return v, ok
}

// GetOption returns value for key as mo.Option.
func (m *ShardedConcurrentMap[K, V]) GetOption(key K) mo.Option[V] {
	v, ok := m.Get(key)
	if !ok {
		return mo.None[V]()
	}
	return mo.Some(v)
}

// GetOrDefault returns value for key or fallback when key does not exist.
func (m *ShardedConcurrentMap[K, V]) GetOrDefault(key K, fallback V) V {
	v, ok := m.Get(key)
	if !ok {
		return fallback
	}
	return v
}

// GetOrStore returns existing value if key exists; otherwise stores and returns value.
func (m *ShardedConcurrentMap[K, V]) GetOrStore(key K, value V) (V, bool) {
	if m == nil {
		return value, false
	}
	s := m.shard(key)
	s.mu.Lock()
	if v, ok := s.core.Get(key); ok {
		s.mu.Unlock()
		return v, true
	}
	s.core.Set(key, value)
	s.mu.Unlock()
	return value, false
}

// Delete removes key and reports whether it existed.
func (m *ShardedConcurrentMap[K, V]) Delete(key K) bool {
	if m == nil {
		return false
	}
	s := m.shard(key)
	s.mu.Lock()
	ok := s.core.Delete(key)
	s.mu.Unlock()
	return ok
}

// LoadAndDelete removes key and returns previous value.
func (m *ShardedConcurrentMap[K, V]) LoadAndDelete(key K) (V, bool) {
	var zero V
	if m == nil {
		return zero, false
	}
	s := m.shard(key)
	s.mu.Lock()
	v, ok := s.core.Get(key)
	if ok {
		s.core.Delete(key)
	}
	s.mu.Unlock()
	return v, ok
}

// LoadAndDeleteOption removes key and returns previous value as mo.Option.
func (m *ShardedConcurrentMap[K, V]) LoadAndDeleteOption(key K) mo.Option[V] {
	v, ok := m.LoadAndDelete(key)
	if !ok {
		return mo.None[V]()
	}
	return mo.Some(v)
}

// Len returns total entry count (requires locking all shards).
func (m *ShardedConcurrentMap[K, V]) Len() int {
	if m == nil {
		return 0
	}
	n := 0
	for i := range m.shards {
		m.shards[i].mu.RLock()
		n += m.shards[i].core.Len()
		m.shards[i].mu.RUnlock()
	}
	return n
}

// IsEmpty reports whether map has no entries.
func (m *ShardedConcurrentMap[K, V]) IsEmpty() bool {
	return m.Len() == 0
}

// Clear removes all entries from all shards.
func (m *ShardedConcurrentMap[K, V]) Clear() {
	if m == nil {
		return
	}
	for i := range m.shards {
		m.shards[i].mu.Lock()
		m.shards[i].core.Clear()
		m.shards[i].mu.Unlock()
	}
}

// Keys returns a snapshot of all keys (locks all shards).
func (m *ShardedConcurrentMap[K, V]) Keys() []K {
	if m == nil {
		return nil
	}
	var out []K
	for i := range m.shards {
		m.shards[i].mu.RLock()
		out = append(out, m.shards[i].core.Keys()...)
		m.shards[i].mu.RUnlock()
	}
	return out
}

// Values returns a snapshot of all values (locks all shards).
func (m *ShardedConcurrentMap[K, V]) Values() []V {
	if m == nil {
		return nil
	}
	var out []V
	for i := range m.shards {
		m.shards[i].mu.RLock()
		out = append(out, m.shards[i].core.Values()...)
		m.shards[i].mu.RUnlock()
	}
	return out
}

// All returns a copied built-in map (locks all shards).
func (m *ShardedConcurrentMap[K, V]) All() map[K]V {
	if m == nil {
		return map[K]V{}
	}
	out := make(map[K]V)
	for i := range m.shards {
		m.shards[i].mu.RLock()
		m.shards[i].core.Range(func(k K, v V) bool {
			out[k] = v
			return true
		})
		m.shards[i].mu.RUnlock()
	}
	return out
}

// Range iterates a stable snapshot until fn returns false.
func (m *ShardedConcurrentMap[K, V]) Range(fn func(key K, value V) bool) {
	if m == nil || fn == nil {
		return
	}
	for k, v := range m.All() {
		if !fn(k, v) {
			return
		}
	}
}
