# arcgo

`arcgo` is a modular Go toolkit for backend infrastructure.
It is package-oriented, supports incremental adoption, and allows inter-package composition.

English | [Chinese](./README_ZH.md)

> **Documentation**: Browse the [Hugo documentation site](./docs/) for a unified documentation experience.

## Packages

| Package | Description |
| --- | --- |
| `authx` | Opinionated security layer on Authboss + Casbin |
| `collectionx` | Generic collections and concurrent-safe structures |
| `configx` | Layered config loading and validation |
| `dbx` | Schema-first, generic-first ORM core on top of `database/sql` |
| `dix` | Strongly typed modular app framework built on top of `do` |
| `eventx` | In-memory typed event bus |
| `clientx` | Protocol-oriented client packages with shared conventions (HTTP/TCP/UDP) |
| `httpx` | Typed HTTP routing across adapters (optional submodules: `httpx/middleware`, `httpx/websocket`) |
| `kvx` | Redis/Valkey unified typed access framework |
| `logx` | Structured logging with zerolog + slog bridge |
| `observabilityx` | Optional observability facade with OTel/Prometheus adapters |

## Install

Quick start (`@latest`):

```bash
go get github.com/DaiYuANg/arcgo/dix@latest
go get github.com/DaiYuANg/arcgo/httpx@latest
go get github.com/DaiYuANg/arcgo/dbx@latest
go get github.com/DaiYuANg/arcgo/kvx@latest
```

Pinned version (recommended for production/CI):

```bash
go get github.com/DaiYuANg/arcgo/authx@v0.0.1
go get github.com/DaiYuANg/arcgo/clientx@v0.0.1
go get github.com/DaiYuANg/arcgo/collectionx@v0.0.1
go get github.com/DaiYuANg/arcgo/configx@v0.0.1
go get github.com/DaiYuANg/arcgo/dbx@v0.0.1
go get github.com/DaiYuANg/arcgo/dix@v0.0.1
go get github.com/DaiYuANg/arcgo/eventx@v0.0.1
go get github.com/DaiYuANg/arcgo/httpx@v0.0.1
go get github.com/DaiYuANg/arcgo/kvx@v0.0.1
go get github.com/DaiYuANg/arcgo/logx@v0.0.1
go get github.com/DaiYuANg/arcgo/observabilityx@v0.0.1
```

Optional submodules:

```bash
go get github.com/DaiYuANg/arcgo/dbx/migrate@v0.0.1
go get github.com/DaiYuANg/arcgo/httpx/middleware@v0.0.1
go get github.com/DaiYuANg/arcgo/httpx/websocket@v0.0.1
```

## How To Choose Quickly

- You need container/data helpers: start with `collectionx`.
- You need opinionated auth/authz abstraction on Authboss + Casbin: start with `authx`.
- You need config from `.env` + file + env vars: start with `configx`.
- You need an ORM with schema modeling, query DSL, and migrations: start with `dbx` (it also provides pure SQL templates via `dbx/sqltmplx`).
- You need a strongly typed modular application framework: start with `dix`.
- You need process-local pub/sub with typed payloads: start with `eventx`.
- You need protocol-oriented clients with shared retry/TLS/hooks conventions: start with `clientx`.
- You need unified typed HTTP handlers across frameworks: start with `httpx`.
- You need Redis/Valkey typed object mapping + repository-style access: start with `kvx`.
- You need structured logs and rotation: start with `logx`.
- You need optional telemetry abstraction (OTel/Prometheus): start with `observabilityx`.

## Typical Stack Combinations

- API service baseline: `httpx + configx + logx`
- Modular app baseline: `dix + configx + logx`
- Domain-events in a monolith: `eventx + logx`
- Data-heavy utility/internal libs: `collectionx + configx`
- Data layer (ORM + pure SQL helpers): `dbx` (includes `dbx/sqltmplx`)
- Redis/Valkey data access: `kvx + configx + logx`

## Build & QA

```bash
go tool task fmt
go tool task lint
go tool task test
go tool task check
```

## Docs Utility (Cross-Platform)

```bash
go run ./scripts/deploy-docs help
go run ./scripts/deploy-docs sync
go run ./scripts/deploy-docs build
go run ./scripts/deploy-docs serve
go run ./scripts/deploy-docs deploy
# optional:
# DOCS_REMOTE=origin DOCS_BRANCH=gh-pages go run ./scripts/deploy-docs deploy
```

## clientx Examples

```bash
go run ./clientx/examples/edge_http
go run ./clientx/examples/internal_rpc_tcp
go run ./clientx/examples/low_latency_udp
```

## Git Pre-Commit Hook

This repo uses `lefthook` (managed via `go tool`).

Install hooks (run once per clone):

```bash
go tool task git:hooks:install
```

Run hooks manually:

```bash
go tool task git:hooks:run
```

The `pre-commit` hook runs:

- `go tool task fmt`
- `go tool task lint`
```

## Notes

- Code comments are English-only.
- All documentation is available at the Hugo documentation site.
- Release policy: Before the Go "generic method" proposal is released/adopted, this library will not be formally published. After the proposal lands, expect potentially wide breaking updates. At this stage, we do not recommend using it in production.
