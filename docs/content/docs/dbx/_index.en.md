---
title: 'dbx'
linkTitle: 'dbx'
description: 'Type-safe generic-first ORM core on top of database/sql'
weight: 7
---

## dbx

`dbx` is a schema-first, generic-first ORM core built on top of `database/sql`.
It keeps database metadata in `Schema[E]`, keeps entities as data carriers, and currently covers these core pipelines:

- typed schema and relation modeling
- typed query DSL and SQL rendering
- mapper / struct-mapper reads with codec support
- pure SQL execution through `sqltmplx` statements
- relation loading for `BelongsTo` / `HasOne` / `HasMany` / `ManyToMany`
- schema planning, validation, conservative auto-migrate, and migration runner
- runtime logging, hooks, transactions, and benchmark coverage

## Current Status

The current `dbx` implementation includes:

- `Schema[E]` as the single source of database metadata
- `Column[E, T]` and typed relation refs
- query DSL support for aggregates, subqueries, CTE, `UNION ALL`, `CASE WHEN`, batch insert, `INSERT ... SELECT`, upsert, and `RETURNING`
- `StructMapper[E]` (schema-less pure DTO mapping) and `Mapper[E]` (schema-bound, for CRUD/relation load); `RowsScanner` as read contract
- field codecs with built-in `json`, `text`, `unix_time`, `unix_milli_time`, `unix_nano_time`, `rfc3339_time`, and `rfc3339nano_time`
- scoped custom codecs via `dbx.WithMapperCodecs(...)`
- `DB.SQL()` / `Tx.SQL()` as the pure SQL execution entry
- relation loading APIs and relation-aware join helpers
- `PlanSchemaChanges`, `ValidateSchemas`, `AutoMigrate`, and `MigrationPlan.SQLPreview()`
- `dbx/migrate` runner for Go migrations and Flyway-style SQL migrations
- benchmark coverage across mapper, query builder, SQL executor, relation loading, schema planning, and migrate runner

## Internal Engines

The public API remains `dbx`-centric. Internally, the current implementation uses:

- `scan` for the read-side scan pipeline
- `Atlas` for schema planning / validation on supported dialects
- `goose` as the migration runner engine inside `dbx/migrate`
- `hot` for runtime cache storage

These are implementation details. The exposed API is still `dbx`, `dbx/sqltmplx`, and `dbx/migrate`.

## Package Layout

- Core ORM API: `github.com/DaiYuANg/arcgo/dbx`
- Shared dialect contracts: `github.com/DaiYuANg/arcgo/dbx/dialect` (see [Dialect](./dialect))
- Built-in query + schema dialects:
  - `github.com/DaiYuANg/arcgo/dbx/dialect/sqlite`
  - `github.com/DaiYuANg/arcgo/dbx/dialect/postgres`
  - `github.com/DaiYuANg/arcgo/dbx/dialect/mysql`
- SQL template engine in the same ecosystem:
  - `github.com/DaiYuANg/arcgo/dbx/sqltmplx`
- Migration runner package:
  - `github.com/DaiYuANg/arcgo/dbx/migrate`

## Schema First

Schema owns database metadata. Entities only carry field mapping tags.

```go
package main

import "github.com/DaiYuANg/arcgo/dbx"

type Role struct {
    ID   int64  `dbx:"id"`
    Name string `dbx:"name"`
}

type User struct {
    ID       int64  `dbx:"id"`
    Username string `dbx:"username"`
    Email    string `dbx:"email_address"`
    Status   int    `dbx:"status"`
    RoleID   int64  `dbx:"role_id"`
}

type RoleSchema struct {
    dbx.Schema[Role]
    ID   dbx.Column[Role, int64]  `dbx:"id,pk,auto"`
    Name dbx.Column[Role, string] `dbx:"name,unique"`
}

type UserSchema struct {
    dbx.Schema[User]
    ID       dbx.Column[User, int64]   `dbx:"id,pk,auto"`
    Username dbx.Column[User, string]  `dbx:"username"`
    Email    dbx.Column[User, string]  `dbx:"email_address,unique"`
    Status   dbx.Column[User, int]     `dbx:"status,default=1"`
    RoleID   dbx.Column[User, int64]   `dbx:"role_id,ref=roles.id,ondelete=cascade"`
    Role     dbx.BelongsTo[User, Role] `rel:"table=roles,local=role_id,target=id"`
}

var Roles = dbx.MustSchema("roles", RoleSchema{})
var Users = dbx.MustSchema("users", UserSchema{})
```

## Query DSL

`dbx` renders typed queries into `BoundQuery`, then executes them through `DB` or `Tx`. For "build once, execute many" reuse, call `Build` once and use `ExecBound`, `QueryAllBound`, `QueryCursorBound`, or `QueryEachBound` in a loop:

```go
query := dbx.Select(Users.ID, Users.Username).From(Users).Where(Users.Status.Eq(1))
bound, _ := dbx.Build(session, query)
for range batches {
    items, _ := dbx.QueryAllBound(ctx, session, bound, mapper)
    // ...
}
```

```go
statusLabel := dbx.CaseWhen[string](Users.Status.Eq(1), "active").
    When(Users.Status.Eq(2), "blocked").
    Else("unknown").
    As("status_label")

activeUsers := dbx.NamedTable("active_users")
activeID := dbx.NamedColumn[int64](activeUsers, "id")
activeName := dbx.NamedColumn[string](activeUsers, "username")

query := dbx.Select(activeID, activeName, statusLabel).
    With("active_users",
        dbx.Select(Users.ID, Users.Username).
            From(Users).
            Where(Users.Status.Eq(1)),
    ).
    From(activeUsers).
    UnionAll(
        dbx.Select(Users.ID, Users.Username, statusLabel).
            From(Users).
            Where(Users.Status.Ne(1)),
    )
```

## Mapper, StructMapper, and Codecs

- **StructMapper[E]** — schema-less pure DTO mapping. Use for arbitrary SQL (SQLList, SQLGet, QueryAll) when no Schema is available. Maps result columns to struct fields by name from struct tags.
- **Mapper[E]** — schema-bound; extends StructMapper with a schema-derived field subset. Use for CRUD, relation load, repository when you have a Schema.
- **RowsScanner[E]** — read contract; both implement it. Dependency: StructMapper is independent; Mapper depends on Schema.

```go
type Preferences struct {
    Theme string   `json:"theme"`
    Flags []string `json:"flags"`
}

type Account struct {
    ID          int64       `dbx:"id"`
    Preferences Preferences `dbx:"preferences,codec=json"`
    Tags        []string    `dbx:"tags,codec=csv"`
}

csvCodec := dbx.NewCodec[[]string](
    "csv",
    func(src any) ([]string, error) { /* ... */ },
    func(values []string) (any, error) { /* ... */ },
)

mapper := dbx.MustStructMapperWithOptions[Account](
    dbx.WithMapperCodecs(csvCodec),
)
```

## Relation Loading

`dbx` now supports batch relation loading in addition to join helpers.

```go
userMapper := dbx.MustMapper[User](Users)
roleMapper := dbx.MustMapper[Role](Roles)

if err := dbx.LoadBelongsTo(
    ctx,
    core,
    users,
    Users,
    userMapper,
    Users.Role,
    Roles,
    roleMapper,
    func(index int, user *User, role mo.Option[Role]) {
        // attach resolved role here
    },
); err != nil {
    panic(err)
}
```

## Pure SQL Entry

`sqltmplx` stays responsible for template compile / render / validate. `dbx` owns execution, transaction handling, hooks, and logging.

```go
//go:embed sql/**/*.sql
var sqlFS embed.FS

registry := sqltmplx.NewRegistry(sqlFS, core.Dialect())

items, err := dbx.SQLList(
    ctx,
    core,
    registry.MustStatement("sql/user/find_active.sql"),
    struct {
        Status int `dbx:"status"`
    }{Status: 1},
    dbx.MustStructMapper[UserSummary](),
)
if err != nil {
    panic(err)
}
```

Pure SQL helpers:

- `db.SQL().Exec(...)` / `tx.SQL().Exec(...)`
- `dbx.SQLList(...)`
- `dbx.SQLGet(...)`
- `dbx.SQLFind(...)`
- `dbx.SQLScalar(...)`
- `dbx.SQLScalarOption(...)`

`SQLFind` and `SQLScalarOption` return `mo.Option[T]`.

## Schema Planning and Migration Runner

`dbx` supports schema planning, validation, SQL preview, conservative auto-migrate, and a separate migration runner.

```go
plan, err := core.PlanSchemaChanges(ctx, Roles, Users)
if err != nil {
    panic(err)
}

for _, sqlText := range plan.SQLPreview() {
    fmt.Println(sqlText)
}

runner := migrate.NewRunner(core.SQLDB(), core.Dialect(), migrate.RunnerOptions{ValidateHash: true})
_, err = runner.UpGo(ctx, migrate.NewGoMigration("1", "create users", up, nil))
if err != nil {
    panic(err)
}
```

Current behavior:

- build missing tables
- add missing columns
- add missing indexes
- add missing foreign keys and checks when the dialect supports it
- stop and report when a manual migration is required

## Options and Presets

Options use the functional Option pattern and are composable (later overrides earlier). Presets: `DefaultOptions()`, `ProductionOptions()` (debug off), `TestOptions()` (debug on for SQL logging). Individual options: `WithLogger`, `WithHooks`, `WithDebug`. See [Options](./options).

## Runtime Logging and Hooks

`DB` and `Tx` provide runtime observation hooks and `slog` debug logging. Pure SQL statements also carry their statement names into hook events and debug logs. For slow-query detection, Duration, Metadata (trace_id, request_id), see [Observability](./observability).

```go
core := dbx.NewWithOptions(
    sqlDB,
    sqlite.Dialect{},
    dbx.WithLogger(logger),
    dbx.WithDebug(true),
    dbx.WithHooks(dbx.HookFuncs{
        AfterFunc: func(_ context.Context, event *dbx.HookEvent) {
            fmt.Println(event.Operation, event.Statement)
        },
    }),
)
```

## Benchmarks

`dbx` now includes benchmark coverage for its major pipelines.

Run locally:

```bash
go test ./dbx -run '^$' -bench .
go test ./dbx/migrate -run '^$' -bench .
```

Covered areas:

- mapper metadata and scan path
- codec-aware reads and write assignments
- query build and SQL render
- relation loading
- schema planning / validation / SQL preview
- SQL executor helpers
- migration file source and runner

## Examples

- Example guide: [dbx examples](./examples)
- Runnable examples:
  - [examples/dbx/basic](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/basic)
  - [examples/dbx/codec](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/codec)
  - [examples/dbx/mutation](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/mutation)
  - [examples/dbx/query_advanced](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/query_advanced)
  - [examples/dbx/relations](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/relations)
  - [examples/dbx/migration](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/migration)
  - [examples/dbx/pure_sql](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/pure_sql)

## Verification

```bash
go test ./dbx/...
go test ./examples/dbx/...
go run ./examples/dbx/basic
go run ./examples/dbx/codec
go run ./examples/dbx/mutation
go run ./examples/dbx/query_advanced
go run ./examples/dbx/relations
go run ./examples/dbx/migration
go run ./examples/dbx/pure_sql
```
