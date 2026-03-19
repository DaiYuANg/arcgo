# dbx examples

This module demonstrates the current `dbx` core API with a real SQLite driver.

Examples:
- `go run ./examples/dbx/basic`
- `go run ./examples/dbx/mutation`
- `go run ./examples/dbx/relations`
- `go run ./examples/dbx/migration`
- `go run ./examples/dbx/pure_sql`

Coverage:
- schema as the single database metadata source
- aggregate queries, subqueries, batch insert, insert-select, upsert, returning
- mapper-based entity scan
- projection queries
- relation join helpers
- pure SQL execution via `sqltmplx` registry + `dbx.SQL*`
- conservative auto-migrate / validate / migration plan
- `slog` debug SQL logging and hooks
