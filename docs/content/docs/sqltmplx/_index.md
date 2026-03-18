---
title: 'sqltmplx'
linkTitle: 'sqltmplx'
description: 'SQL-first conditional template renderer with optional parser-backed validation'
weight: 12
---

## sqltmplx

`sqltmplx` is a SQL-first conditional template renderer for Go.
It keeps control flow inside SQL comments, preserves sample literals for tooling, and renders bind variables plus arguments at runtime.

## Package Layout

- Core renderer: `github.com/DaiYuANg/arcgo/dbx/sqltmplx`
- Dialect contracts: `github.com/DaiYuANg/arcgo/dbx/sqltmplx/dialect`
- Built-in dialects:
  - `github.com/DaiYuANg/arcgo/dbx/sqltmplx/dialect/mysql`
  - `github.com/DaiYuANg/arcgo/dbx/sqltmplx/dialect/postgres`
  - `github.com/DaiYuANg/arcgo/dbx/sqltmplx/dialect/sqlite`
- Validator contract and registry: `github.com/DaiYuANg/arcgo/dbx/sqltmplx/validate`
- Optional parser-backed validators:
  - `github.com/DaiYuANg/arcgo/dbx/sqltmplx/validate/mysqlparser`
  - `github.com/DaiYuANg/arcgo/dbx/sqltmplx/validate/postgresparser`
  - `github.com/DaiYuANg/arcgo/dbx/sqltmplx/validate/sqliteparser`

This split keeps the core package lightweight.
Importing `sqltmplx` no longer forces all database parser dependencies into the same module graph.

## Supported Template Features

- `/*%if expr */ ... /*%end */`
- `/*%where */ ... /*%end */`
- `/*%set */ ... /*%end */`
- Doma-style placeholders: `/* Name */'alice'`
- Slice expansion: `/* IDs */(1, 2, 3)`
- Expression helpers: `empty(x)`, `blank(x)`, `present(x)`
- Struct binding by field name first, then `sqltmpl`, `db`, `json` aliases

## Quick Start

```go
package main

import (
    "fmt"

    "github.com/DaiYuANg/arcgo/dbx/sqltmplx"
    "github.com/DaiYuANg/arcgo/dbx/sqltmplx/dialect/postgres"
    "github.com/DaiYuANg/arcgo/dbx/sqltmplx/validate"
    _ "github.com/DaiYuANg/arcgo/dbx/sqltmplx/validate/postgresparser"
)

type Query struct {
    Tenant string `db:"tenant"`
    Name   string `json:"name"`
    IDs    []int  `json:"ids"`
}

func main() {
    engine := sqltmplx.New(
        postgres.Dialect{},
        sqltmplx.WithValidator(validate.NewSQLParser(postgres.Dialect{})),
    )

    tpl := `
SELECT id, tenant, name
FROM users
/*%where */
/*%if present(Tenant) */
  AND tenant = /* Tenant */'acme'
/*%end */
/*%if present(Name) */
  AND name = /* Name */'alice'
/*%end */
/*%if !empty(IDs) */
  AND id IN (/* IDs */(1, 2, 3))
/*%end */
/*%end */
ORDER BY id DESC
`

    bound, err := engine.Render(tpl, Query{
        Tenant: "acme",
        Name:   "alice",
        IDs:    []int{1, 2, 3},
    })
    if err != nil {
        panic(err)
    }

    fmt.Println(bound.Query)
    fmt.Println(bound.Args)
}
```

## Validation Model

Use `validate.NewSQLParser(dialect)` when you want validator selection by dialect name.
This works through registration, so the specific backend package must be imported somewhere in your program.

Example:

```go
import (
    "github.com/DaiYuANg/arcgo/dbx/sqltmplx/validate"
    _ "github.com/DaiYuANg/arcgo/dbx/sqltmplx/validate/mysqlparser"
)
```

You can also bypass the registry and pass a validator implementation directly:

```go
sqltmplx.WithValidator(mysqlparser.New())
```

## Compile Once, Render Many

```go
engine := sqltmplx.New(mysql.Dialect{})
tpl, err := engine.Compile(queryText)
if err != nil {
    panic(err)
}

bound1, _ := tpl.Render(map[string]any{"Tenant": "acme", "Status": "PAID"})
bound2, _ := tpl.Render(map[string]any{"Tenant": "acme", "Status": "SHIPPED"})

fmt.Println(bound1.Query)
fmt.Println(bound2.Query)
```

## Examples

- Example guide: [sqltmplx examples](./examples)
- Runnable examples:
  - [examples/sqltmplx/basic](https://github.com/DaiYuANg/arcgo/tree/main/examples/sqltmplx/basic)
  - [examples/sqltmplx/postgres](https://github.com/DaiYuANg/arcgo/tree/main/examples/sqltmplx/postgres)
  - [examples/sqltmplx/sqlite_update](https://github.com/DaiYuANg/arcgo/tree/main/examples/sqltmplx/sqlite_update)
  - [examples/sqltmplx/precompile](https://github.com/DaiYuANg/arcgo/tree/main/examples/sqltmplx/precompile)

## Testing and Benchmarks

```bash
go test ./dbx/sqltmplx/...
go test ./dbx/sqltmplx -run ^$ -bench . -benchmem
go test ./dbx/sqltmplx/render -run ^$ -bench . -benchmem
go test ./dbx/sqltmplx/validate/mysqlparser -run ^$ -bench . -benchmem
```



