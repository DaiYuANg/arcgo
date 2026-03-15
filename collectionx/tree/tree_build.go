package tree

import (
	collectionmapping "github.com/DaiYuANg/archgo/collectionx/mapping"
	"github.com/samber/lo"
)

// Build constructs a tree from entries.
func Build[K comparable, V any](entries []Entry[K, V]) (*Tree[K, V], error) {
	tree := NewTree[K, V]()
	if len(entries) == 0 {
		return tree, nil
	}

	var buildErr error
	lo.ForEach(entries, func(entry Entry[K, V], _ int) {
		if buildErr != nil {
			return
		}
		if tree.Has(entry.ID) {
			buildErr = ErrNodeAlreadyExists
			return
		}
		tree.nodes.Set(entry.ID, newNode(entry.ID, entry.Value))
	})
	if buildErr != nil {
		return nil, buildErr
	}

	lo.ForEach(entries, func(entry Entry[K, V], _ int) {
		if buildErr != nil {
			return
		}

		node, _ := tree.nodes.Get(entry.ID)
		if entry.ParentID.IsAbsent() {
			tree.roots.Add(node)
			return
		}

		parentID := entry.ParentID.MustGet()
		parent, ok := tree.nodes.Get(parentID)
		if !ok {
			buildErr = ErrParentNotFound
			return
		}

		node.parent = parent
		parent.children.Add(node)
	})
	if buildErr != nil {
		return nil, buildErr
	}

	if lo.SomeBy(tree.nodes.Values(), func(node *Node[K, V]) bool {
		return hasParentCycle(node)
	}) {
		return nil, ErrCycleDetected
	}

	return tree, nil
}

func hasParentCycle[K comparable, V any](node *Node[K, V]) bool {
	visited := collectionmapping.NewMap[*Node[K, V], struct{}]()
	for current := node; current != nil; current = current.parent {
		if _, exists := visited.Get(current); exists {
			return true
		}
		visited.Set(current, struct{}{})
	}
	return false
}
