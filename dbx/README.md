# dbx

`dbx` is ArcGo's schema-first, generic-first ORM core on top of `database/sql`.

Current status:
- typed `Schema[E]`, `Column[E, T]`, and relation refs
- typed query DSL including aggregates, subqueries, CTE, `UNION ALL`, `CASE WHEN`, batch insert, `INSERT ... SELECT`, upsert, and `RETURNING`
- `StructMapper[E]` (schema-less pure DTO mapping) and `Mapper[E]` (schema-bound, for CRUD/relation load); field codecs; `RowsScanner` as read contract
- pure SQL execution through `dbx.SQL*` and `dbx/sqltmplx`
- Query DSL reuse: `Build` once, then `ExecBound` / `QueryAllBound` / `QueryCursorBound` / `QueryEachBound` in a loop; sqltmplx: reuse via `MustStatement`
- relation loading, schema planning, validation, SQL preview, auto-migrate, and migration runner
- runtime hooks, `slog` debug SQL logging, and benchmark coverage

Dialect abstraction (see [docs/content/docs/dbx/dialect](../docs/content/docs/dbx/dialect.md)):
- `Contract` / `Dialect` for bind var, quote, limit/offset
- `QueryFeaturesProvider` (optional) for upsert/returning—avoids name-based branching in render
- `SchemaDialect` (in dbx) for DDL/Inspect when using legacy migrate; Atlas uses dialect name for driver selection

Internal engines currently used by `dbx`:
- `scan`
- `Atlas`
- `goose`
- `hot`

Open: `dbx.Open(WithDriver, WithDSN, WithDialect, ApplyOptions(...))`—dbx manages the connection; no raw `*sql.DB` needed. Options: `WithLogger`, `WithHooks`, `WithDebug`; presets: `DefaultOptions()`, `ProductionOptions()`, `TestOptions()`.

Primary docs (Hugo):
- [docs/content/docs/dbx/](../docs/content/docs/dbx/) — Options, Observability, Dialect, examples

Schema vs Mapper dependency:
- `StructMapper` is schema-less: use for pure DTO mapping with arbitrary SQL (SQLList, SQLGet, QueryAll, etc.).
- `Mapper` depends on Schema: use for CRUD, relation load, repository when table structure is known.
- Read APIs accept `RowsScanner`; both StructMapper and Mapper implement it.

Runnable examples:
- `examples/dbx/basic`
- `examples/dbx/codec`
- `examples/dbx/mutation`
- `examples/dbx/query_advanced`
- `examples/dbx/relations`
- `examples/dbx/migration`
- `examples/dbx/pure_sql`
