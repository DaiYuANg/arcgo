---
title: 'dbx'
linkTitle: 'dbx'
description: '基于 database/sql 的类型安全、泛型优先 ORM 核心'
weight: 7
---

## dbx

`dbx` 是一个构建在 `database/sql` 之上的类型安全、泛型优先 ORM 核心。
它把 schema 作为唯一数据库元数据来源，把 entity 保持为数据承载结构，同时提供四条并行能力：

- 结构化查询 DSL
- 基于 `sqltmplx` 和 `BoundQuery` 的纯 SQL 执行
- schema 校验与保守式自动迁移
- 运行时日志、hook 与事务封装

## 设计方向

`dbx` 并不是一个带状态的重 ORM。
当前设计重点是：

- `Schema[E]` 作为唯一数据库元数据源
- 带显式泛型信息的 `Column[E, T]` 和关系引用
- `Mapper[E]` 负责 schema-aware 实体映射
- `StructMapper[E]` 负责纯 SQL 和 DTO 扫描
- `DB` / `Tx` 封装在 `*sql.DB` / `*sql.Tx` 之上
- `DB.SQL()` / `Tx.SQL()` 作为纯 SQL 执行入口
- 基于 `slog` 的 debug 日志和 hook
- 保守的 schema diff 与 migration planning

## 包结构

- ORM 核心 API：`github.com/DaiYuANg/arcgo/dbx`
- 共享方言契约：`github.com/DaiYuANg/arcgo/dbx/dialect`
- 内置 query + schema 方言：
  - `github.com/DaiYuANg/arcgo/dbx/dialect/sqlite`
  - `github.com/DaiYuANg/arcgo/dbx/dialect/postgres`
  - `github.com/DaiYuANg/arcgo/dbx/dialect/mysql`
- 同生态 SQL 模板引擎：
  - `github.com/DaiYuANg/arcgo/dbx/sqltmplx`
- 可选 migration runner：
  - `github.com/DaiYuANg/arcgo/dbx/migrate`

## 核心建模

- `Schema[E]`：表级元数据根节点
- `Column[E, T]`：强类型列引用与 predicate/assignment 入口
- `BelongsTo/HasOne/HasMany/ManyToMany`：强类型关系引用
- `Mapper[E]`：schema-aware 映射与 scan plan cache
- `StructMapper[E]`：面向纯 SQL 和 DTO 的结构扫描器
- `BoundQuery`：渲染后的 SQL、参数以及可选 statement name
- `DB` / `Tx`：带日志与 hook 的运行时执行封装
- `SQLExecutor`：由 `DB.SQL()` / `Tx.SQL()` 返回的纯 SQL facade

## Schema First

数据库元数据全部由 schema 维护。
entity 只负责字段映射 tag。

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
    ID       dbx.Column[User, int64]    `dbx:"id,pk,auto"`
    Username dbx.Column[User, string]   `dbx:"username"`
    Email    dbx.Column[User, string]   `dbx:"email_address,unique"`
    Status   dbx.Column[User, int]      `dbx:"status,default=1"`
    RoleID   dbx.Column[User, int64]    `dbx:"role_id,ref=roles.id,ondelete=cascade"`
    Role     dbx.BelongsTo[User, Role]  `rel:"table=roles,local=role_id,target=id"`
    Roles    dbx.ManyToMany[User, Role] `rel:"table=roles,target=id,join=user_roles,join_local=user_id,join_target=role_id"`
}

var Roles = dbx.MustSchema("roles", RoleSchema{})
var Users = dbx.MustSchema("users", UserSchema{})
```

## Query DSL

`dbx` 会先把强类型查询渲染成 `BoundQuery`，再通过 `DB` 或 `Tx` 执行。

```go
query := dbx.Select(Users.ID, Users.Username, Users.Email).
    From(Users).
    Where(Users.Status.Eq(1)).
    OrderBy(Users.ID.Asc())

bound, err := dbx.Build(core, query)
if err != nil {
    panic(err)
}

fmt.Println(bound.SQL)
fmt.Println(bound.Args)
```

## Mapper 与 Projection

用 `Mapper[E]` 做 schema-aware 的结果扫描和字段级投影。

```go
mapper := dbx.MustMapper[User](Users)

items, err := dbx.QueryAll(
    ctx,
    core,
    dbx.Select(Users.AllColumns()...).From(Users).Where(Users.Status.Eq(1)),
    mapper,
)
if err != nil {
    panic(err)
}

summaryMapper := dbx.MustMapper[UserSummary](Users)
summaries, err := dbx.QueryAll(
    ctx,
    core,
    dbx.MustSelectMapped(Users, summaryMapper),
    summaryMapper,
)
```

如果只是纯 SQL 或 DTO 查询，优先使用 `StructMapper[E]`，不要为了扫描去依赖 schema-aware `Mapper[E]`。

## 纯 SQL 入口

`sqltmplx` 继续负责模板编译、渲染和校验。
`dbx` 负责执行、事务、hook 和日志。

```go
//go:embed sql/**/*.sql
var sqlFS embed.FS

registry := sqltmplx.NewRegistry(sqlFS, core.Dialect())

items, err := dbx.SQLList(
    ctx,
    core.SQL(),
    registry.MustStatement("sql/user/find_active.sql"),
    struct {
        Status int `dbx:"status"`
    }{Status: 1},
    dbx.MustStructMapper[UserSummary](),
)
if err != nil {
    panic(err)
}

count, err := dbx.SQLScalar[int64](
    ctx,
    core.SQL(),
    registry.MustStatement("sql/user/count_by_status.sql"),
    struct {
        Status int `dbx:"status"`
    }{Status: 1},
)
if err != nil {
    panic(err)
}
```

当前纯 SQL 辅助入口包括：

- `db.SQL().Exec(...)` / `tx.SQL().Exec(...)`
- `dbx.SQLList(...)`
- `dbx.SQLGet(...)`
- `dbx.SQLFind(...)`
- `dbx.SQLScalar(...)`
- `dbx.SQLScalarOption(...)`

其中 `SQLFind` 和 `SQLScalarOption` 返回 `mo.Option[T]`。

## 纯 SQL 事务示例

```go
tx, err := core.BeginTx(ctx, nil)
if err != nil {
    panic(err)
}

if _, err := tx.SQL().Exec(
    ctx,
    registry.MustStatement("sql/user/update_status.sql"),
    struct {
        Status   int    `dbx:"status"`
        Username string `dbx:"username"`
    }{
        Status:   2,
        Username: "bob",
    },
); err != nil {
    _ = tx.Rollback()
    panic(err)
}

if err := tx.Commit(); err != nil {
    panic(err)
}
```

## 关系与 Join Helper

alias 和 relation metadata 可以直接桥接成 join。

```go
users := dbx.Alias(Users, "u")
roles := dbx.Alias(Roles, "r")

query := dbx.Select(users.ID, users.Username, roles.Name).From(users)
if _, err := query.JoinRelation(users, users.Role, roles); err != nil {
    panic(err)
}
```

many-to-many 也是同样的入口：

```go
query := dbx.Select(users.Username, roles.Name).From(users)
if _, err := query.JoinRelation(users, users.Roles, roles); err != nil {
    panic(err)
}
```

## 运行时日志与 Hook

`DB` 和 `Tx` 内建了运行时 hook 和 `slog` debug SQL 日志。
纯 SQL statement 的名称也会进入 hook event 和 debug 日志。

```go
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

core := dbx.NewWithOptions(
    sqlDB,
    sqlite.Dialect{},
    dbx.WithLogger(logger),
    dbx.WithDebug(true),
    dbx.WithHooks(dbx.HookFuncs{
        AfterFunc: func(_ context.Context, event *dbx.HookEvent) {
            fmt.Println("operation:", event.Operation)
            fmt.Println("statement:", event.Statement)
        },
    }),
)
```

## Schema 校验与 Auto-Migrate

`dbx` 当前已经支持 schema inspect、diff、migration planning 和保守式 auto-migrate。

行为边界：

- 构建缺失表
- 补充缺失列
- 补充缺失索引
- 在方言支持时补充外键和 check
- 一旦发现需要手工处理的变更就停止并报告

```go
plan, err := core.PlanSchemaChanges(ctx, Roles, Users)
if err != nil {
    panic(err)
}

for _, action := range plan.Actions {
    fmt.Println(action.Kind, action.Executable, action.Summary)
}

report, err := core.AutoMigrate(ctx, Roles, Users)
if err != nil {
    panic(err)
}
fmt.Println(report.Valid())
```

## 当前范围

`dbx` 目前已经比较完整的部分：

- schema-first 建模
- typed query build 与 execution
- typed mapping 与 projection
- 通过 `sqltmplx` statement 做纯 SQL 执行
- relation-aware join helper
- runtime logging 与 hook
- schema diff / plan / validate / auto-migrate

仍然处于迭代中的部分：

- 超出保守 auto-migrate 的更强 DDL planning
- 更高层的 repository / active-record 易用性封装

## 示例

- 示例总览页：[dbx examples](./examples)
- 可运行示例：
  - [examples/dbx/basic](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/basic)
  - [examples/dbx/relations](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/relations)
  - [examples/dbx/migration](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/migration)
  - [examples/dbx/pure_sql](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/pure_sql)

## 测试

```bash
go test ./dbx/...
go run ./examples/dbx/basic
go run ./examples/dbx/relations
go run ./examples/dbx/migration
go run ./examples/dbx/pure_sql
```
