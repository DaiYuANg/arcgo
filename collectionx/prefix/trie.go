package prefix

import (
	"slices"

	collectionmapping "github.com/DaiYuANg/archgo/collectionx/mapping"
	"github.com/samber/lo"
	"github.com/samber/mo"
)

type trieNode[V any] struct {
	children collectionmapping.Map[rune, *trieNode[V]]
	hasValue bool
	value    V
}

// Trie is a prefix tree for string keys.
// Zero value is ready to use.
type Trie[V any] struct {
	root *trieNode[V]
	size int
}

type keyValue[V any] struct {
	key   string
	value V
}

// NewTrie creates an empty trie.
func NewTrie[V any]() *Trie[V] {
	return &Trie[V]{}
}

// NewPrefixMap creates an empty prefix map.
// PrefixMap shares the same implementation as Trie.
func NewPrefixMap[V any]() *Trie[V] {
	return NewTrie[V]()
}

// Put stores key -> value.
// Returns true when inserted as a new key, false when updated existing key.
func (t *Trie[V]) Put(key string, value V) bool {
	if t == nil {
		return false
	}
	t.ensureRoot()

	node := t.root
	for _, ch := range key {
		next, ok := node.children.Get(ch)
		if !ok {
			next = &trieNode[V]{}
			node.children.Set(ch, next)
		}
		node = next
	}

	isNew := !node.hasValue
	node.value = value
	node.hasValue = true
	if isNew {
		t.size++
	}
	return isNew
}

// Get returns value by exact key.
func (t *Trie[V]) Get(key string) (V, bool) {
	var zero V
	if t == nil || t.root == nil {
		return zero, false
	}
	node, ok := t.findNode(key)
	if !ok || !node.hasValue {
		return zero, false
	}
	return node.value, true
}

// GetOption returns value by exact key as mo.Option.
func (t *Trie[V]) GetOption(key string) mo.Option[V] {
	value, ok := t.Get(key)
	if !ok {
		return mo.None[V]()
	}
	return mo.Some(value)
}

// Has reports whether exact key exists.
func (t *Trie[V]) Has(key string) bool {
	_, ok := t.Get(key)
	return ok
}

// HasPrefix reports whether prefix exists in trie paths.
func (t *Trie[V]) HasPrefix(prefix string) bool {
	if t == nil || t.root == nil {
		return false
	}
	_, ok := t.findNode(prefix)
	return ok
}

// Delete removes key and returns whether key existed.
func (t *Trie[V]) Delete(key string) bool {
	if t == nil || t.root == nil {
		return false
	}
	runes := []rune(key)
	removed := t.deleteRec(t.root, runes, 0)
	if removed {
		t.size--
	}
	return removed
}

// Len returns stored key count.
func (t *Trie[V]) Len() int {
	if t == nil {
		return 0
	}
	return t.size
}

// IsEmpty reports whether trie has no keys.
func (t *Trie[V]) IsEmpty() bool {
	return t.Len() == 0
}

// Clear removes all keys.
func (t *Trie[V]) Clear() {
	if t == nil {
		return
	}
	t.root = nil
	t.size = 0
}

// KeysWithPrefix returns all keys that start with prefix.
func (t *Trie[V]) KeysWithPrefix(prefix string) []string {
	pairs := t.pairsWithPrefix(prefix)
	if len(pairs) == 0 {
		return nil
	}
	return lo.Map(pairs, func(item keyValue[V], _ int) string {
		return item.key
	})
}

// ValuesWithPrefix returns all values under prefix.
func (t *Trie[V]) ValuesWithPrefix(prefix string) []V {
	pairs := t.pairsWithPrefix(prefix)
	if len(pairs) == 0 {
		return nil
	}
	return lo.Map(pairs, func(item keyValue[V], _ int) V {
		return item.value
	})
}

// RangePrefix iterates keys with prefix in lexicographic key order until fn returns false.
func (t *Trie[V]) RangePrefix(prefix string, fn func(key string, value V) bool) {
	if fn == nil {
		return
	}
	for _, item := range t.pairsWithPrefix(prefix) {
		if !fn(item.key, item.value) {
			return
		}
	}
}

func (t *Trie[V]) ensureRoot() {
	if t.root == nil {
		t.root = &trieNode[V]{}
	}
}

func (t *Trie[V]) findNode(key string) (*trieNode[V], bool) {
	node := t.root
	for _, ch := range key {
		next, ok := node.children.Get(ch)
		if !ok {
			return nil, false
		}
		node = next
	}
	return node, true
}

func (t *Trie[V]) deleteRec(node *trieNode[V], runes []rune, depth int) bool {
	if node == nil {
		return false
	}
	if depth == len(runes) {
		if !node.hasValue {
			return false
		}
		node.hasValue = false
		var zero V
		node.value = zero
		return true
	}

	ch := runes[depth]
	child, ok := node.children.Get(ch)
	if !ok {
		return false
	}
	removed := t.deleteRec(child, runes, depth+1)
	if !removed {
		return false
	}

	if !child.hasValue && child.children.Len() == 0 {
		node.children.Delete(ch)
	}
	return true
}

func (t *Trie[V]) collectPairs(node *trieNode[V], path *[]rune, out *[]keyValue[V]) {
	if node == nil {
		return
	}
	if node.hasValue {
		*out = append(*out, keyValue[V]{
			key:   string(*path),
			value: node.value,
		})
	}

	if node.children.Len() == 0 {
		return
	}

	keys := node.children.Keys()
	slices.Sort(keys)
	for _, ch := range keys {
		*path = append(*path, ch)
		child, _ := node.children.Get(ch)
		t.collectPairs(child, path, out)
		*path = (*path)[:len(*path)-1]
	}
}

func (t *Trie[V]) pairsWithPrefix(prefix string) []keyValue[V] {
	if t == nil || t.root == nil {
		return nil
	}
	startNode, ok := t.findNode(prefix)
	if !ok {
		return nil
	}

	out := make([]keyValue[V], 0)
	t.collectPairs(startNode, new([]rune(prefix)), &out)
	return out
}
