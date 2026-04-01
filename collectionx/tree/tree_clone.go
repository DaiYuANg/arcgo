package tree

import "github.com/samber/lo"

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

	stack := lo.Map(lo.Range(rootCount), func(index int, _ int) pair {
		root, _ := t.roots.Get(index)
		rootClone := newNode(root.ID(), root.Value())
		cloned.roots.Add(rootClone)
		cloned.nodes.Set(rootClone.ID(), rootClone)
		return pair{source: root, target: rootClone}
	})

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		lo.ForEach(lo.Range(current.source.children.Len()), func(index int, _ int) {
			sourceChild, _ := current.source.children.Get(index)
			targetChild := newNode(sourceChild.ID(), sourceChild.Value())
			targetChild.parent = current.target
			current.target.children.Add(targetChild)
			cloned.nodes.Set(targetChild.ID(), targetChild)
			stack = append(stack, pair{source: sourceChild, target: targetChild})
		})
	}

	return cloned
}

func cloneSubtreeDetached[K comparable, V any](root *Node[K, V]) *Node[K, V] {
	if root == nil {
		return nil
	}

	rootClone := newNode(root.ID(), root.Value())
	type pair struct {
		source *Node[K, V]
		target *Node[K, V]
	}
	stack := []pair{{source: root, target: rootClone}}

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		lo.ForEach(lo.Range(current.source.children.Len()), func(index int, _ int) {
			sourceChild, _ := current.source.children.Get(index)
			targetChild := newNode(sourceChild.ID(), sourceChild.Value())
			targetChild.parent = current.target
			current.target.children.Add(targetChild)
			stack = append(stack, pair{source: sourceChild, target: targetChild})
		})
	}

	return rootClone
}
