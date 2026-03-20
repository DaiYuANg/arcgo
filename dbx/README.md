# dbx

`dbx` is ArcGo's schema-first, generic-first ORM core on top of `database/sql`.

Current status:
- typed `Schema[E]`, `Column[E, T]`, and relation refs
- typed query DSL including aggregates, subqueries, CTE, `UNION ALL`, `CASE WHEN`, batch insert, `INSERT ... SELECT`, upsert, and `RETURNING`
- `Mapper[E]` / `StructMapper[E]` with field codecs and pure SQL reads
- pure SQL execution through `dbx.SQL*` and `dbx/sqltmplx`
- relation loading, schema planning, validation, SQL preview, auto-migrate, and migration runner
- runtime hooks, `slog` debug SQL logging, and benchmark coverage

Internal engines currently used by `dbx`:
- `scan`
- `Atlas`
- `goose`
- `hot`

Primary docs:
- `docs/content/docs/dbx/_index.md`
- `docs/content/docs/dbx/examples.md`

Runnable examples:
- `examples/dbx/basic`
- `examples/dbx/codec`
- `examples/dbx/mutation`
- `examples/dbx/query_advanced`
- `examples/dbx/relations`
- `examples/dbx/migration`
- `examples/dbx/pure_sql`
