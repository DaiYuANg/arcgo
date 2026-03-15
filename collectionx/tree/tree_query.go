package tree

import collectionlist "github.com/DaiYuANg/archgo/collectionx/list"

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

	ancestors := collectionlist.NewList[*Node[K, V]]()
	for current := node.parent; current != nil; current = current.parent {
		ancestors.Add(current)
	}
	return ancestors.Values()
}

// Descendants returns all descendants in DFS pre-order.
func (t *Tree[K, V]) Descendants(id K) []*Node[K, V] {
	node, ok := t.Get(id)
	if !ok {
		return nil
	}

	children := node.Children()
	if len(children) == 0 {
		return nil
	}

	descendants := collectionlist.NewList[*Node[K, V]]()
	stack := make([]*Node[K, V], 0, len(children))
	for i := len(children) - 1; i >= 0; i-- {
		stack = append(stack, children[i])
	}

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		descendants.Add(current)

		currentChildren := current.Children()
		for i := len(currentChildren) - 1; i >= 0; i-- {
			stack = append(stack, currentChildren[i])
		}
	}

	return descendants.Values()
}

// RangeDFS iterates all nodes in DFS pre-order until fn returns false.
func (t *Tree[K, V]) RangeDFS(fn func(node *Node[K, V]) bool) {
	if t == nil || fn == nil {
		return
	}

	for _, root := range t.Roots() {
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
