# dbx examples

This module demonstrates the current `dbx` core API with a real SQLite driver.

Examples:
- `go run ./examples/dbx/basic`
- `go run ./examples/dbx/codec`
- `go run ./examples/dbx/mutation`
- `go run ./examples/dbx/query_advanced`
- `go run ./examples/dbx/relations`
- `go run ./examples/dbx/migration`
- `go run ./examples/dbx/pure_sql`
- `go run ./examples/dbx/id_generation`

Coverage:
- schema as the single database metadata source
- aggregate queries, subqueries, batch insert, insert-select, upsert, returning
- advanced querydsl features such as `WITH`, `UNION ALL`, and `CASE WHEN`
- mapper-based entity scan and field codecs
- built-in codecs such as `json` / `text` / `unix_milli_time`
- scoped custom codecs via `dbx.WithMapperCodecs(...)`
- projection queries
- relation join helpers and relation loading
- pure SQL execution via `sqltmplx` registry + `dbx.SQL*`
- typed ID generation strategies via marker types (`NewIDColumn`)
- conservative auto-migrate / validate / migration runner
- `slog` debug SQL logging and hooks
