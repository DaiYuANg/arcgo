package tree

import (
	"errors"

	collectionlist "github.com/DaiYuANg/arcgo/collectionx/list"
	collectionmapping "github.com/DaiYuANg/arcgo/collectionx/mapping"
	"github.com/samber/lo"
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

// Get returns node by id.
func (t *Tree[K, V]) Get(id K) (*Node[K, V], bool) {
	if t == nil || t.nodes == nil {
		return nil, false
	}
	return t.nodes.Get(id)
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

// Clone returns a deep copy preserving parent-children structure.
func (t *Tree[K, V]) Clone() *Tree[K, V] {
	cloned := NewTree[K, V]()
	if t == nil || t.nodes == nil || t.nodes.IsEmpty() {
		return cloned
	}

	type pair struct {
		source *Node[K, V]
		target *Node[K, V]
	}

	stack := make([]pair, 0)
	for _, root := range t.Roots() {
		rootClone := newNode(root.ID(), root.Value())
		cloned.roots.Add(rootClone)
		cloned.nodes.Set(rootClone.ID(), rootClone)
		stack = append(stack, pair{source: root, target: rootClone})
	}

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		for _, sourceChild := range current.source.Children() {
			targetChild := newNode(sourceChild.ID(), sourceChild.Value())
			targetChild.parent = current.target
			current.target.children.Add(targetChild)
			cloned.nodes.Set(targetChild.ID(), targetChild)
			stack = append(stack, pair{source: sourceChild, target: targetChild})
		}
	}

	return cloned
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
			children := item.node.Children()
			for i := len(children) - 1; i >= 0; i-- {
				stack = append(stack, stackItem{node: children[i]})
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
		id:       id,
		value:    value,
		children: collectionlist.NewList[*Node[K, V]](),
	}
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

		for _, sourceChild := range current.source.Children() {
			targetChild := newNode(sourceChild.ID(), sourceChild.Value())
			targetChild.parent = current.target
			current.target.children.Add(targetChild)
			stack = append(stack, pair{source: sourceChild, target: targetChild})
		}
	}

	return rootClone
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
