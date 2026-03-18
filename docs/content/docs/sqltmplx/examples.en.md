---
title: 'sqltmplx examples'
linkTitle: 'examples'
description: 'Runnable examples for sqltmplx'
weight: 10
---

## sqltmplx Examples

This page collects the runnable `examples/sqltmplx` programs and maps them to the API surface they demonstrate.

## Run Locally

Run from the `examples/sqltmplx` module:

```bash
cd examples/sqltmplx
go run ./basic
go run ./postgres
go run ./sqlite_update
go run ./precompile
```

## Example Matrix

| Example | Focus | Directory |
| --- | --- | --- |
| `basic` | MySQL dialect, registry-based validator selection, map parameters | [examples/sqltmplx/basic](https://github.com/DaiYuANg/arcgo/tree/main/examples/sqltmplx/basic) |
| `postgres` | PostgreSQL bind variables, struct tag binding, parser-backed validation | [examples/sqltmplx/postgres](https://github.com/DaiYuANg/arcgo/tree/main/examples/sqltmplx/postgres) |
| `sqlite_update` | `set` block cleanup for dynamic update statements | [examples/sqltmplx/sqlite_update](https://github.com/DaiYuANg/arcgo/tree/main/examples/sqltmplx/sqlite_update) |
| `precompile` | `Compile()` once and render many times | [examples/sqltmplx/precompile](https://github.com/DaiYuANg/arcgo/tree/main/examples/sqltmplx/precompile) |

## Example: Dynamic WHERE

```go
engine := sqltmplx.New(
    postgres.Dialect{},
    sqltmplx.WithValidator(validate.NewSQLParser(postgres.Dialect{})),
)

bound, err := engine.Render(tpl, Query{
    Tenant: "acme",
    Name:   "alice",
    IDs:    []int{1, 2, 3},
})
if err != nil {
    panic(err)
}
```

## Example: Dynamic SET

```go
engine := sqltmplx.New(sqlite.Dialect{})

bound, err := engine.Render(tpl, UpdateCommand{
    ID:     42,
    Name:   "alice",
    Status: "active",
})
if err != nil {
    panic(err)
}
```

## Example: Template Reuse

```go
engine := sqltmplx.New(mysql.Dialect{})
tpl, err := engine.Compile(queryText)
if err != nil {
    panic(err)
}

bound1, _ := tpl.Render(map[string]any{"Tenant": "acme", "Status": "PAID"})
bound2, _ := tpl.Render(map[string]any{"Tenant": "acme", "Status": "SHIPPED"})
```


