package list

import "slices"

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
