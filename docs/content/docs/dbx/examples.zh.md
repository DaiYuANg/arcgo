---
title: 'dbx 示例'
linkTitle: 'examples'
description: 'dbx 的可运行示例'
weight: 10
---

## dbx 示例

这一页汇总了 `examples/dbx` 下的可运行程序，并说明它们分别覆盖哪些 API 场景。

## 本地运行

在 `examples/dbx` 模块目录下执行：

```bash
cd examples/dbx
go run ./basic
go run ./mutation
go run ./relations
go run ./migration
go run ./pure_sql
```

也可以直接从仓库根目录执行：

```bash
go run ./examples/dbx/basic
go run ./examples/dbx/mutation
go run ./examples/dbx/relations
go run ./examples/dbx/migration
go run ./examples/dbx/pure_sql
```

## 示例矩阵

| 示例 | 重点 | 目录 |
| --- | --- | --- |
| `basic` | schema-first 建模、mapper 扫描、projection、事务、debug SQL、hooks | [examples/dbx/basic](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/basic) |
| `mutation` | 聚合查询、子查询、批量插入、insert-select、upsert、returning | [examples/dbx/mutation](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/mutation) |
| `relations` | alias + relation metadata + `JoinRelation`，覆盖 `BelongsTo` 和 `ManyToMany` | [examples/dbx/relations](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/relations) |
| `migration` | `PlanSchemaChanges`、`AutoMigrate`、`ValidateSchemas`、`ForeignKeys()` | [examples/dbx/migration](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/migration) |
| `pure_sql` | `sqltmplx` registry、`dbx.SQLList/SQLScalar`、statement 名称日志、`tx.SQL().Exec(...)` | [examples/dbx/pure_sql](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/pure_sql) |

## 示例：结合 Mapper 查询

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

## 示例：Query DSL 写入

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

## 示例：带 Excluded 值的 Upsert

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

## 示例：结合 sqltmplx 做纯 SQL

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
