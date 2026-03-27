package tree_test

import (
	"testing"

	tree "github.com/DaiYuANg/arcgo/collectionx/tree"
)

const (
	benchTreeNodes     = 10_000
	benchTreeBranching = 4
	benchTreeLeafID    = benchTreeNodes
)

func buildBenchTree(tb testing.TB) *tree.Tree[int, int] {
	tb.Helper()
	tr := tree.NewTree[int, int]()
	if err := tr.AddRoot(0, 0); err != nil {
		tb.Fatalf("AddRoot() error = %v", err)
	}
	for i := 1; i <= benchTreeNodes; i++ {
		parentID := (i - 1) / benchTreeBranching
		if err := tr.AddChild(parentID, i, i); err != nil {
			tb.Fatalf("AddChild(%d, %d) error = %v", parentID, i, err)
		}
	}
	return tr
}

func buildBenchConcurrentTree(tb testing.TB) *tree.ConcurrentTree[int, int] {
	tb.Helper()
	tr := tree.NewConcurrentTree[int, int]()
	if err := tr.AddRoot(0, 0); err != nil {
		tb.Fatalf("AddRoot() error = %v", err)
	}
	for i := 1; i <= benchTreeNodes; i++ {
		parentID := (i - 1) / benchTreeBranching
		if err := tr.AddChild(parentID, i, i); err != nil {
			tb.Fatalf("AddChild(%d, %d) error = %v", parentID, i, err)
		}
	}
	return tr
}

func BenchmarkTreeGet(b *testing.B) {
	tr := buildBenchTree(b)
	mask := benchTreeNodes - 1

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		_, _ = tr.Get((i & mask) + 1)
	}
}

func BenchmarkTreeChildren(b *testing.B) {
	tr := buildBenchTree(b)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = tr.Children(0)
	}
}

func BenchmarkTreeAncestors(b *testing.B) {
	tr := buildBenchTree(b)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = tr.Ancestors(benchTreeLeafID)
	}
}

func BenchmarkTreeDescendants(b *testing.B) {
	tr := buildBenchTree(b)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = tr.Descendants(0)
	}
}

func BenchmarkTreeClone(b *testing.B) {
	tr := buildBenchTree(b)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		clone := tr.Clone()
		if clone.Len() != tr.Len() {
			b.Fatalf("unexpected clone length: %d", clone.Len())
		}
	}
}

func BenchmarkConcurrentTreeGetParallel(b *testing.B) {
	tr := buildBenchConcurrentTree(b)
	mask := benchTreeNodes - 1

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = tr.Get((i & mask) + 1)
			i++
		}
	})
}

func BenchmarkConcurrentTreeDescendants(b *testing.B) {
	tr := buildBenchConcurrentTree(b)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = tr.Descendants(0)
	}
}

func BenchmarkTreeAddRootAddChild(b *testing.B) {
	const nodesPerRun = 1000

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		tr := tree.NewTree[int, int]()
		if err := tr.AddRoot(0, 0); err != nil {
			b.Fatalf("AddRoot() error = %v", err)
		}
		for j := 1; j <= nodesPerRun; j++ {
			parentID := (j - 1) / benchTreeBranching
			if err := tr.AddChild(parentID, j, j); err != nil {
				b.Fatalf("AddChild(%d, %d) error = %v", parentID, j, err)
			}
		}
	}
}

func BenchmarkTreeRemove(b *testing.B) {
	tr := buildBenchTree(b)
	leafID := benchTreeNodes

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		tr.Remove(leafID)
		if err := tr.AddChild((leafID-1)/benchTreeBranching, leafID, leafID); err != nil {
			b.Fatalf("AddChild(%d, %d) error = %v", (leafID-1)/benchTreeBranching, leafID, err)
		}
	}
}

func BenchmarkTreeMove(b *testing.B) {
	tr := buildBenchTree(b)
	// Move node 1 (child of 0) to be under node 2, then move back
	idToMove := 1
	fromParent := 0
	toParent := 2

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if err := tr.Move(idToMove, toParent); err != nil {
			b.Fatalf("Move(%d, %d) error = %v", idToMove, toParent, err)
		}
		if err := tr.Move(idToMove, fromParent); err != nil {
			b.Fatalf("Move(%d, %d) error = %v", idToMove, fromParent, err)
		}
	}
}

func BenchmarkTreeRangeDFS(b *testing.B) {
	tr := buildBenchTree(b)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		tr.RangeDFS(func(node *tree.Node[int, int]) bool {
			_ = node
			return true
		})
	}
}

func BenchmarkConcurrentTreeAddChildParallel(b *testing.B) {
	tr := tree.NewConcurrentTree[int, int]()
	if err := tr.AddRoot(0, 0); err != nil {
		b.Fatalf("AddRoot() error = %v", err)
	}
	// Pre-create branch roots so parallel goroutines can add to different parents
	for i := 1; i <= benchTreeBranching; i++ {
		if err := tr.AddChild(0, i, i); err != nil {
			b.Fatalf("AddChild(0, %d) error = %v", i, err)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		parentID := 1
		childID := 10000
		for pb.Next() {
			childID++
			if err := tr.AddChild(parentID, childID, childID); err != nil {
				b.Fatalf("AddChild(%d, %d) error = %v", parentID, childID, err)
			}
		}
	})
}
