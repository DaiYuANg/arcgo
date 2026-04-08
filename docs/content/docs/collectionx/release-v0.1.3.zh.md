---
title: 'collectionx v0.1.3'
linkTitle: 'release v0.1.3'
description: 'collectionx 的派生视图缓存与序列化性能优化'
weight: 41
---

`collectionx v0.1.3` 是一个以性能为主的补丁版本。公开 API 没有变化，但核心集合里那些重复快照与重复序列化的路径现在明显更便宜了。

## 重点更新

- 为高频快照式方法增加了派生缓存，例如：
  - `OrderedMap.Values()`
  - `OrderedSet.Values()`
  - `Table.ColumnKeys()` / `ConcurrentTable.ColumnKeys()`
  - `RangeSet.Ranges()`
  - `RangeMap.Entries()`
- 为常见结构及其并发变体增加了 `ToJSON()` / `String()` 缓存：
  - `Map`、`List`、`Set`、`OrderedSet`
  - `OrderedMap`、`MultiMap`、`Table`
  - `ConcurrentList`、`ConcurrentSet`、`ConcurrentMap`、`ConcurrentMultiMap`、`ConcurrentTable`
- 简化了 `Trie` 的前缀遍历路径构造，降低了 `Trie.KeysWithPrefix`、`Trie.RangePrefix`、`Trie.ValuesWithPrefix` 的分配压力。

## 语义保证

- 现有 API 与方法名都保持不变。
- 快照式方法依然返回独立副本；修改返回的 slice 或 JSON bytes 不会污染内部缓存。
- 缓存失效跟随 mutation 路径，所以收益主要体现在 read-heavy 场景，写入语义不变。

## 代表性 benchmark

这一轮比较明显的结果：

- `RootMapToJSON`: 约 `251399 ns/op` -> `1369 ns/op`
- `RootSetToJSON`: 约 `27241 ns/op` -> `710 ns/op`
- `RootListToJSON`: 约 `18267 ns/op` -> `849 ns/op`
- `OrderedMapValues`: 约 `44496 ns/op` -> `5380 ns/op`
- `OrderedSetValues`: 约 `7042 ns/op` -> `3639 ns/op`
- `TrieRangePrefix`: 约 `36862 ns/op` -> `29676 ns/op`

## 验证

已通过：

```bash
go test ./collectionx/...
go test -run ^$ -bench "Root(MapToJSON|SetToJSON|ListToJSON)|Ordered(SetValues|MapValues)|OrderedMapToJSON|MultiMapToJSON|TableToJSON|Concurrent(MapToJSON|TableToJSON)" -benchmem ./collectionx ./collectionx/mapping ./collectionx/set
go test -run ^$ -bench "RangeSetRanges|RangeMapEntries" -benchmem ./collectionx/interval
go test -run ^$ -bench "Trie(KeysWithPrefix|RangePrefix|ValuesWithPrefix)" -benchmem ./collectionx/prefix
```
