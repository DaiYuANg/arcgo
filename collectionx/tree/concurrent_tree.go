package tree

import "sync"

// ConcurrentTree is a goroutine-safe tree wrapper.
// Read APIs that return nodes always clone data to avoid exposing mutable internals.
// Zero value is ready to use.
type ConcurrentTree[K comparable, V any] struct {
	mu   sync.RWMutex
	tree *Tree[K, V]
}

// NewConcurrentTree creates an empty concurrent tree.
func NewConcurrentTree[K comparable, V any]() *ConcurrentTree[K, V] {
	return &ConcurrentTree[K, V]{
		tree: NewTree[K, V](),
	}
}

// BuildConcurrent constructs a concurrent tree from entries.
func BuildConcurrent[K comparable, V any](entries []Entry[K, V]) (*ConcurrentTree[K, V], error) {
	tree, err := Build(entries)
	if err != nil {
		return nil, err
	}
	return &ConcurrentTree[K, V]{tree: tree}, nil
}

// AddRoot inserts one root node.
func (t *ConcurrentTree[K, V]) AddRoot(id K, value V) error {
	if t == nil {
		return ErrNodeNotFound
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ensureInitLocked()
	return t.tree.AddRoot(id, value)
}

// AddChild inserts one child node under parentID.
func (t *ConcurrentTree[K, V]) AddChild(parentID K, id K, value V) error {
	if t == nil {
		return ErrNodeNotFound
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ensureInitLocked()
	return t.tree.AddChild(parentID, id, value)
}

// Move moves node id under newParentID.
func (t *ConcurrentTree[K, V]) Move(id K, newParentID K) error {
	if t == nil {
		return ErrNodeNotFound
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ensureInitLocked()
	return t.tree.Move(id, newParentID)
}

// Remove deletes one node and its whole subtree.
func (t *ConcurrentTree[K, V]) Remove(id K) bool {
	if t == nil {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ensureInitLocked()
	return t.tree.Remove(id)
}

// SetValue updates node value by id.
func (t *ConcurrentTree[K, V]) SetValue(id K, value V) bool {
	if t == nil {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ensureInitLocked()
	return t.tree.SetValue(id, value)
}

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
	return cloneSubtreeDetached(node).Children()
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
	roots := t.tree.Roots()
	if len(roots) == 0 {
		return nil
	}
	out := make([]*Node[K, V], 0, len(roots))
	for _, root := range roots {
		out = append(out, cloneSubtreeDetached(root))
	}
	return out
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

	t.mu.RLock()
	if t.tree == nil {
		t.mu.RUnlock()
		return
	}
	roots := t.tree.Roots()
	clonedRoots := make([]*Node[K, V], 0, len(roots))
	for _, root := range roots {
		clonedRoots = append(clonedRoots, cloneSubtreeDetached(root))
	}
	t.mu.RUnlock()

	for _, root := range clonedRoots {
		stack := []*Node[K, V]{root}
		for len(stack) > 0 {
			current := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if !fn(current) {
				return
			}

			children := current.Children()
			for i := len(children) - 1; i >= 0; i-- {
				stack = append(stack, children[i])
			}
		}
	}
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

// Clear removes all nodes.
func (t *ConcurrentTree[K, V]) Clear() {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ensureInitLocked()
	t.tree.Clear()
}

// Snapshot returns an immutable-style copy in a normal Tree.
func (t *ConcurrentTree[K, V]) Snapshot() *Tree[K, V] {
	if t == nil {
		return NewTree[K, V]()
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.tree == nil {
		return NewTree[K, V]()
	}
	return t.tree.Clone()
}

func (t *ConcurrentTree[K, V]) ensureInitLocked() {
	if t.tree == nil {
		t.tree = NewTree[K, V]()
	}
}

func descendantsFromRoot[K comparable, V any](root *Node[K, V]) []*Node[K, V] {
	if root == nil {
		return nil
	}

	children := root.Children()
	if len(children) == 0 {
		return nil
	}

	out := make([]*Node[K, V], 0)
	stack := make([]*Node[K, V], 0, len(children))
	for i := len(children) - 1; i >= 0; i-- {
		stack = append(stack, children[i])
	}

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		out = append(out, current)

		currentChildren := current.Children()
		for i := len(currentChildren) - 1; i >= 0; i-- {
			stack = append(stack, currentChildren[i])
		}
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
