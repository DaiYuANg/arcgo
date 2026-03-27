package tree

// Get returns node by id.
func (t *Tree[K, V]) Get(id K) (*Node[K, V], bool) {
	if t == nil || t.nodes == nil {
		return nil, false
	}
	return t.nodes.Get(id)
}

// Has reports whether id exists.
func (t *Tree[K, V]) Has(id K) bool {
	_, ok := t.Get(id)
	return ok
}

// Parent returns parent node by child id.
func (t *Tree[K, V]) Parent(id K) (*Node[K, V], bool) {
	node, ok := t.Get(id)
	if !ok || node.parent == nil {
		return nil, false
	}
	return node.parent, true
}

// Children returns children snapshot by node id.
func (t *Tree[K, V]) Children(id K) []*Node[K, V] {
	node, ok := t.Get(id)
	if !ok {
		return nil
	}
	return node.Children()
}

// Roots returns root nodes snapshot.
func (t *Tree[K, V]) Roots() []*Node[K, V] {
	if t == nil || t.roots == nil {
		return nil
	}
	return t.roots.Values()
}

// Ancestors returns parent chain from direct parent to top root.
func (t *Tree[K, V]) Ancestors(id K) []*Node[K, V] {
	node, ok := t.Get(id)
	if !ok {
		return nil
	}

	depth := 0
	for current := node.parent; current != nil; current = current.parent {
		depth++
	}
	if depth == 0 {
		return nil
	}

	ancestors := make([]*Node[K, V], 0, depth)
	for current := node.parent; current != nil; current = current.parent {
		ancestors = append(ancestors, current)
	}
	return ancestors
}

// Descendants returns all descendants in DFS pre-order.
func (t *Tree[K, V]) Descendants(id K) []*Node[K, V] {
	node, ok := t.Get(id)
	if !ok {
		return nil
	}

	if node.children.Len() == 0 {
		return nil
	}

	capacity := t.Len()
	if capacity <= 0 {
		capacity = node.children.Len()
	}
	descendants := make([]*Node[K, V], 0, capacity)
	stack := appendChildrenReverse(make([]*Node[K, V], 0, capacity), node)

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		descendants = append(descendants, current)

		stack = appendChildrenReverse(stack, current)
	}

	return descendants
}

// RangeDFS iterates all nodes in DFS pre-order until fn returns false.
func (t *Tree[K, V]) RangeDFS(fn func(node *Node[K, V]) bool) {
	if t == nil || fn == nil {
		return
	}

	rangeDFSRoots(t.Roots(), fn)
}

// Len returns total node count.
func (t *Tree[K, V]) Len() int {
	if t == nil || t.nodes == nil {
		return 0
	}
	return t.nodes.Len()
}

// IsEmpty reports whether tree has no nodes.
func (t *Tree[K, V]) IsEmpty() bool {
	return t.Len() == 0
}

func rangeDFSRoots[K comparable, V any](roots []*Node[K, V], fn func(node *Node[K, V]) bool) {
	for _, root := range roots {
		if !rangeDFSFromRoot(root, fn) {
			return
		}
	}
}

func rangeDFSFromRoot[K comparable, V any](root *Node[K, V], fn func(node *Node[K, V]) bool) bool {
	if root == nil {
		return true
	}

	stack := []*Node[K, V]{root}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if !fn(current) {
			return false
		}

		stack = appendChildrenReverse(stack, current)
	}

	return true
}

func appendChildrenReverse[K comparable, V any](stack []*Node[K, V], node *Node[K, V]) []*Node[K, V] {
	if node == nil {
		return stack
	}

	for i := node.children.Len() - 1; i >= 0; i-- {
		child, _ := node.children.Get(i)
		stack = append(stack, child)
	}

	return stack
}
