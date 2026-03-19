---
title: 'sqltmplx'
linkTitle: 'sqltmplx'
description: 'SQL 优先的条件模板渲染器，支持可选的 parser 校验'
weight: 12
---

## sqltmplx

`sqltmplx` 是一个面向 Go 的 SQL 优先条件模板渲染器。
它把控制逻辑放在 SQL 注释里，保留可执行的 sample literal，并在运行时渲染真正的 bind 变量和参数列表。

## 包结构

- 核心渲染器：`github.com/DaiYuANg/arcgo/dbx/sqltmplx`
- 方言契约：`github.com/DaiYuANg/arcgo/dbx/dialect`
- 内置方言：
  - `github.com/DaiYuANg/arcgo/dbx/dialect/mysql`
  - `github.com/DaiYuANg/arcgo/dbx/dialect/postgres`
  - `github.com/DaiYuANg/arcgo/dbx/dialect/sqlite`
- 校验器契约与注册中心：`github.com/DaiYuANg/arcgo/dbx/sqltmplx/validate`
- 可选 parser 校验后端：
  - `github.com/DaiYuANg/arcgo/dbx/sqltmplx/validate/mysqlparser`
  - `github.com/DaiYuANg/arcgo/dbx/sqltmplx/validate/postgresparser`
  - `github.com/DaiYuANg/arcgo/dbx/sqltmplx/validate/sqliteparser`

这样拆分后，核心包保持轻量。
引入 `sqltmplx` 不会再默认把全部数据库 parser 依赖拉进同一个模块图里。

## 支持的模板能力

- `/*%if expr */ ... /*%end */`
- `/*%where */ ... /*%end */`
- `/*%set */ ... /*%end */`
- Doma 风格占位符：`/* Name */'alice'`
- 切片展开：`/* IDs */(1, 2, 3)`
- 表达式辅助函数：`empty(x)`、`blank(x)`、`present(x)`
- 结构体绑定优先按字段名，再按 `sqltmpl`、`db`、`json` 别名匹配

## 快速开始

```go
package main

import (
    "fmt"

    "github.com/DaiYuANg/arcgo/dbx/sqltmplx"
    "github.com/DaiYuANg/arcgo/dbx/dialect/postgres"
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

## 校验模型

如果你想按方言名自动选择校验器，使用 `validate.NewSQLParser(dialect)`。
它基于注册机制工作，所以程序里必须显式导入具体后端包。

例如：

```go
import (
    "github.com/DaiYuANg/arcgo/dbx/sqltmplx/validate"
    _ "github.com/DaiYuANg/arcgo/dbx/sqltmplx/validate/mysqlparser"
)
```

如果你不想走注册中心，也可以直接传具体校验器实现：

```go
sqltmplx.WithValidator(mysqlparser.New())
```

## 编译一次，多次渲染

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

## 示例

- 示例说明页：[sqltmplx examples](./examples)
- 可运行示例：
  - [examples/sqltmplx/basic](https://github.com/DaiYuANg/arcgo/tree/main/examples/sqltmplx/basic)
  - [examples/sqltmplx/postgres](https://github.com/DaiYuANg/arcgo/tree/main/examples/sqltmplx/postgres)
  - [examples/sqltmplx/sqlite_update](https://github.com/DaiYuANg/arcgo/tree/main/examples/sqltmplx/sqlite_update)
  - [examples/sqltmplx/precompile](https://github.com/DaiYuANg/arcgo/tree/main/examples/sqltmplx/precompile)

## 测试与 Benchmark

```bash
go test ./dbx/sqltmplx/...
go test ./dbx/sqltmplx -run ^$ -bench . -benchmem
go test ./dbx/sqltmplx/render -run ^$ -bench . -benchmem
go test ./dbx/sqltmplx/validate/mysqlparser -run ^$ -bench . -benchmem
```



