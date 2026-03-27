package tree

import (
	"slices"

	collectionmapping "github.com/DaiYuANg/arcgo/collectionx/mapping"
)

// Build constructs a tree from entries.
func Build[K comparable, V any](entries []Entry[K, V]) (*Tree[K, V], error) {
	tree := NewTree[K, V]()
	if len(entries) == 0 {
		return tree, nil
	}

	if err := addBuildNodes(tree, entries); err != nil {
		return nil, err
	}

	if err := linkBuildNodes(tree, entries); err != nil {
		return nil, err
	}

	if hasTreeCycle(tree.nodes.Values()) {
		return nil, ErrCycleDetected
	}

	return tree, nil
}

func addBuildNodes[K comparable, V any](tree *Tree[K, V], entries []Entry[K, V]) error {
	for _, entry := range entries {
		if tree.Has(entry.ID) {
			return ErrNodeAlreadyExists
		}

		tree.nodes.Set(entry.ID, newNode(entry.ID, entry.Value))
	}

	return nil
}

func linkBuildNodes[K comparable, V any](tree *Tree[K, V], entries []Entry[K, V]) error {
	for _, entry := range entries {
		node, _ := tree.nodes.Get(entry.ID)
		if entry.ParentID.IsAbsent() {
			tree.roots.Add(node)
			continue
		}

		parentID := entry.ParentID.MustGet()
		parent, ok := tree.nodes.Get(parentID)
		if !ok {
			return ErrParentNotFound
		}

		node.parent = parent
		parent.children.Add(node)
	}

	return nil
}

func hasTreeCycle[K comparable, V any](nodes []*Node[K, V]) bool {
	return slices.ContainsFunc(nodes, hasParentCycle[K, V])
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
