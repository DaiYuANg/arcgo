---
title: 'collectionx'
linkTitle: 'collectionx'
description: 'Generic Collections and Concurrency-Safe Structures'
weight: 2
---

## collectionx

`collectionx` provides strongly typed collection data structures for Go, including concurrent variants and non-standard structures like `MultiMap`, `Table`, `Trie`, and interval structures.

## Roadmap

- Module roadmap: [collectionx roadmap](./roadmap)
- Global roadmap: [ArcGo roadmap](../roadmap)

## Why Use collectionx

Go standard containers are intentionally minimal. `collectionx` focuses on:

- Generic, strongly typed API
- Predictable semantics and explicit method names
- Optional concurrency-safe structures when needed
- Useful non-standard structures inspired by Java ecosystem

## Package Layout

- `collectionx/set`
  - `Set`, `ConcurrentSet`, `MultiSet`, `OrderedSet`
- `collectionx/mapping`
  - `Map`, `ConcurrentMap`, `BiMap`, `OrderedMap`, `MultiMap`, `Table`
- `collectionx/list`
  - `List`, `ConcurrentList`, `Deque`, `RingBuffer`, `PriorityQueue`
- `collectionx/interval`
  - `Range`, `RangeSet`, `RangeMap`
- `collectionx/prefix`
  - `Trie` / `PrefixMap`
- `collectionx/tree`
  - `Tree`, `ConcurrentTree` (parent-child hierarchy)

## 0 to 1 Runnable Example

- Quick start directory: [collectionx/examples/quickstart](https://github.com/DaiYuANg/archgo/tree/main/collectionx/examples/quickstart)
- Run from repository root:

```bash
go run ./collectionx/examples/quickstart
```

## Use Cases

### 1) Quick Deduplication with `Set`

```go
s := set.NewSet[string]()
s.Add("A", "A", "B")
fmt.Println(s.Len()) // 2
fmt.Println(s.Contains("B"))
```

### 2) Preserve Insertion Order with `OrderedSet` / `OrderedMap`

```go
os := set.NewOrderedSet[int]()
os.Add(3, 1, 3, 2)
fmt.Println(os.Values()) // [3 1 2]

om := mapping.NewOrderedMap[string, int]()
om.Set("x", 1)
om.Set("y", 2)
om.Set("x", 9) // Update doesn't change order
fmt.Println(om.Keys())   // [x y]
fmt.Println(om.Values()) // [9 2]
```

### 3) One-to-Many with `MultiMap`

```go
mm := mapping.NewMultiMap[string, int]()
mm.PutAll("tag", 1, 2, 3)
fmt.Println(mm.Get("tag"))        // [1 2 3]
fmt.Println(mm.ValueCount())       // 3
removed := mm.DeleteValueIf("tag", func(v int) bool { return v%2 == 0 })
fmt.Println(removed, mm.Get("tag")) // 1 [1 3]
```

### 4) 2D Indexing with `Table` (Guava Style)

```go
t := mapping.NewTable[string, string, int]()
t.Put("r1", "c1", 10)
t.Put("r1", "c2", 20)
t.Put("r2", "c1", 30)

v, ok := t.Get("r1", "c2")
fmt.Println(v, ok) // 20 true
fmt.Println(t.Row("r1"))
fmt.Println(t.Column("c1"))
```

### 5) Prefix Lookup with `Trie`

```go
tr := prefix.NewTrie[int]()
tr.Put("user:1", 1)
tr.Put("user:2", 2)
tr.Put("order:9", 9)

fmt.Println(tr.KeysWithPrefix("user:")) // [user:1 user:2]
```

### 6) Queue and Buffer with `list` Package

```go
dq := list.NewDeque[int]()
dq.PushBack(1, 2)
dq.PushFront(0)
fmt.Println(dq.Values()) // [0 1 2]

rb := list.NewRingBuffer[int](2)
_ = rb.Push(1)
_ = rb.Push(2)
evicted := rb.Push(3) // Evicts 1
fmt.Println(evicted)
```

### 7) Interval Operations

```go
rs := interval.NewRangeSet[int]()
rs.Add(1, 5)
rs.Add(5, 8) // Adjacent ranges are merged
fmt.Println(rs.Ranges())

rm := interval.NewRangeMap[int, string]()
rm.Put(0, 10, "A")
rm.Put(3, 5, "B") // Overlapping coverage
v, _ := rm.Get(4)
fmt.Println(v) // B
```

### 8) Parent-Child Hierarchy with `Tree`

```go
org := tree.NewTree[int, string]()
_ = org.AddRoot(1, "CEO")
_ = org.AddChild(1, 2, "CTO")
_ = org.AddChild(2, 3, "Platform Lead")

parent, _ := org.Parent(3)
fmt.Println(parent.ID())          // 2
fmt.Println(len(org.Descendants(1))) // 2
```

## Concurrency-Safe Types: When to Use

Use concurrent variants only when access is shared across goroutines:

- `ConcurrentSet`
- `ConcurrentMap`
- `ConcurrentMultiMap`
- `ConcurrentTable`
- `ConcurrentList`
- `ConcurrentTree`

For single-goroutine or externally synchronized workflows, non-concurrent types are typically faster.

## API Style Notes

- Most `All/Values/Row/Column` style methods return copies/snapshots to avoid accidental mutation leakage.
- `GetOption` methods use `mo.Option` for nullable-style reads.
- Many structures support zero-value behavior but constructors are still recommended for clarity.

## JSON and Logging Helpers

Most structures provide:

- `ToJSON() ([]byte, error)` for quick serialization
- `MarshalJSON() ([]byte, error)` for `json.Marshal(x)` to work directly
- `String() string` for log-friendly output

Example:

```go
s := set.NewSet[string]("a", "b")
raw, _ := s.ToJSON()
fmt.Println(string(raw))  // ["a","b"]
fmt.Println(s.String())   // ["a","b"]

payload, _ := json.Marshal(s) // Same behavior via MarshalJSON
_ = payload
```

## Benchmarks

```bash
go test ./collectionx/... -run ^$ -bench . -benchmem
```

You can target a single package:

```bash
go test ./collectionx/mapping -run ^$ -bench . -benchmem
go test ./collectionx/prefix -run ^$ -bench Trie -benchmem
```

## Practical Tips

- Prefer `Table` when you're manually using nested maps.
- Prefer `OrderedMap/OrderedSet` when result order matters (serialization, deterministic tests).
- Prefer `Trie` for large prefix searches instead of repeated linear scans.
- Prefer `MultiSet` when count frequency is the primary operation.
- Prefer `Tree` when your model is naturally parent-child (org charts, categories, menu trees).

## FAQ

### Should I always use concurrent variants?

No. Use concurrent variants only when multiple goroutines share the same structure instance.
If access is single-threaded or already externally synchronized, non-concurrent variants are simpler and faster.

### Are returned slices/maps safe to modify?

For most snapshot-style APIs (`Values`, `All`, `Row`, `Column`, etc.), return values are copies.
Modifying returned objects typically doesn't modify internal state.

### Why does `OrderedMap` keep old insertion order on update?

It's intentionally designed to behave like insertion-order maps in other ecosystems: updates change values, not order.

### How does `RangeSet` handle adjacent ranges?

For half-open ranges, adjacent ranges are normalized and merged (e.g., `[1,5)` + `[5,8)`).

## Troubleshooting

### `Publish` style code needs deterministic order but map-backed structures seem random

`Map`, `Set`, and similar hash-backed structures don't guarantee iteration order.
Use `OrderedMap` / `OrderedSet` for deterministic order.

### `Trie.KeysWithPrefix` allocates more than expected

Prefix collection returns new slices and traverses matching subtrees.
For hot paths:

- Narrow the prefix when possible.
- Use `RangePrefix` callback style when available.
- Avoid building large temporary snapshots on every request.

### `MultiMap` or `Table` memory growth after long runs

Common causes are unbounded key growth and missing cleanup paths.
Use `Delete`, `DeleteColumn`, `DeleteRow`, `DeleteValueIf`, or periodic resets based on business lifecycle.

## Anti-Patterns

- Using `Concurrent*` structures everywhere by default.
- Relying on hash-map iteration order in tests or business logic.
- Treating snapshot-returning APIs as live views and expecting in-place synchronization.
- Building huge temporary collections per request instead of incremental updates.
- Using `RangeMap` only for point lookups; use plain maps if interval semantics aren't necessary.
