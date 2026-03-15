package tree

import (
	"errors"

	collectionlist "github.com/DaiYuANg/archgo/collectionx/list"
	collectionmapping "github.com/DaiYuANg/archgo/collectionx/mapping"
	"github.com/samber/mo"
)

var (
	// ErrNodeAlreadyExists indicates a duplicate node id.
	ErrNodeAlreadyExists = errors.New("tree: node already exists")
	// ErrNodeNotFound indicates the node does not exist.
	ErrNodeNotFound = errors.New("tree: node not found")
	// ErrParentNotFound indicates the parent node does not exist.
	ErrParentNotFound = errors.New("tree: parent node not found")
	// ErrCycleDetected indicates an operation would create a cycle.
	ErrCycleDetected = errors.New("tree: cycle detected")
)

// Entry describes a node used for bulk tree construction.
// ParentID.None means this entry is a root node.
type Entry[K comparable, V any] struct {
	ID       K
	ParentID mo.Option[K]
	Value    V
}

// RootEntry creates one root entry.
func RootEntry[K comparable, V any](id K, value V) Entry[K, V] {
	return Entry[K, V]{
		ID:       id,
		ParentID: mo.None[K](),
		Value:    value,
	}
}

// ChildEntry creates one child entry.
func ChildEntry[K comparable, V any](id K, parentID K, value V) Entry[K, V] {
	return Entry[K, V]{
		ID:       id,
		ParentID: mo.Some(parentID),
		Value:    value,
	}
}

// Node represents one tree node with parent-children links.
type Node[K comparable, V any] struct {
	id       K
	value    V
	parent   *Node[K, V]
	children *collectionlist.List[*Node[K, V]]
}

// ID returns node id.
func (n *Node[K, V]) ID() K {
	var zero K
	if n == nil {
		return zero
	}
	return n.id
}

// Value returns node value.
func (n *Node[K, V]) Value() V {
	var zero V
	if n == nil {
		return zero
	}
	return n.value
}

// SetValue updates node value.
func (n *Node[K, V]) SetValue(value V) {
	if n == nil {
		return
	}
	n.value = value
}

// Parent returns parent node. Root nodes return nil.
func (n *Node[K, V]) Parent() *Node[K, V] {
	if n == nil {
		return nil
	}
	return n.parent
}

// Children returns child nodes as a snapshot.
func (n *Node[K, V]) Children() []*Node[K, V] {
	if n == nil || n.children == nil {
		return nil
	}
	return n.children.Values()
}

// ChildCount returns child count.
func (n *Node[K, V]) ChildCount() int {
	if n == nil || n.children == nil {
		return 0
	}
	return n.children.Len()
}

// IsRoot reports whether node is a root.
func (n *Node[K, V]) IsRoot() bool {
	return n != nil && n.parent == nil
}

// IsLeaf reports whether node has no children.
func (n *Node[K, V]) IsLeaf() bool {
	return n != nil && n.ChildCount() == 0
}

// Tree stores parent-children relationships by node id.
// Zero value is ready to use.
type Tree[K comparable, V any] struct {
	nodes *collectionmapping.Map[K, *Node[K, V]]
	roots *collectionlist.List[*Node[K, V]]
}

// NewTree creates an empty tree.
func NewTree[K comparable, V any]() *Tree[K, V] {
	return &Tree[K, V]{
		nodes: collectionmapping.NewMap[K, *Node[K, V]](),
		roots: collectionlist.NewList[*Node[K, V]](),
	}
}
