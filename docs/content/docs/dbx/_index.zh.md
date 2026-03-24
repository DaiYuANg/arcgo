---
title: 'dbx'
linkTitle: 'dbx'
description: '基于 database/sql 的类型安全、泛型优先 ORM 核心'
weight: 7
---

## dbx

`dbx` 是一个构建在 `database/sql` 之上的 schema-first、generic-first ORM 核心。
它把数据库元数据集中在 `Schema[E]`，把 entity 保持为数据承载结构，目前已经覆盖这些核心链路：

- 强类型 schema 与 relation 建模
- 强类型 Query DSL 与 SQL 渲染
- 带 codec 的 mapper / struct-mapper 读取链路
- 通过 `sqltmplx` statement 做纯 SQL 执行
- `BelongsTo` / `HasOne` / `HasMany` / `ManyToMany` 的关系加载
- schema planning、validation、保守式 auto-migrate 与 migration runner
- 运行时日志、hook、事务以及 benchmark 覆盖

## 当前状态

当前 `dbx` 已包含：

- `Schema[E]` 作为唯一数据库元数据源
- `Column[E, T]` 与强类型 relation ref
- 支持聚合、子查询、CTE、`UNION ALL`、`CASE WHEN`、批量插入、`INSERT ... SELECT`、upsert、`RETURNING` 的 Query DSL
- `StructMapper[E]`（无 schema 纯 DTO 映射）与 `Mapper[E]`（依赖 Schema，用于 CRUD/关系加载）；`RowsScanner` 作为读取契约
- 字段 codec 系统，内建 `json`、`text`、`unix_time`、`unix_milli_time`、`unix_nano_time`、`rfc3339_time`、`rfc3339nano_time`
- 通过 `dbx.WithMapperCodecs(...)` 注入 scoped custom codec
- `DB.SQL()` / `Tx.SQL()` 作为纯 SQL 执行入口
- 关系加载 API 与 relation-aware join helper
- `PlanSchemaChanges`、`ValidateSchemas`、`AutoMigrate`、`MigrationPlan.SQLPreview()`
- `dbx/migrate` 下的 Go migration 与 Flyway 风格 SQL migration runner
- 针对 mapper、query、sql executor、relation、schema、migrate 的 benchmark 覆盖

## 内部实现引擎

公开 API 仍然是 `dbx` 风格，第三方库只作为内部实现依赖：

- `scan`：读侧扫描链路
- `Atlas`：支持方言上的 schema planning / validation
- `goose`：`dbx/migrate` 内部 migration runner 引擎
- `hot`：运行时缓存存储

这些都不是对外 API，外部入口仍然是 `dbx`、`dbx/sqltmplx`、`dbx/migrate`。

## 包结构

- ORM 核心 API：`github.com/DaiYuANg/arcgo/dbx`
- 共享方言契约：`github.com/DaiYuANg/arcgo/dbx/dialect`（详见 [Dialect](./dialect)）
- 内置 query + schema 方言：
  - `github.com/DaiYuANg/arcgo/dbx/dialect/sqlite`
  - `github.com/DaiYuANg/arcgo/dbx/dialect/postgres`
  - `github.com/DaiYuANg/arcgo/dbx/dialect/mysql`
- 同生态 SQL 模板引擎：
  - `github.com/DaiYuANg/arcgo/dbx/sqltmplx`
- migration runner 包：
  - `github.com/DaiYuANg/arcgo/dbx/migrate`

## 文档导航

- 从这里开始：[Getting Started](./getting-started)
- Schema 声明与建模：[Schema Design](./schema-design)
- 端到端 CRUD：[CRUD Tutorial](./tutorial-crud)
- 关系加载实战：[Relations Tutorial](./tutorial-relations)
- Schema 规划与迁移：[Migration Tutorial](./tutorial-migration)
- 模板化纯 SQL：[Pure SQL Tutorial](./tutorial-pure-sql)
- ID 策略与运行时生成器配置：[ID Generation](./id-generation)
- 索引声明与迁移行为：[Indexes](./indexes)
- 运行时选项：[Options](./options)
- 日志与 Hook：[Observability](./observability)
- 生产落地清单：[Production Checklist](./production-checklist)
- API 速查：[API Quick Reference](./api-reference)
- 泛型仓储抽象：[Repository Mode](./repository)
- 方言抽象：[Dialect](./dialect)
- dbx + 纯 SQL 模板：[sqltmplx Integration](./sqltmplx)
- 可运行示例：[Examples](./examples)
- 基准说明：[Benchmarks](./benchmarks)

## Schema First

数据库元数据全部由 schema 维护，entity 只负责字段映射 tag。

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
    ID   dbx.Column[Role, int64]  `dbx:"id,pk"`
    Name dbx.Column[Role, string] `dbx:"name,unique"`
}

type UserSchema struct {
    dbx.Schema[User]
    ID       dbx.Column[User, int64]   `dbx:"id,pk"`
    Username dbx.Column[User, string]  `dbx:"username"`
    Email    dbx.Column[User, string]  `dbx:"email_address,unique"`
    Status   dbx.Column[User, int]     `dbx:"status,default=1"`
    RoleID   dbx.Column[User, int64]   `dbx:"role_id,ref=roles.id,ondelete=cascade"`
    Role     dbx.BelongsTo[User, Role] `rel:"table=roles,local=role_id,target=id"`
}

var Roles = dbx.MustSchema("roles", RoleSchema{})
var Users = dbx.MustSchema("users", UserSchema{})
```

如果你要显式指定 ID 生成策略，推荐使用 marker type 的强类型 API：

```go
type Event struct {
    ID   int64  `dbx:"id"`
    Name string `dbx:"name"`
}

type EventSchema struct {
    dbx.Schema[Event]
    ID   dbx.IDColumn[Event, int64, dbx.IDSnowflake] `dbx:"id,pk"`
    Name dbx.Column[Event, string]                   `dbx:"name"`
}

var Events = dbx.MustSchema("events", EventSchema{})
```

## Query DSL

`dbx` 会先把强类型查询渲染成 `BoundQuery`，再通过 `DB` 或 `Tx` 执行。若要「构建一次，多次执行」，可先调用 `Build` 一次，再在循环中使用 `ExecBound`、`QueryAllBound`、`QueryCursorBound` 或 `QueryEachBound`：

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

## Mapper、StructMapper 与 Codec

- **StructMapper[E]** — 无 schema 的纯 DTO 映射。用于任意 SQL（SQLList、SQLGet、QueryAll）且无 Schema 时，按 struct tag 列名映射结果列到 struct 字段。
- **Mapper[E]** — 依赖 Schema；在 StructMapper 基础上按 schema 列过滤字段。用于 CRUD、关系加载、repository 等需要 Schema 的场景。
- **RowsScanner[E]** — 读取契约；两者均实现。依赖方向：StructMapper 独立；Mapper 依赖 Schema。

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

## 关系加载

除了 join helper，`dbx` 现在已经支持批量关系加载。

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
        // 在这里把角色挂回用户
    },
); err != nil {
    panic(err)
}
```

## 纯 SQL 入口

`sqltmplx` 继续负责模板 compile / render / validate，`dbx` 负责执行、事务、hook 和日志。

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

当前纯 SQL 辅助入口：

- `db.SQL().Exec(...)` / `tx.SQL().Exec(...)`
- `dbx.SQLList(...)`
- `dbx.SQLGet(...)`
- `dbx.SQLFind(...)`
- `dbx.SQLScalar(...)`
- `dbx.SQLScalarOption(...)`

其中 `SQLFind` 和 `SQLScalarOption` 返回 `mo.Option[T]`。

## Schema Planning 与 Migration Runner

`dbx` 当前已经支持 schema planning、validation、SQL preview、保守式 auto-migrate，以及独立的 migration runner。

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

当前 auto-migrate 的行为边界：

- 构建缺失表
- 补充缺失列
- 补充缺失索引
- 在方言支持时补充外键和 check
- 一旦发现需要手工处理的变更就停止并报告

## Options 与预设

Options 使用函数式 Option 模式，可组合（后者覆盖前者）。预设：`DefaultOptions()`（显式默认）、`ProductionOptions()`（debug 关闭）、`TestOptions()`（debug 开启，用于 SQL 日志）。单个选项：`WithLogger`、`WithHooks`、`WithDebug`。详见 [Options](./options)。  
主键 ID 的强类型策略配置，详见 [ID Generation](./id-generation)。

## 运行时日志与 Hook

`DB` 和 `Tx` 内建了运行时 hook 与 `slog` debug SQL 日志。纯 SQL statement 名称也会进入 hook event 和 debug 日志。慢查询检测、Duration、Metadata（trace_id、request_id）等详见 [Observability 可观测性](./observability)。

```go
core := dbx.NewWithOptions(
    sqlDB,
    sqlite.New(),
    dbx.WithLogger(logger),
    dbx.WithDebug(true),
    dbx.WithHooks(dbx.HookFuncs{
        AfterFunc: func(_ context.Context, event *dbx.HookEvent) {
            fmt.Println(event.Operation, event.Statement)
        },
    }),
)
```

## Benchmark

`dbx` 现在已经补齐了主要链路的 benchmark。

本地执行：

```bash
go test ./dbx -run '^$' -bench .
go test ./dbx/migrate -run '^$' -bench .
```

已覆盖：

- mapper metadata 与 scan 路径
- codec-aware 读取与写入 assignment
- query build 与 SQL render
- relation loading
- schema planning / validation / SQL preview
- SQL executor helper
- migration file source 与 runner

## 示例

- 示例总览页：[dbx examples](./examples)
- 可运行示例：
  - [examples/dbx/basic](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/basic)
  - [examples/dbx/codec](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/codec)
  - [examples/dbx/mutation](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/mutation)
  - [examples/dbx/query_advanced](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/query_advanced)
  - [examples/dbx/relations](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/relations)
  - [examples/dbx/migration](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/migration)
  - [examples/dbx/pure_sql](https://github.com/DaiYuANg/arcgo/tree/main/examples/dbx/pure_sql)

## 验证

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
