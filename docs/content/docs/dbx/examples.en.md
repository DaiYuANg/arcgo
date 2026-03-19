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
go run ./mutation
go run ./relations
go run ./migration
go run ./pure_sql
```

You can also run directly from the repository root:

```bash
go run ./examples/dbx/basic
go run ./examples/dbx/mutation
go run ./examples/dbx/relations
go run ./examples/dbx/migration
go run ./examples/dbx/pure_sql
```

## Example Matrix

| Example | Focus | Directory |
| --- | --- | --- |
| `basic` | schema-first modeling, mapper scan, projection, tx, debug SQL, hooks | [examples/dbx/basic](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/basic) |
| `mutation` | aggregate queries, subqueries, batch insert, insert-select, upsert, returning | [examples/dbx/mutation](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/mutation) |
| `relations` | alias + relation metadata + `JoinRelation` for `BelongsTo` and `ManyToMany` | [examples/dbx/relations](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/relations) |
| `migration` | `PlanSchemaChanges`, `AutoMigrate`, `ValidateSchemas`, `ForeignKeys()` | [examples/dbx/migration](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/migration) |
| `pure_sql` | `sqltmplx` registry, `dbx.SQLList/SQLScalar`, statement-name logging, `tx.SQL().Exec(...)` | [examples/dbx/pure_sql](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/pure_sql) |

## Example: Query with Mapper

```go
mapper := dbx.MustMapper[shared.User](catalog.Users)
items, err := dbx.QueryAll(
    ctx,
    core,
    dbx.Select(catalog.Users.AllColumns()...).
        From(catalog.Users).
        Where(catalog.Users.Status.Eq(1)).
        OrderBy(catalog.Users.ID.Asc()),
    mapper,
)
if err != nil {
    panic(err)
}
```

## Example: Query DSL Mutations

```go
archiveMapper := dbx.MustMapper[userArchive](archive)

items, err := dbx.QueryAll(
    ctx,
    core,
    dbx.InsertInto(archive).
        Columns(archive.Username, archive.Status).
        FromSelect(
            dbx.Select(catalog.Users.Username, catalog.Users.Status).
                From(catalog.Users).
                Where(catalog.Users.Status.Eq(1)),
        ).
        Returning(archive.ID, archive.Username, archive.Status),
    archiveMapper,
)
if err != nil {
    panic(err)
}
```

## Example: Upsert with Excluded Values

```go
items, err := dbx.QueryAll(
    ctx,
    core,
    dbx.InsertInto(archive).
        Values(
            archive.Username.Set("alice"),
            archive.Status.Set(9),
        ).
        OnConflict(archive.Username).
        DoUpdateSet(archive.Status.SetExcluded()).
        Returning(archive.ID, archive.Username, archive.Status),
    archiveMapper,
)
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
