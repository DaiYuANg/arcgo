# dbx Benchmarks

Run: `go test ./dbx -run '^$' -bench . -benchmem -count=3`

## Bottleneck Summary (real sqlite, arm64)

| Benchmark | ns/op | allocs | Notes |
|-----------|-------|--------|-------|
| ValidateSchemasSQLiteAtlasMatched | ~257k | 740 | Schema diff + Atlas; heaviest |
| PlanSchemaChangesSQLiteAtlasEmpty | ~59k | 413 | Atlas schema planning |
| LoadManyToMany | ~35k | 160 | 3+ queries, join table scan |
| LoadBelongsTo | ~17k | 68 | 2 queries (parent + children) |
| LoadHasMany | ~12k | 97 | Batch relation load |
| QueryAllStructMapper | ~8k | 67 | Full query + scan |
| SQLList / SQLGet | ~5k | 34–44 | Statement + scan |
| BuildInsertUpsertReturning | ~2.3k | 47–51 | Query build |
| MapperInsertAssignments | ~800 | 11 | Assignment build |
| NewStructMapperCached | ~32 | 1 | Metadata cache hit |

## Optimization Priorities

1. **ValidateSchemas / PlanSchemaChanges** — Atlas + schema diff; consider caching compiled schema or reducing allocs.
2. **Relation loading** — Multiple round-trips; batch or reduce queries where possible.
3. **Query + scan path** — Mapper scan, column binding; already optimized with scan plan cache.
4. **Build* (render)** — SQL building; moderate allocs, acceptable for non-hot path.

## Notes

- Benchmarks use real in-memory sqlite (`:memory:`) for realistic I/O.
- Hot paths (mapper scan, bind) are kept allocation-conscious.
- Schema/Atlas operations are not hot in typical request handling.
