package tree

import "github.com/samber/lo"

// Get returns node by id as a detached subtree clone.
func (t *ConcurrentTree[K, V]) Get(id K) (*Node[K, V], bool) {
	if t == nil {
		return nil, false
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.tree == nil {
		return nil, false
	}
	node, ok := t.tree.Get(id)
	if !ok {
		return nil, false
	}
	return cloneNodeWithAncestors(node), true
}

// Has reports whether id exists.
func (t *ConcurrentTree[K, V]) Has(id K) bool {
	if t == nil {
		return false
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.tree == nil {
		return false
	}
	return t.tree.Has(id)
}

// Parent returns parent node by child id as a detached subtree clone.
func (t *ConcurrentTree[K, V]) Parent(id K) (*Node[K, V], bool) {
	if t == nil {
		return nil, false
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.tree == nil {
		return nil, false
	}
	node, ok := t.tree.Get(id)
	if !ok || node.parent == nil {
		return nil, false
	}
	return cloneNodeWithAncestors(node.parent), true
}

// Children returns children snapshot by node id.
func (t *ConcurrentTree[K, V]) Children(id K) []*Node[K, V] {
	if t == nil {
		return nil
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.tree == nil {
		return nil
	}
	node, ok := t.tree.Get(id)
	if !ok {
		return nil
	}
	if node.children.Len() == 0 {
		return nil
	}

	return lo.Map(lo.Range(node.children.Len()), func(index int, _ int) *Node[K, V] {
		child, _ := node.children.Get(index)
		return cloneSubtreeDetached(child)
	})
}

// Roots returns root nodes snapshot.
func (t *ConcurrentTree[K, V]) Roots() []*Node[K, V] {
	if t == nil {
		return nil
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.tree == nil {
		return nil
	}
	if t.tree.roots == nil || t.tree.roots.Len() == 0 {
		return nil
	}
	return lo.Map(lo.Range(t.tree.roots.Len()), func(index int, _ int) *Node[K, V] {
		root, _ := t.tree.roots.Get(index)
		return cloneSubtreeDetached(root)
	})
}

// Ancestors returns parent chain from direct parent to top root.
func (t *ConcurrentTree[K, V]) Ancestors(id K) []*Node[K, V] {
	if t == nil {
		return nil
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.tree == nil {
		return nil
	}
	node, ok := t.tree.Get(id)
	if !ok {
		return nil
	}

	ancestors := make([]*Node[K, V], 0)
	for current := node.parent; current != nil; current = current.parent {
		ancestors = append(ancestors, current)
	}
	if len(ancestors) == 0 {
		return nil
	}

	out := make([]*Node[K, V], len(ancestors))
	var parentClone *Node[K, V]
	for i := len(ancestors) - 1; i >= 0; i-- {
		currentClone := newNode(ancestors[i].ID(), ancestors[i].Value())
		currentClone.parent = parentClone
		if parentClone != nil {
			parentClone.children.Add(currentClone)
		}
		out[i] = currentClone
		parentClone = currentClone
	}
	return out
}

// Descendants returns all descendants in DFS pre-order.
func (t *ConcurrentTree[K, V]) Descendants(id K) []*Node[K, V] {
	if t == nil {
		return nil
	}
	t.mu.RLock()
	if t.tree == nil {
		t.mu.RUnlock()
		return nil
	}
	node, ok := t.tree.Get(id)
	if !ok {
		t.mu.RUnlock()
		return nil
	}
	cloned := cloneSubtreeDetached(node)
	t.mu.RUnlock()
	return descendantsFromRoot(cloned)
}

// RangeDFS iterates all nodes in DFS pre-order until fn returns false.
func (t *ConcurrentTree[K, V]) RangeDFS(fn func(node *Node[K, V]) bool) {
	if t == nil || fn == nil {
		return
	}

	rangeDFSRoots(t.snapshotClonedRoots(), fn)
}

// Len returns total node count.
func (t *ConcurrentTree[K, V]) Len() int {
	if t == nil {
		return 0
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.tree == nil {
		return 0
	}
	return t.tree.Len()
}

// IsEmpty reports whether tree has no nodes.
func (t *ConcurrentTree[K, V]) IsEmpty() bool {
	return t.Len() == 0
}

func descendantsFromRoot[K comparable, V any](root *Node[K, V]) []*Node[K, V] {
	if root == nil {
		return nil
	}

	if root.children.Len() == 0 {
		return nil
	}

	out := make([]*Node[K, V], 0, root.children.Len())
	stack := appendChildrenReverse(make([]*Node[K, V], 0, root.children.Len()), root)

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		out = append(out, current)

		stack = appendChildrenReverse(stack, current)
	}

	return out
}

func cloneNodeWithAncestors[K comparable, V any](node *Node[K, V]) *Node[K, V] {
	if node == nil {
		return nil
	}

	targetClone := cloneSubtreeDetached(node)
	currentClone := targetClone
	for currentParent := node.parent; currentParent != nil; currentParent = currentParent.parent {
		parentClone := newNode(currentParent.ID(), currentParent.Value())
		currentClone.parent = parentClone
		parentClone.children.Add(currentClone)
		currentClone = parentClone
	}
	return targetClone
}

func (t *ConcurrentTree[K, V]) snapshotClonedRoots() []*Node[K, V] {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.tree == nil || t.tree.roots == nil || t.tree.roots.Len() == 0 {
		return nil
	}

	return lo.Map(lo.Range(t.tree.roots.Len()), func(index int, _ int) *Node[K, V] {
		root, _ := t.tree.roots.Get(index)
		return cloneSubtreeDetached(root)
	})
}
