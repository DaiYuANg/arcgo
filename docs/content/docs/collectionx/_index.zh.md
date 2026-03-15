---
title: 'collectionx'
linkTitle: 'collectionx'
description: '泛型集合与并发安全结构'
weight: 2
---

## collectionx

`collectionx` 为 Go 提供强类型的集合数据结构，包括并发安全变体和非标准结构（如 `MultiMap`、`Table`、`Trie`、区间结构）。

## 路线图

- 模块路线图：[collectionx roadmap](./roadmap)
- 全局路线图：[ArcGo roadmap](../roadmap)

## 为什么使用 collectionx

Go 标准容器是有意保持最小化。`collectionx` 专注于：

- 泛型、强类型 API
- 可预测的语义和明确的方法名
- 必要时可选的并发安全结构
- 受 Java 生态系统启发的有用非标准结构

## 包布局

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
  - `Tree`, `ConcurrentTree` (父子层级)

## 0 到 1 可运行示例

- 快速开始目录：[collectionx/examples/quickstart](https://github.com/DaiYuANg/archgo/tree/main/collectionx/examples/quickstart)
- 从仓库根目录运行：

```bash
go run ./collectionx/examples/quickstart
```

## 使用场景

### 1) 使用 `Set` 快速去重

```go
s := set.NewSet[string]()
s.Add("A", "A", "B")
fmt.Println(s.Len()) // 2
fmt.Println(s.Contains("B"))
```

### 2) 使用 `OrderedSet` / `OrderedMap` 保留插入顺序

```go
os := set.NewOrderedSet[int]()
os.Add(3, 1, 3, 2)
fmt.Println(os.Values()) // [3 1 2]

om := mapping.NewOrderedMap[string, int]()
om.Set("x", 1)
om.Set("y", 2)
om.Set("x", 9) // 更新不改变顺序
fmt.Println(om.Keys())   // [x y]
fmt.Println(om.Values()) // [9 2]
```

### 3) 一键多值使用 `MultiMap`

```go
mm := mapping.NewMultiMap[string, int]()
mm.PutAll("tag", 1, 2, 3)
fmt.Println(mm.Get("tag"))        // [1 2 3]
fmt.Println(mm.ValueCount())       // 3
removed := mm.DeleteValueIf("tag", func(v int) bool { return v%2 == 0 })
fmt.Println(removed, mm.Get("tag")) // 1 [1 3]
```

### 4) 使用 `Table` 进行 2D 索引（Guava 风格）

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

### 5) 使用 `Trie` 进行前缀查找

```go
tr := prefix.NewTrie[int]()
tr.Put("user:1", 1)
tr.Put("user:2", 2)
tr.Put("order:9", 9)

fmt.Println(tr.KeysWithPrefix("user:")) // [user:1 user:2]
```

### 6) 使用 `list` 包进行队列和缓冲

```go
dq := list.NewDeque[int]()
dq.PushBack(1, 2)
dq.PushFront(0)
fmt.Println(dq.Values()) // [0 1 2]

rb := list.NewRingBuffer[int](2)
_ = rb.Push(1)
_ = rb.Push(2)
evicted := rb.Push(3) // 驱逐 1
fmt.Println(evicted)
```

### 7) 区间操作

```go
rs := interval.NewRangeSet[int]()
rs.Add(1, 5)
rs.Add(5, 8) // 相邻区间会被合并
fmt.Println(rs.Ranges())

rm := interval.NewRangeMap[int, string]()
rm.Put(0, 10, "A")
rm.Put(3, 5, "B") // 重叠覆盖
v, _ := rm.Get(4)
fmt.Println(v) // B
```

### 8) 使用 `Tree` 进行父子层级

```go
org := tree.NewTree[int, string]()
_ = org.AddRoot(1, "CEO")
_ = org.AddChild(1, 2, "CTO")
_ = org.AddChild(2, 3, "Platform Lead")

parent, _ := org.Parent(3)
fmt.Println(parent.ID())          // 2
fmt.Println(len(org.Descendants(1))) // 2
```

## 并发安全类型：何时使用

仅在跨 goroutine 共享访问时使用并发变体：

- `ConcurrentSet`
- `ConcurrentMap`
- `ConcurrentMultiMap`
- `ConcurrentTable`
- `ConcurrentList`
- `ConcurrentTree`

对于单 goroutine 或外部同步的工作流，非并发类型通常更快。

## API 风格说明

- 大多数 `All/Values/Row/Column` 风格的方法返回副本/快照，以避免意外修改泄漏。
- `GetOption` 方法使用 `mo.Option` 进行可空风格读取。
- 许多结构支持零值行为，但仍建议使用构造函数以提高清晰度。

## JSON 和日志辅助

大多数结构提供：

- `ToJSON() ([]byte, error)` 用于快速序列化
- `MarshalJSON() ([]byte, error)` 以便 `json.Marshal(x)` 直接工作
- `String() string` 用于日志友好输出

示例：

```go
s := set.NewSet[string]("a", "b")
raw, _ := s.ToJSON()
fmt.Println(string(raw))  // ["a","b"]
fmt.Println(s.String())   // ["a","b"]

payload, _ := json.Marshal(s) // 通过 MarshalJSON 实现相同行为
_ = payload
```

## 基准测试

```bash
go test ./collectionx/... -run ^$ -bench . -benchmem
```

你可以针对一个包：

```bash
go test ./collectionx/mapping -run ^$ -bench . -benchmem
go test ./collectionx/prefix -run ^$ -bench Trie -benchmem
```

## 实用技巧

- 当你手动使用嵌套 map 时，优先使用 `Table`。
- 当结果顺序很重要时（序列化、确定性测试），优先使用 `OrderedMap/OrderedSet`。
- 对于大量前缀搜索，优先使用 `Trie` 而不是重复线性扫描。
- 当计数频率是主要操作时，优先使用 `MultiSet`。
- 当你的模型是自然的父子结构（组织图、类别、菜单树）时，优先使用 `Tree`。

## 常见问题

### 我应该总是使用并发变体吗？

不。仅在多个 goroutine 共享同一结构实例时使用并发变体。
如果访问是单线程的或已经外部同步，非并发变体更简单更快。

### 返回的切片/map 可以安全修改吗？

对于大多数快照风格的 API（`Values`、`All`、`Row`、`Column` 等），返回值是副本。
修改返回的对象通常不会修改内部状态。

### 为什么 `OrderedMap` 在更新时保持旧的插入顺序？

它有意表现得像其他生态系统中的插入顺序 map：更新改变值，不改变顺序。

### `RangeSet` 如何处理相邻区间？

对于半开区间，相邻区间会被标准化和合并（例如 `[1,5)` + `[5,8)`）。

## 故障排除

### `Publish` 风格代码需要确定性顺序但 map 支持的结构看起来随机

`Map`、`Set` 和类似的 hash 支持结构不保证迭代顺序。
如果需要确定性顺序，使用 `OrderedMap` / `OrderedSet`。

### `Trie.KeysWithPrefix` 分配超出预期

前缀收集返回新切片并遍历匹配的子树。
对于热路径：

- 尽可能缩小前缀。
- 可能时使用 `RangePrefix` 回调风格。
- 避免在每次请求时转换大型快照。

### 长时间运行后 `MultiMap` 或 `Table` 内存增长

常见原因是无界键增长和缺少清理路径。
使用 `Delete`、`DeleteColumn`、`DeleteRow`、`DeleteValueIf` 或基于业务生命周期的定期重置。

## 反模式

- 默认在所有地方使用 `Concurrent*` 结构。
- 在测试或业务逻辑中依赖 hash-map 迭代顺序。
- 将快照返回 API 视为实时视图并期望原地同步。
- 每次请求构建巨大的临时集合而不是增量更新。
- 仅用于点查找时使用 `RangeMap`；如果区间语义不必要，使用普通 map。
