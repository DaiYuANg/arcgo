package collectionx

import (
	"github.com/DaiYuANg/archgo/collectionx/prefix"
	"github.com/samber/mo"
)

type trieWritable[V any] interface {
	Put(key string, value V) bool
	Delete(key string) bool
	clearable
}

type trieReadable[V any] interface {
	Get(key string) (V, bool)
	GetOption(key string) mo.Option[V]
	Has(key string) bool
	HasPrefix(prefix string) bool
	sized
	KeysWithPrefix(prefix string) []string
	ValuesWithPrefix(prefix string) []V
	RangePrefix(prefix string, fn func(key string, value V) bool)
}

type Trie[V any] interface {
	trieWritable[V]
	trieReadable[V]
	jsonStringer
}

func NewTrie[V any]() Trie[V] {
	return prefix.NewTrie[V]()
}

func NewPrefixMap[V any]() Trie[V] {
	return prefix.NewPrefixMap[V]()
}
