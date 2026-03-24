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
go run ./codec
go run ./mutation
go run ./query_advanced
go run ./relations
go run ./migration
go run ./pure_sql
```

也可以直接从仓库根目录执行：

```bash
go run ./examples/dbx/basic
go run ./examples/dbx/codec
go run ./examples/dbx/mutation
go run ./examples/dbx/query_advanced
go run ./examples/dbx/relations
go run ./examples/dbx/migration
go run ./examples/dbx/pure_sql
```

## 示例矩阵

| 示例 | 重点 | 目录 |
| --- | --- | --- |
| `basic` | schema-first 建模、mapper 扫描、projection、事务、debug SQL、hooks | [examples/dbx/basic](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/basic) |
| `codec` | 内建 codec、scoped custom codec、struct mapper 读取、mapper 写入 assignment | [examples/dbx/codec](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/codec) |
| `mutation` | 聚合查询、子查询、批量插入、insert-select、upsert、returning | [examples/dbx/mutation](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/mutation) |
| `query_advanced` | `WITH`、`UNION ALL`、`CASE WHEN`、named table、result column | [examples/dbx/query_advanced](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/query_advanced) |
| `relations` | alias + relation metadata + `JoinRelation`，以及 `LoadBelongsTo`、`LoadManyToMany` | [examples/dbx/relations](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/relations) |
| `migration` | `PlanSchemaChanges`、`SQLPreview`、`AutoMigrate`、`ValidateSchemas`、`migrate.NewRunner(core.SQLDB(), core.Dialect(), ...).UpGo/UpSQL` | [examples/dbx/migration](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/migration) |
| `pure_sql` | `sqltmplx` registry、`dbx.SQLList/SQLGet/SQLFind/SQLScalar`、statement 名称日志、`tx.SQL().Exec(...)` | [examples/dbx/pure_sql](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/pure_sql) |

## 示例：Codec 与 StructMapper

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

## 示例：高级 Query DSL

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

## 示例：关系加载

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
        // 在这里挂回角色
    },
); err != nil {
    panic(err)
}
```

## 示例：Schema Plan 预览与 Runner

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

## 示例：结合 sqltmplx 做纯 SQL

```go
registry := sqltmplx.NewRegistry(sqlFS, core.Dialect())

items, err := dbx.SQLList(
    ctx,
    core,
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
