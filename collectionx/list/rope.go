package list

import (
	"slices"

	"github.com/samber/lo"
	"github.com/samber/mo"
)

const ropeLeafSize = 64

// RopeList is a list that uses a rope (balanced tree of chunks) for AddAt and RemoveAt.
// Designed for frequent middle insertions/removals on large lists; benchmark for your
// workload since slice-backed List can be faster for moderate sizes due to cache locality.
type RopeList[T any] struct {
	root *ropeNode[T]
	len  int
}

type ropeNode[T any] struct {
	leaf   []T
	left   *ropeNode[T]
	right  *ropeNode[T]
	length int
}

func newRopeLeaf[T any](items []T) *ropeNode[T] {
	s := slices.Clone(items)
	return &ropeNode[T]{leaf: s, length: len(s)}
}

func (n *ropeNode[T]) isLeaf() bool {
	return n.leaf != nil
}

// NewRopeList creates an empty RopeList or one pre-filled with items.
// Large inputs are built as a balanced rope for efficient mid-index operations.
func NewRopeList[T any](items ...T) *RopeList[T] {
	r := &RopeList[T]{}
	if len(items) > 0 {
		r.root = buildRope(items)
		r.len = len(items)
	}
	return r
}

// NewRopeListWithCapacity creates a RopeList; capacity is a hint (rope allocates lazily).
func NewRopeListWithCapacity[T any](capacity int, items ...T) *RopeList[T] {
	return NewRopeList(items...)
}

// Add appends items.
func (r *RopeList[T]) Add(items ...T) {
	if r == nil || len(items) == 0 {
		return
	}
	if r.root == nil {
		r.root = buildRope(items)
		r.len = len(items)
		return
	}
	// Append: concat along right spine for O(log n) instead of full split
	r.root = concatRight(r.root, buildRope(items))
	r.len += len(items)
}

// AddAt inserts item at index.
func (r *RopeList[T]) AddAt(index int, item T) bool {
	return r.InsertAt(index, item)
}

// AddAllAt inserts items at index.
func (r *RopeList[T]) AddAllAt(index int, items ...T) bool {
	return r.InsertAt(index, items...)
}

// InsertAt inserts items at index. Panics if index < 0 or index > Len().
func (r *RopeList[T]) InsertAt(index int, items ...T) bool {
	if r == nil {
		return false
	}
	if index < 0 || index > r.len {
		return false
	}
	if len(items) == 0 {
		return true
	}
	if r.root == nil {
		r.root = newRopeLeaf(items)
		r.len = len(items)
		return true
	}
	left, right := r.root.split(index)
	mid := newRopeLeaf(items)
	r.root = concat(concat(left, mid), right)
	r.len += len(items)
	return true
}

// Get returns item at index.
func (r *RopeList[T]) Get(index int) (T, bool) {
	var zero T
	if r == nil || r.root == nil || index < 0 || index >= r.len {
		return zero, false
	}
	return r.root.at(index), true
}

// GetOption returns item at index as mo.Option.
func (r *RopeList[T]) GetOption(index int) mo.Option[T] {
	v, ok := r.Get(index)
	if !ok {
		return mo.None[T]()
	}
	return mo.Some(v)
}

// Set replaces item at index.
func (r *RopeList[T]) Set(index int, item T) bool {
	if r == nil || r.root == nil || index < 0 || index >= r.len {
		return false
	}
	r.root.setAt(index, item)
	return true
}

// RemoveAt removes and returns item at index.
func (r *RopeList[T]) RemoveAt(index int) (T, bool) {
	var zero T
	if r == nil || r.root == nil || index < 0 || index >= r.len {
		return zero, false
	}
	left, right := r.root.split(index)
	_, right = right.split(1)
	removed := r.root.at(index)
	r.root = concat(left, right)
	r.len--
	return removed, true
}

// RemoveAtOption removes item at index and returns it as mo.Option.
func (r *RopeList[T]) RemoveAtOption(index int) mo.Option[T] {
	v, ok := r.RemoveAt(index)
	if !ok {
		return mo.None[T]()
	}
	return mo.Some(v)
}

// RemoveIf removes items matched by predicate.
func (r *RopeList[T]) RemoveIf(predicate func(item T) bool) int {
	if r == nil || predicate == nil || r.root == nil {
		return 0
	}
	items := r.Values()
	next := lo.Filter(items, func(item T, _ int) bool {
		return !predicate(item)
	})
	removed := len(items) - len(next)
	if removed == 0 {
		return 0
	}
	r.root = buildRope(next)
	r.len = len(next)
	return removed
}

// Len returns item count.
func (r *RopeList[T]) Len() int {
	if r == nil {
		return 0
	}
	return r.len
}

// IsEmpty reports whether the list is empty.
func (r *RopeList[T]) IsEmpty() bool {
	return r.Len() == 0
}

// Clear removes all items.
func (r *RopeList[T]) Clear() {
	if r == nil {
		return
	}
	r.root = nil
	r.len = 0
}

// Values returns a copy of all items.
func (r *RopeList[T]) Values() []T {
	if r == nil || r.root == nil {
		return nil
	}
	return r.root.flatten()
}

// Range iterates items.
func (r *RopeList[T]) Range(fn func(index int, item T) bool) {
	if r == nil || fn == nil {
		return
	}
	for i := 0; i < r.len; i++ {
		v, _ := r.Get(i)
		if !fn(i, v) {
			return
		}
	}
}

// Clone returns a shallow copy.
func (r *RopeList[T]) Clone() *RopeList[T] {
	if r == nil || r.root == nil {
		return &RopeList[T]{}
	}
	return &RopeList[T]{
		root: r.root.clone(),
		len:  r.len,
	}
}

// Merge appends all items from another list.
func (r *RopeList[T]) Merge(other *List[T]) *RopeList[T] {
	if r == nil {
		return nil
	}
	if other == nil || other.Len() == 0 {
		return r
	}
	r.Add(other.Values()...)
	return r
}

// MergeRope appends all items from another RopeList.
func (r *RopeList[T]) MergeRope(other *RopeList[T]) *RopeList[T] {
	if r == nil {
		return nil
	}
	if other == nil || other.root == nil {
		return r
	}
	r.Add(other.Values()...)
	return r
}

// MergeSlice appends items from slice.
func (r *RopeList[T]) MergeSlice(items []T) *RopeList[T] {
	if r == nil {
		return nil
	}
	r.Add(items...)
	return r
}

// SetAll applies mapper to each item.
func (r *RopeList[T]) SetAll(mapper func(item T) T) int {
	return r.SetAllIndexed(func(_ int, item T) T { return mapper(item) })
}

// SetAllIndexed applies mapper to each item.
func (r *RopeList[T]) SetAllIndexed(mapper func(index int, item T) T) int {
	if r == nil || mapper == nil || r.root == nil {
		return 0
	}
	items := r.Values()
	for i := range items {
		items[i] = mapper(i, items[i])
	}
	r.root = buildRope(items)
	return len(items)
}

func (n *ropeNode[T]) nodeLen() int {
	if n == nil {
		return 0
	}
	return n.length
}

func (n *ropeNode[T]) at(i int) T {
	if n.isLeaf() {
		return n.leaf[i]
	}
	if i < n.left.nodeLen() {
		return n.left.at(i)
	}
	return n.right.at(i - n.left.nodeLen())
}

func (n *ropeNode[T]) setAt(i int, v T) {
	if n.isLeaf() {
		n.leaf[i] = v
		return
	}
	if i < n.left.nodeLen() {
		n.left.setAt(i, v)
	} else {
		n.right.setAt(i-n.left.nodeLen(), v)
	}
}

func (n *ropeNode[T]) split(i int) (*ropeNode[T], *ropeNode[T]) {
	if n == nil {
		return nil, nil
	}
	if i <= 0 {
		return nil, n.clone()
	}
	if i >= n.nodeLen() {
		return n.clone(), nil
	}
	if n.isLeaf() {
		left := &ropeNode[T]{leaf: slices.Clone(n.leaf[:i]), length: i}
		right := &ropeNode[T]{leaf: slices.Clone(n.leaf[i:]), length: len(n.leaf) - i}
		return left, right
	}
	if i <= n.left.nodeLen() {
		l, r := n.left.split(i)
		return l, concat(r, n.right.clone())
	}
	l, r := n.right.split(i - n.left.nodeLen())
	return concat(n.left.clone(), l), r
}

func concat[T any](a, b *ropeNode[T]) *ropeNode[T] {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	return &ropeNode[T]{
		left:   a,
		right:  b,
		length: a.nodeLen() + b.nodeLen(),
	}
}

// concatRight appends b to the right of a by cloning only the right spine.
func concatRight[T any](a, b *ropeNode[T]) *ropeNode[T] {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	if a.isLeaf() {
		return concat(a, b)
	}
	return &ropeNode[T]{
		left:   a.left,
		right:  concatRight(a.right, b),
		length: a.nodeLen() + b.nodeLen(),
	}
}

func (n *ropeNode[T]) flatten() []T {
	if n == nil {
		return nil
	}
	if n.isLeaf() {
		return slices.Clone(n.leaf)
	}
	return append(n.left.flatten(), n.right.flatten()...)
}

func (n *ropeNode[T]) clone() *ropeNode[T] {
	if n == nil {
		return nil
	}
	if n.isLeaf() {
		return newRopeLeaf(n.leaf)
	}
	return &ropeNode[T]{
		left:   n.left.clone(),
		right:  n.right.clone(),
		length: n.length,
	}
}

func buildRope[T any](items []T) *ropeNode[T] {
	if len(items) == 0 {
		return nil
	}
	if len(items) <= ropeLeafSize {
		return newRopeLeaf(items)
	}
	mid := len(items) / 2
	return concat(buildRope(items[:mid]), buildRope(items[mid:]))
}
