---
title: 'collectionx'
linkTitle: 'collectionx'
description: '泛型集合与并发安全结构'
weight: 2
---

## collectionx

`collectionx` 为 Go 提供强类型集合数据结构，包含并发安全变体以及 `MultiMap`、`Table`、`Trie`、区间类型等非标准但常用的结构。

## 当前能力

- **泛型优先** API，方法名清晰、语义可预期。
- **按需并发**：在跨 goroutine 共享实例时使用 `Concurrent*` 变体。
- **实用扩展结构**：有序映射、多重映射、二维 `Table`、前缀 `Trie`、区间映射、父子 `Tree` 等。

## 包结构

- `github.com/DaiYuANg/arcgo/collectionx/set` — `Set`、`ConcurrentSet`、`MultiSet`、`OrderedSet`
- `github.com/DaiYuANg/arcgo/collectionx/mapping` — `Map`、`ConcurrentMap`、`BiMap`、`OrderedMap`、`MultiMap`、`Table`
- `github.com/DaiYuANg/arcgo/collectionx/list` — `List`、`ConcurrentList`、`Deque`、`RingBuffer`、`PriorityQueue`
- `github.com/DaiYuANg/arcgo/collectionx/interval` — `Range`、`RangeSet`、`RangeMap`
- `github.com/DaiYuANg/arcgo/collectionx/prefix` — `Trie` / `PrefixMap`
- `github.com/DaiYuANg/arcgo/collectionx/tree` — `Tree`、`ConcurrentTree`

## 文档导航

- 第一个可运行程序（`Set` + `OrderedMap`）：[快速开始](./getting-started)
- 集合、有序结构、`MultiMap`、`Table`、JSON：[映射、集合与表](./mapping-recipes)
- 列表、区间、Trie、树：[列表与结构化数据](./structured-data)
- 版本说明：[collectionx v0.1.3](./release-v0.1.3)
- 版本说明：[collectionx v0.1.2](./release-v0.1.2)

## 安装 / 导入

```bash
go get github.com/DaiYuANg/arcgo/collectionx@latest
```

按需 import **子包**（例如 `collectionx/set`、`collectionx/mapping`）。

## 为何使用 collectionx

标准库容器刻意保持精简。`collectionx` 强调泛型与强类型、在需要时明确顺序语义，并在各结构间保持一致的工程化约定。

## 并发安全类型

仅在**同一实例被多 goroutine 共享**时使用并发变体，例如：

- `ConcurrentSet`、`ConcurrentMap`、`ConcurrentMultiMap`、`ConcurrentTable`、`ConcurrentList`、`ConcurrentTree`

单 goroutine 或已有外部同步时，优先非并发类型以降低开销。

## API 风格说明

- 多数 `Values` / `All` / `Row` / `Column` 类方法返回**快照**，避免误改内部状态。
- 部分 `GetOption` 使用 `mo.Option` 表达可空读取。
- 即使零值可用，仍建议使用构造函数以增强可读性。

## JSON 与日志

多数结构提供 `ToJSON`、`MarshalJSON`（配合 `json.Marshal`）与 `String()`。示例见 [映射、集合与表](./mapping-recipes)。

## Benchmark

```bash
go test ./collectionx/... -run ^$ -bench . -benchmem
```

单包：

```bash
go test ./collectionx/mapping -run ^$ -bench . -benchmem
go test ./collectionx/prefix -run ^$ -bench Trie -benchmem
```

## 实践建议

- 二维键优先用 `Table`，而不是手写嵌套 map。
- 需要稳定迭代顺序（测试、API、序列化）时用 `OrderedMap` / `OrderedSet`。
- 大量前缀查询用 `Trie`，避免对字符串键反复线性扫描。
- 以频次为主用 `MultiSet`。
- 自然父子关系用 `Tree`（组织架构、类目、菜单等）。

## 常见问题

**是否应默认使用 `Concurrent*`？**  
否。仅在没有外部同步且多 goroutine 共享同一实例时使用。

**返回的 slice 能否随意改？**  
快照 API 返回副本；修改返回值通常不影响内部状态。

**为何 `OrderedMap` 更新 value 不改变键顺序？**  
有意为之：更新只改值，与常见「插入顺序」语义一致。

**`RangeSet` 如何合并区间？**  
半开区间 `[start, end)` 会规范化；相邻区间会合并（如 `[1,5)` + `[5,8)`）。

## 疑难排查

- **迭代顺序不确定** — 哈希 `Map`/`Set` 无序；需要顺序请用 `OrderedMap`/`OrderedSet`。
- **`Trie.KeysWithPrefix` 分配** — 会构造新 slice；可缩小前缀、优先 `RangePrefix`，热路径避免每次拉全量快照。
- **`MultiMap`/`Table` 内存膨胀** — 使用 `Delete`、`DeleteRow`、`DeleteColumn`、`DeleteValueIf` 或按业务周期清理。

## 反模式

- 默认处处 `Concurrent*`。
- 在测试或业务中依赖哈希 map 的迭代顺序。
- 把快照 API 当成可同步的「实时视图」。
- 仅需点查时用 `RangeMap` 过度建模。

## 集成指南

- **configx**：加载配置后先归一化为强类型 map/list，再绑定服务。
- **clientx** / **kvx**：用集合工具整理缓存与索引，避免临时容器散落。
- **dix**：由模块 provider 提供集合实例，避免包级全局可变状态。

## 生产注意

- 选满足不变量的最简结构。
- 在 API 边界写清顺序语义（`OrderedMap` 与哈希 map）。
- 对并发类型也要明确所有权与生命周期，即使内部带锁。
