package collectionx

import "github.com/DaiYuANg/archgo/collectionx/tree"

type treeWritable[K comparable, V any] interface {
	AddRoot(id K, value V) error
	AddChild(parentID K, id K, value V) error
	Move(id K, newParentID K) error
	Remove(id K) bool
	SetValue(id K, value V) bool
	clearable
}

type treeReadable[K comparable, V any] interface {
	Get(id K) (*tree.Node[K, V], bool)
	Has(id K) bool
	Parent(id K) (*tree.Node[K, V], bool)
	Children(id K) []*tree.Node[K, V]
	Roots() []*tree.Node[K, V]
	Ancestors(id K) []*tree.Node[K, V]
	Descendants(id K) []*tree.Node[K, V]
	RangeDFS(fn func(node *tree.Node[K, V]) bool)
	sized
}

type Tree[K comparable, V any] interface {
	treeWritable[K, V]
	treeReadable[K, V]
	clonable[*tree.Tree[K, V]]
	jsonStringer
}

func NewTree[K comparable, V any]() Tree[K, V] {
	return tree.NewTree[K, V]()
}

type ConcurrentTree[K comparable, V any] interface {
	treeWritable[K, V]
	treeReadable[K, V]
	snapshotable[*tree.Tree[K, V]]
	jsonStringer
}

func NewConcurrentTree[K comparable, V any]() ConcurrentTree[K, V] {
	return tree.NewConcurrentTree[K, V]()
}

type TreeNode[K comparable, V any] = tree.Node[K, V]

type TreeEntry[K comparable, V any] = tree.Entry[K, V]

func NewRootTreeEntry[K comparable, V any](id K, value V) TreeEntry[K, V] {
	return tree.RootEntry(id, value)
}

func NewChildTreeEntry[K comparable, V any](id K, parentID K, value V) TreeEntry[K, V] {
	return tree.ChildEntry(id, parentID, value)
}

func BuildTree[K comparable, V any](entries []TreeEntry[K, V]) (Tree[K, V], error) {
	return tree.Build(entries)
}

func BuildConcurrentTree[K comparable, V any](entries []TreeEntry[K, V]) (ConcurrentTree[K, V], error) {
	return tree.BuildConcurrent(entries)
}

var (
	ErrTreeNodeAlreadyExists = tree.ErrNodeAlreadyExists
	ErrTreeNodeNotFound      = tree.ErrNodeNotFound
	ErrTreeParentNotFound    = tree.ErrParentNotFound
	ErrTreeCycleDetected     = tree.ErrCycleDetected
)
