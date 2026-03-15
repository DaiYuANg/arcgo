package main

import (
	"fmt"

	"github.com/DaiYuANg/archgo/collectionx/interval"
	"github.com/DaiYuANg/archgo/collectionx/list"
	"github.com/DaiYuANg/archgo/collectionx/mapping"
	"github.com/DaiYuANg/archgo/collectionx/prefix"
	"github.com/DaiYuANg/archgo/collectionx/set"
	"github.com/DaiYuANg/archgo/collectionx/tree"
)

func main() {
	// Set
	users := set.NewSet[string]()
	users.Add("alice", "bob", "alice")
	fmt.Println("set:", users.Values(), "len:", users.Len())
	fmt.Println("set string:", users.String())

	// OrderedMap
	scores := mapping.NewOrderedMap[string, int]()
	scores.Set("alice", 95)
	scores.Set("bob", 88)
	scores.Set("alice", 99)
	fmt.Println("ordered map keys:", scores.Keys())
	fmt.Println("ordered map values:", scores.Values())

	// MultiMap
	tags := mapping.NewMultiMap[string, string]()
	tags.PutAll("backend", "go", "api", "infra")
	fmt.Println("multimap backend:", tags.Get("backend"))

	// Table
	matrix := mapping.NewTable[string, string, int]()
	matrix.Put("row1", "col1", 1)
	matrix.Put("row1", "col2", 2)
	matrix.Put("row2", "col1", 3)
	fmt.Println("table row1:", matrix.Row("row1"))
	fmt.Println("table col1:", matrix.Column("col1"))

	// List + Deque
	l := list.NewList[int](1, 3)
	_ = l.AddAt(1, 2)
	fmt.Println("list:", l.Values())

	dq := list.NewDeque[int]()
	dq.PushBack(2, 3)
	dq.PushFront(1)
	fmt.Println("deque:", dq.Values())

	// Trie
	tr := prefix.NewTrie[int]()
	tr.Put("user:1", 1)
	tr.Put("user:2", 2)
	tr.Put("order:9", 9)
	fmt.Println("trie prefix user:", tr.KeysWithPrefix("user:"))

	// RangeSet + RangeMap
	rs := interval.NewRangeSet[int]()
	rs.Add(1, 5)
	rs.Add(5, 8)
	fmt.Println("range set:", rs.Ranges())

	rm := interval.NewRangeMap[int, string]()
	rm.Put(0, 10, "A")
	rm.Put(3, 5, "B")
	v, _ := rm.Get(4)
	fmt.Println("range map get(4):", v)

	// Tree (parent-children)
	org := tree.NewTree[int, string]()
	_ = org.AddRoot(1, "CEO")
	_ = org.AddChild(1, 2, "CTO")
	_ = org.AddChild(1, 3, "CFO")
	_ = org.AddChild(2, 4, "Eng Manager")
	fmt.Println("tree roots:", len(org.Roots()), "descendants of 1:", len(org.Descendants(1)))

	corg := tree.NewConcurrentTree[int, string]()
	_ = corg.AddRoot(100, "ROOT")
	_ = corg.AddChild(100, 101, "CHILD")
	fmt.Println("concurrent tree len:", corg.Len())
}
