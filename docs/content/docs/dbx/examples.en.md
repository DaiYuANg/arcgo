---
title: 'dbx examples'
linkTitle: 'examples'
description: 'Runnable examples for dbx'
weight: 10
---

## dbx Examples

This page collects the runnable `examples/dbx` programs and maps them to the API surface they demonstrate.

## Run Locally

Run from the `examples/dbx` module:

```bash
cd examples/dbx
go run ./basic
go run ./codec
go run ./mutation
go run ./query_advanced
go run ./relations
go run ./migration
go run ./pure_sql
```

You can also run directly from the repository root:

```bash
go run ./examples/dbx/basic
go run ./examples/dbx/codec
go run ./examples/dbx/mutation
go run ./examples/dbx/query_advanced
go run ./examples/dbx/relations
go run ./examples/dbx/migration
go run ./examples/dbx/pure_sql
```

## Example Matrix

| Example | Focus | Directory |
| --- | --- | --- |
| `basic` | schema-first modeling, mapper scan, projection, tx, debug SQL, hooks | [examples/dbx/basic](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/basic) |
| `codec` | built-in codecs, scoped custom codecs, struct mapper reads, mapper write assignments | [examples/dbx/codec](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/codec) |
| `mutation` | aggregate queries, subqueries, batch insert, insert-select, upsert, returning | [examples/dbx/mutation](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/mutation) |
| `query_advanced` | `WITH`, `UNION ALL`, `CASE WHEN`, named tables, result columns | [examples/dbx/query_advanced](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/query_advanced) |
| `relations` | alias + relation metadata + `JoinRelation`, plus `LoadBelongsTo` and `LoadManyToMany` | [examples/dbx/relations](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/relations) |
| `migration` | `PlanSchemaChanges`, `SQLPreview`, `AutoMigrate`, `ValidateSchemas`, `core.Migrator(...).UpGo/UpSQL` | [examples/dbx/migration](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/migration) |
| `pure_sql` | `sqltmplx` registry, `dbx.SQLList/SQLGet/SQLFind/SQLScalar`, statement-name logging, `tx.SQL().Exec(...)` | [examples/dbx/pure_sql](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/pure_sql) |

## Example: Codec and StructMapper

```go
mapper := dbx.MustStructMapperWithOptions[shared.Account](
    dbx.WithMapperCodecs(csvCodec),
)

items, err := dbx.QueryAll(
    ctx,
    core,
    dbx.Select(catalog.Accounts.AllColumns()...).From(catalog.Accounts),
    mapper,
)
if err != nil {
    panic(err)
}
```

## Example: Advanced Query DSL

```go
statusLabel := dbx.CaseWhen[string](catalog.Users.Status.Eq(1), "active").
    Else("inactive").
    As("status_label")

activeUsers := dbx.NamedTable("active_users")
activeID := dbx.NamedColumn[int64](activeUsers, "id")
activeName := dbx.NamedColumn[string](activeUsers, "username")

query := dbx.Select(activeID, activeName, statusLabel).
    With("active_users",
        dbx.Select(catalog.Users.ID, catalog.Users.Username).
            From(catalog.Users).
            Where(catalog.Users.Status.Eq(1)),
    ).
    From(activeUsers)
```

## Example: Relation Loading

```go
if err := dbx.LoadBelongsTo(
    ctx,
    core,
    users,
    catalog.Users,
    userMapper,
    catalog.Users.Role,
    catalog.Roles,
    roleMapper,
    func(index int, user *shared.User, role mo.Option[shared.Role]) {
        // attach role
    },
); err != nil {
    panic(err)
}
```

## Example: Schema Plan Preview and Runner

```go
plan, err := core.PlanSchemaChanges(ctx, catalog.Roles, catalog.Users, catalog.UserRoles)
if err != nil {
    panic(err)
}

for _, sqlText := range plan.SQLPreview() {
    fmt.Println(sqlText)
}

runner := core.Migrator(migrate.RunnerOptions{ValidateHash: true})
_, err = runner.UpSQL(ctx, source)
if err != nil {
    panic(err)
}
```

## Example: Pure SQL With sqltmplx

```go
registry := sqltmplx.NewRegistry(sqlFS, core.Dialect())

items, err := dbx.SQLList(
    ctx,
    core.SQL(),
    registry.MustStatement("sql/user/find_active.sql"),
    struct {
        Status int `dbx:"status"`
    }{Status: 1},
    dbx.MustStructMapper[shared.UserSummary](),
)
if err != nil {
    panic(err)
}
```
