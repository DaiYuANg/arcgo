---
title: 'sqltmplx 示例'
linkTitle: 'examples'
description: 'sqltmplx 可运行示例'
weight: 10
---

## sqltmplx 示例

这一页汇总了 `examples/sqltmplx` 下的可运行程序，并说明它们分别覆盖哪些 API 场景。

## 本地运行

在 `examples/sqltmplx` 模块目录下执行：

```bash
cd examples/sqltmplx
go run ./basic
go run ./postgres
go run ./sqlite_update
go run ./precompile
```

## 示例矩阵

| 示例 | 重点 | 目录 |
| --- | --- | --- |
| `basic` | MySQL 方言、注册式 validator 选择、map 参数 | [examples/sqltmplx/basic](https://github.com/DaiYuANg/arcgo/tree/main/examples/sqltmplx/basic) |
| `postgres` | PostgreSQL bind 变量、结构体 tag 绑定、parser 校验 | [examples/sqltmplx/postgres](https://github.com/DaiYuANg/arcgo/tree/main/examples/sqltmplx/postgres) |
| `sqlite_update` | 动态 `set` 语句清理 | [examples/sqltmplx/sqlite_update](https://github.com/DaiYuANg/arcgo/tree/main/examples/sqltmplx/sqlite_update) |
| `precompile` | `Compile()` 一次，多次渲染 | [examples/sqltmplx/precompile](https://github.com/DaiYuANg/arcgo/tree/main/examples/sqltmplx/precompile) |

## 示例：动态 WHERE

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

## 示例：动态 SET

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

## 示例：模板复用

```go
engine := sqltmplx.New(mysql.Dialect{})
tpl, err := engine.Compile(queryText)
if err != nil {
    panic(err)
}

bound1, _ := tpl.Render(map[string]any{"Tenant": "acme", "Status": "PAID"})
bound2, _ := tpl.Render(map[string]any{"Tenant": "acme", "Status": "SHIPPED"})
```


