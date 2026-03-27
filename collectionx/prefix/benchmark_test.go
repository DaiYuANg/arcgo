package prefix_test

import (
	"strconv"
	"testing"

	prefix "github.com/DaiYuANg/arcgo/collectionx/prefix"
)

const benchTrieKeySpace = 1 << 12

func makeBenchTrieKeys() []string {
	keys := make([]string, benchTrieKeySpace)
	for i := range benchTrieKeySpace {
		keys[i] = "user/" + strconv.Itoa(i>>8) + "/profile/" + strconv.Itoa(i)
	}
	return keys
}

func BenchmarkTriePut(b *testing.B) {
	t := prefix.NewTrie[int]()
	keys := makeBenchTrieKeys()
	mask := benchTrieKeySpace - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		t.Put(keys[i&mask], i)
	}
}

func BenchmarkTrieGet(b *testing.B) {
	t := prefix.NewTrie[int]()
	keys := makeBenchTrieKeys()
	for i, key := range keys {
		t.Put(key, i)
	}

	mask := benchTrieKeySpace - 1
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		_, _ = t.Get(keys[i&mask])
	}
}

func BenchmarkTrieDeleteReinsert(b *testing.B) {
	t := prefix.NewTrie[int]()
	keys := makeBenchTrieKeys()
	for i, key := range keys {
		t.Put(key, i)
	}
	mask := benchTrieKeySpace - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		key := keys[i&mask]
		t.Delete(key)
		t.Put(key, i)
	}
}

func BenchmarkTrieKeysWithPrefix(b *testing.B) {
	t := prefix.NewTrie[int]()
	keys := makeBenchTrieKeys()
	for i, key := range keys {
		t.Put(key, i)
	}
	prefixKey := "user/7/profile/"

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = t.KeysWithPrefix(prefixKey)
	}
}

func BenchmarkTrieValuesWithPrefix(b *testing.B) {
	t := prefix.NewTrie[int]()
	keys := makeBenchTrieKeys()
	for i, key := range keys {
		t.Put(key, i)
	}
	prefixKey := "user/7/profile/"

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = t.ValuesWithPrefix(prefixKey)
	}
}

func BenchmarkTrieRangePrefix(b *testing.B) {
	t := prefix.NewTrie[int]()
	keys := makeBenchTrieKeys()
	for i, key := range keys {
		t.Put(key, i)
	}
	prefixKey := "user/7/profile/"

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		t.RangePrefix(prefixKey, func(key string, value int) bool {
			_ = value
			return true
		})
	}
}

func BenchmarkTrieHas(b *testing.B) {
	t := prefix.NewTrie[int]()
	keys := makeBenchTrieKeys()
	for i, key := range keys {
		t.Put(key, i)
	}
	mask := benchTrieKeySpace - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		_ = t.Has(keys[i&mask])
	}
}

func BenchmarkTrieHasPrefix(b *testing.B) {
	t := prefix.NewTrie[int]()
	keys := makeBenchTrieKeys()
	for i, key := range keys {
		t.Put(key, i)
	}
	prefixKey := "user/7/profile/"

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = t.HasPrefix(prefixKey)
	}
}
