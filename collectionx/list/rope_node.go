package list

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

func (n *ropeNode[T]) insertAt(i int, item T) *ropeNode[T] {
	if n == nil {
		return newRopeLeaf([]T{item})
	}

	if n.isLeaf() {
		return n.insertIntoLeaf(i, item)
	}

	leftLen := n.left.nodeLen()
	if i <= leftLen {
		n.left = n.left.insertAt(i, item)
	} else {
		n.right = n.right.insertAt(i-leftLen, item)
	}
	n.recomputeLength()
	return n.rebalanceIfNeeded()
}

func (n *ropeNode[T]) insertIntoLeaf(i int, item T) *ropeNode[T] {
	if len(n.leaf) < ropeLeafSize {
		var zero T
		n.leaf = append(n.leaf, zero)
		copy(n.leaf[i+1:], n.leaf[i:])
		n.leaf[i] = item
		n.length = len(n.leaf)
		return n
	}

	merged := make([]T, len(n.leaf)+1)
	copy(merged, n.leaf[:i])
	merged[i] = item
	copy(merged[i+1:], n.leaf[i:])

	mid := len(merged) / 2
	n.left = newRopeLeaf(merged[:mid])
	n.right = newRopeLeaf(merged[mid:])
	n.leaf = nil
	n.recomputeLength()
	return n
}

func (n *ropeNode[T]) removeAt(i int) (*ropeNode[T], T, bool) {
	var zero T
	if n == nil {
		return nil, zero, false
	}

	if n.isLeaf() {
		removed := n.leaf[i]
		copy(n.leaf[i:], n.leaf[i+1:])
		last := len(n.leaf) - 1
		n.leaf[last] = zero
		n.leaf = n.leaf[:last]
		n.length = len(n.leaf)
		if len(n.leaf) == 0 {
			return nil, removed, true
		}
		return n, removed, true
	}

	leftLen := n.left.nodeLen()
	var removed T
	var ok bool
	if i < leftLen {
		n.left, removed, ok = n.left.removeAt(i)
	} else {
		n.right, removed, ok = n.right.removeAt(i - leftLen)
	}
	if !ok {
		return n, zero, false
	}

	n = n.compact()
	if n != nil {
		n.recomputeLength()
		n = n.rebalanceIfNeeded()
	}
	return n, removed, true
}

func (n *ropeNode[T]) compact() *ropeNode[T] {
	if n == nil {
		return nil
	}
	if n.left == nil {
		return n.right
	}
	if n.right == nil {
		return n.left
	}
	if n.left.isLeaf() && n.right.isLeaf() && len(n.left.leaf)+len(n.right.leaf) <= ropeLeafSize {
		merged := make([]T, 0, len(n.left.leaf)+len(n.right.leaf))
		merged = append(merged, n.left.leaf...)
		merged = append(merged, n.right.leaf...)
		return newRopeLeaf(merged)
	}
	return n
}

func (n *ropeNode[T]) recomputeLength() {
	if n == nil {
		return
	}
	if n.isLeaf() {
		n.length = len(n.leaf)
		return
	}
	n.length = n.left.nodeLen() + n.right.nodeLen()
}

func (n *ropeNode[T]) rebalanceIfNeeded() *ropeNode[T] {
	if n == nil || n.isLeaf() {
		return n
	}

	leftLen := n.left.nodeLen()
	rightLen := n.right.nodeLen()
	smaller := min(leftLen, rightLen)
	if smaller == 0 {
		return n.compact()
	}
	if n.length <= ropeLeafSize*2 {
		return n
	}
	if leftLen <= rightLen*4 && rightLen <= leftLen*4 {
		return n
	}

	items := n.flatten()
	return buildRope(items)
}

func (n *ropeNode[T]) flatten() []T {
	if n == nil {
		return nil
	}

	out := make([]T, 0, n.nodeLen())
	return n.appendTo(out)
}

func (n *ropeNode[T]) appendTo(dst []T) []T {
	if n == nil {
		return dst
	}
	if n.isLeaf() {
		return append(dst, n.leaf...)
	}
	dst = n.left.appendTo(dst)
	return n.right.appendTo(dst)
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
	left := buildRope(items[:mid])
	right := buildRope(items[mid:])
	return &ropeNode[T]{
		left:   left,
		right:  right,
		length: left.nodeLen() + right.nodeLen(),
	}
}
