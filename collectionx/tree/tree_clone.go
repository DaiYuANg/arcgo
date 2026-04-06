package tree

// Clone returns a deep copy preserving parent-children structure.
func (t *Tree[K, V]) Clone() *Tree[K, V] {
	if t == nil || t.nodes == nil || t.nodes.IsEmpty() {
		return NewTree[K, V]()
	}

	rootCount := 0
	if t.roots != nil {
		rootCount = t.roots.Len()
	}
	cloned := newTreeWithCapacity[K, V](t.nodes.Len(), rootCount)

	type pair struct {
		source *Node[K, V]
		target *Node[K, V]
	}

	stack := make([]pair, 0, rootCount)
	for index := range rootCount {
		root, _ := t.roots.Get(index)
		rootClone := newNode(root.ID(), root.Value())
		cloned.roots.Add(rootClone)
		cloned.nodes.Set(rootClone.ID(), rootClone)
		stack = append(stack, pair{source: root, target: rootClone})
	}

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		childCount := current.source.children.Len()
		for index := range childCount {
			sourceChild, _ := current.source.children.Get(index)
			targetChild := newNode(sourceChild.ID(), sourceChild.Value())
			targetChild.parent = current.target
			current.target.children.Add(targetChild)
			cloned.nodes.Set(targetChild.ID(), targetChild)
			stack = append(stack, pair{source: sourceChild, target: targetChild})
		}
	}

	return cloned
}

func cloneSubtreeDetached[K comparable, V any](root *Node[K, V]) *Node[K, V] {
	rootClone, _ := cloneSubtreeDetachedWithDescendants(root, false)
	return rootClone
}

func cloneSubtreeDetachedWithDescendants[K comparable, V any](root *Node[K, V], collectDescendants bool) (*Node[K, V], []*Node[K, V]) {
	if root == nil {
		return nil, nil
	}

	rootClone := newNode(root.ID(), root.Value())

	type pair struct {
		source *Node[K, V]
		target *Node[K, V]
	}

	stack := []pair{{source: root, target: rootClone}}
	var descendants []*Node[K, V]
	if collectDescendants && root.children.Len() > 0 {
		descendants = make([]*Node[K, V], 0, root.children.Len())
	}

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if collectDescendants && current.source != root {
			descendants = append(descendants, current.target)
		}

		for index := current.source.children.Len() - 1; index >= 0; index-- {
			sourceChild, _ := current.source.children.Get(index)
			targetChild := newNode(sourceChild.ID(), sourceChild.Value())
			targetChild.parent = current.target
			current.target.children.Add(targetChild)
			stack = append(stack, pair{source: sourceChild, target: targetChild})
		}
	}

	return rootClone, descendants
}
