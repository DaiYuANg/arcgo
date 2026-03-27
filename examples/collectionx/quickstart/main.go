// Package main demonstrates the collectionx quickstart examples.
package main

import (
	"fmt"
	"log"

	"github.com/DaiYuANg/arcgo/collectionx/interval"
	"github.com/DaiYuANg/arcgo/collectionx/list"
	"github.com/DaiYuANg/arcgo/collectionx/mapping"
	"github.com/DaiYuANg/arcgo/collectionx/prefix"
	"github.com/DaiYuANg/arcgo/collectionx/set"
	"github.com/DaiYuANg/arcgo/collectionx/tree"
)

func main() {
	showSetExample()
	showOrderedMapExample()
	showMultiMapExample()
	showTableExample()
	showListAndDequeExample()
	showTrieExample()
	showIntervalExample()
	showTreeExample()
}

func showSetExample() {
	users := set.NewSet[string]()
	users.Add("alice", "bob", "alice")
	printLine("set:", users.Values(), "len:", users.Len())
	printLine("set string:", users.String())
}

func showOrderedMapExample() {
	scores := mapping.NewOrderedMap[string, int]()
	scores.Set("alice", 95)
	scores.Set("bob", 88)
	scores.Set("alice", 99)
	printLine("ordered map keys:", scores.Keys())
	printLine("ordered map values:", scores.Values())
}

func showMultiMapExample() {
	tags := mapping.NewMultiMap[string, string]()
	tags.PutAll("backend", "go", "api", "infra")
	printLine("multimap backend:", tags.Get("backend"))
}

func showTableExample() {
	matrix := mapping.NewTable[string, string, int]()
	matrix.Put("row1", "col1", 1)
	matrix.Put("row1", "col2", 2)
	matrix.Put("row2", "col1", 3)
	printLine("table row1:", matrix.Row("row1"))
	printLine("table col1:", matrix.Column("col1"))
}

func showListAndDequeExample() {
	l := list.NewList[int](1, 3)
	_ = l.AddAt(1, 2)
	printLine("list:", l.Values())

	dq := list.NewDeque[int]()
	dq.PushBack(2, 3)
	dq.PushFront(1)
	printLine("deque:", dq.Values())
}

func showTrieExample() {
	tr := prefix.NewTrie[int]()
	tr.Put("user:1", 1)
	tr.Put("user:2", 2)
	tr.Put("order:9", 9)
	printLine("trie prefix user:", tr.KeysWithPrefix("user:"))
}

func showIntervalExample() {
	rs := interval.NewRangeSet[int]()
	rs.Add(1, 5)
	rs.Add(5, 8)
	printLine("range set:", rs.Ranges())

	rm := interval.NewRangeMap[int, string]()
	rm.Put(0, 10, "A")
	rm.Put(3, 5, "B")
	v, _ := rm.Get(4)
	printLine("range map get(4):", v)
}

func showTreeExample() {
	org := tree.NewTree[int, string]()
	must(org.AddRoot(1, "CEO"))
	must(org.AddChild(1, 2, "CTO"))
	must(org.AddChild(1, 3, "CFO"))
	must(org.AddChild(2, 4, "Eng Manager"))
	printLine("tree roots:", len(org.Roots()), "descendants of 1:", len(org.Descendants(1)))

	corg := tree.NewConcurrentTree[int, string]()
	must(corg.AddRoot(100, "ROOT"))
	must(corg.AddChild(100, 101, "CHILD"))
	printLine("concurrent tree len:", corg.Len())
}

func printLine(values ...any) {
	if _, err := fmt.Println(values...); err != nil {
		log.Printf("print quickstart line: %v", err)
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
