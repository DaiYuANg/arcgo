package tree

import (
	collectionlist "github.com/DaiYuANg/arcgo/collectionx/list"
	collectionmapping "github.com/DaiYuANg/arcgo/collectionx/mapping"
)

// AddRoot inserts one root node.
func (t *Tree[K, V]) AddRoot(id K, value V) error {
	if t == nil {
		return ErrNodeNotFound
	}
	t.ensureInit()
	if t.Has(id) {
		return ErrNodeAlreadyExists
	}

	node := newNode(id, value)
	t.nodes.Set(id, node)
	t.roots.Add(node)
	return nil
}

// AddChild inserts one child node under parentID.
func (t *Tree[K, V]) AddChild(parentID K, id K, value V) error {
	if t == nil {
		return ErrNodeNotFound
	}
	t.ensureInit()
	if t.Has(id) {
		return ErrNodeAlreadyExists
	}

	parent, ok := t.nodes.Get(parentID)
	if !ok {
		return ErrParentNotFound
	}

	node := newNode(id, value)
	node.parent = parent
	parent.children.Add(node)
	t.nodes.Set(id, node)
	return nil
}

// Move moves node id under newParentID.
func (t *Tree[K, V]) Move(id K, newParentID K) error {
	if t == nil || t.nodes == nil {
		return ErrNodeNotFound
	}

	node, ok := t.nodes.Get(id)
	if !ok {
		return ErrNodeNotFound
	}

	newParent, ok := t.nodes.Get(newParentID)
	if !ok {
		return ErrParentNotFound
	}

	if node == newParent {
		return ErrCycleDetected
	}
	for current := newParent; current != nil; current = current.parent {
		if current == node {
			return ErrCycleDetected
		}
	}

	t.detach(node)
	node.parent = newParent
	newParent.children.Add(node)
	return nil
}

// Remove deletes one node and its whole subtree.
func (t *Tree[K, V]) Remove(id K) bool {
	if t == nil || t.nodes == nil {
		return false
	}

	node, ok := t.nodes.Get(id)
	if !ok {
		return false
	}

	t.detach(node)
	t.removeSubtree(node)
	return true
}

// SetValue updates node value by id.
func (t *Tree[K, V]) SetValue(id K, value V) bool {
	node, ok := t.Get(id)
	if !ok {
		return false
	}
	node.value = value
	return true
}

// Clear removes all nodes.
func (t *Tree[K, V]) Clear() {
	if t == nil {
		return
	}
	if t.nodes != nil {
		t.nodes.Clear()
	}
	if t.roots != nil {
		t.roots.Clear()
	}
}

func (t *Tree[K, V]) ensureInit() {
	if t.nodes == nil {
		t.nodes = collectionmapping.NewMap[K, *Node[K, V]]()
	}
	if t.roots == nil {
		t.roots = collectionlist.NewList[*Node[K, V]]()
	}
}

func (t *Tree[K, V]) detach(node *Node[K, V]) {
	if node.parent != nil {
		parent := node.parent
		parent.children.RemoveIf(func(item *Node[K, V]) bool {
			return item == node
		})
		node.parent = nil
		return
	}

	if t.roots != nil {
		t.roots.RemoveIf(func(item *Node[K, V]) bool {
			return item == node
		})
	}
}

func (t *Tree[K, V]) removeSubtree(node *Node[K, V]) {
	if node == nil {
		return
	}

	type stackItem struct {
		node    *Node[K, V]
		visited bool
	}
	stack := []stackItem{{node: node}}
	for len(stack) > 0 {
		item := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if !item.visited {
			stack = append(stack, stackItem{node: item.node, visited: true})
			for i := item.node.children.Len() - 1; i >= 0; i-- {
				child, _ := item.node.children.Get(i)
				stack = append(stack, stackItem{node: child})
			}
			continue
		}

		_ = t.nodes.Delete(item.node.id)
		item.node.parent = nil
		item.node.children.Clear()
	}
}

func newNode[K comparable, V any](id K, value V) *Node[K, V] {
	return &Node[K, V]{
		id:    id,
		value: value,
	}
}
