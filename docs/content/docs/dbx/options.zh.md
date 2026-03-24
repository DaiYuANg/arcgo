---
title: 'dbx Options'
linkTitle: 'Options'
description: 'dbx 的函数式 Options、预设与 Open'
weight: 8
---

## Options

Options 使用函数式 Option 模式，可组合（后者覆盖前者）。

## Open（dbx 管理连接）

使用 `Open` 时，dbx 负责管理连接，无需传入 `*sql.DB`。

```go
db, err := dbx.Open(
    dbx.WithDriver("sqlite"),
    dbx.WithDSN("file:app.db"),
    dbx.WithDialect(sqlite.Dialect{}),
    dbx.ApplyOptions(dbx.WithDebug(true)),
)
if err != nil {
    return err
}
defer db.Close()
```

必须提供：`WithDriver`、`WithDSN`、`WithDialect`。任一缺失时 `Open` 返回 `ErrMissingDriver`、`ErrMissingDSN` 或 `ErrMissingDialect`。通过 `ApplyOptions` 传入 `Option`（WithLogger、WithHooks、WithDebug）。

## Presets 预设

| Preset | 适用场景 |
|--------|----------|
| `DefaultOptions()` | 显式默认（返回 `nil`），等同于不传 options。 |
| `ProductionOptions()` | 生产：debug 关闭。 |
| `TestOptions()` | 测试：debug 开启，便于查看 SQL 日志。 |

## Usage 使用

```go
// 默认
core := dbx.New(raw, dialect)

// 预设 + 覆盖
core, err := dbx.NewWithOptions(raw, dialect, append(dbx.TestOptions(), dbx.WithLogger(myLogger))...)
if err != nil {
    return err
}

// 自定义组合
core, err := dbx.NewWithOptions(raw, dialect,
    dbx.WithLogger(logger),
    dbx.WithDebug(true),
    dbx.WithHooks(dbx.HookFuncs{AfterFunc: myAfterHook}),
)
if err != nil {
    return err
}
```

## Options 表

| Option | 默认 | 说明 |
|--------|------|------|
| `WithLogger(logger)` | `slog.Default()` | 操作事件日志。debug=false 时仅记录错误。 |
| `WithHooks(hooks...)` | `[]` | 每个操作前后执行的 hooks，可叠加。详见 [Observability 可观测性](./observability)（慢查询、Metadata trace_id/request_id 等）。 |
| `WithDebug(enabled)` | `false` | 为 true 时，所有操作以 Debug 级别记录。开发/测试中用于查看 SQL。 |
| `WithNodeID(nodeID)` | 主机名自动推导 | DB 节点标识，内建 Snowflake 生成器会使用它。 |
| `WithIDGenerator(generator)` | 内建生成器 | 覆盖当前 DB 实例的内建 ID 生成器。 |

## Composition 组合

Options 按顺序应用，后者覆盖前者。Hooks 为追加，不替换：

```go
// logger 来自 myLogger，debug 开启，hooks = [h1, h2]
dbx.NewWithOptions(raw, d,
    dbx.WithHooks(h1),
    dbx.WithLogger(myLogger),
    dbx.WithDebug(true),
    dbx.WithHooks(h2),
)
```

`WithNodeID` 与 `WithIDGenerator` 互斥，同时配置会返回错误。

## 错误处理

```go
core, err := dbx.NewWithOptions(raw, d, dbx.WithNodeID(0))
if err != nil {
    if errors.Is(err, dbx.ErrInvalidNodeID) {
        var out *dbx.NodeIDOutOfRangeError
        if errors.As(err, &out) {
            // out.NodeID, out.Min, out.Max
        }
    }
    return err
}
_ = core
```
