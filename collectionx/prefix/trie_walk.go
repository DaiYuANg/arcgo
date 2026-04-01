package prefix

import "github.com/samber/lo"

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

	lo.ForEach(node.childKeys, func(ch rune, _ int) {
		*path = append(*path, ch)
		child, _ := node.children.Get(ch)
		t.collectPairs(child, path, out)
		*path = (*path)[:len(*path)-1]
	})
}

func (t *Trie[V]) collectKeys(node *trieNode[V], path *[]rune, out *[]string) {
	if node == nil {
		return
	}
	if node.hasValue {
		*out = append(*out, string(*path))
	}
	if node.children.Len() == 0 {
		return
	}

	lo.ForEach(node.childKeys, func(ch rune, _ int) {
		*path = append(*path, ch)
		child, _ := node.children.Get(ch)
		t.collectKeys(child, path, out)
		*path = (*path)[:len(*path)-1]
	})
}

func (t *Trie[V]) collectValues(node *trieNode[V], out *[]V) {
	if node == nil {
		return
	}
	if node.hasValue {
		*out = append(*out, node.value)
	}
	if node.children.Len() == 0 {
		return
	}

	lo.ForEach(node.childKeys, func(ch rune, _ int) {
		child, _ := node.children.Get(ch)
		t.collectValues(child, out)
	})
}

func (t *Trie[V]) rangePrefix(node *trieNode[V], path *[]rune, fn func(key string, value V) bool) bool {
	if node == nil {
		return true
	}
	if node.hasValue && !fn(string(*path), node.value) {
		return false
	}
	if node.children.Len() == 0 {
		return true
	}

	return lo.EveryBy(node.childKeys, func(ch rune) bool {
		*path = append(*path, ch)
		child, _ := node.children.Get(ch)
		if !t.rangePrefix(child, path, fn) {
			*path = (*path)[:len(*path)-1]
			return false
		}
		*path = (*path)[:len(*path)-1]
		return true
	})
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
