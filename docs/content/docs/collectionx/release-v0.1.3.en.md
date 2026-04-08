---
title: 'collectionx v0.1.3'
linkTitle: 'release v0.1.3'
description: 'Derived-view caching and serialization performance improvements across collectionx'
weight: 41
---

`collectionx v0.1.3` is a performance-focused patch release. Public APIs remain unchanged, but repeated snapshot and serialization paths are now significantly cheaper across the core collection packages.

## Highlights

- Added derived-view caching for repeated snapshot-style methods such as:
  - `OrderedMap.Values()`
  - `OrderedSet.Values()`
  - `Table.ColumnKeys()` / `ConcurrentTable.ColumnKeys()`
  - `RangeSet.Ranges()`
  - `RangeMap.Entries()`
- Added cached `ToJSON()` / `String()` paths for common structures and their concurrent variants:
  - `Map`, `List`, `Set`, `OrderedSet`
  - `OrderedMap`, `MultiMap`, `Table`
  - `ConcurrentList`, `ConcurrentSet`, `ConcurrentMap`, `ConcurrentMultiMap`, `ConcurrentTable`
- Reduced allocation pressure in `Trie.KeysWithPrefix`, `Trie.RangePrefix`, and `Trie.ValuesWithPrefix` by simplifying prefix path traversal.

## Semantics

- Existing APIs and method names are unchanged.
- Snapshot-style methods still return owned copies; mutating returned slices or JSON bytes does not affect internal caches.
- Cache invalidation is tied to mutation paths, so read-heavy workloads benefit while write semantics stay the same.

## Notable benchmark impact

Representative results from this release cycle:

- `RootMapToJSON`: about `251399 ns/op` -> `1369 ns/op`
- `RootSetToJSON`: about `27241 ns/op` -> `710 ns/op`
- `RootListToJSON`: about `18267 ns/op` -> `849 ns/op`
- `OrderedMapValues`: about `44496 ns/op` -> `5380 ns/op`
- `OrderedSetValues`: about `7042 ns/op` -> `3639 ns/op`
- `TrieRangePrefix`: about `36862 ns/op` -> `29676 ns/op`

## Validation

Verified with:

```bash
go test ./collectionx/...
go test -run ^$ -bench "Root(MapToJSON|SetToJSON|ListToJSON)|Ordered(SetValues|MapValues)|OrderedMapToJSON|MultiMapToJSON|TableToJSON|Concurrent(MapToJSON|TableToJSON)" -benchmem ./collectionx ./collectionx/mapping ./collectionx/set
go test -run ^$ -bench "RangeSetRanges|RangeMapEntries" -benchmem ./collectionx/interval
go test -run ^$ -bench "Trie(KeysWithPrefix|RangePrefix|ValuesWithPrefix)" -benchmem ./collectionx/prefix
```
