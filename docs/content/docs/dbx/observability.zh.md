---
title: 'dbx 可观测性与 Hooks'
linkTitle: 'Observability'
description: 'Hooks、HookEvent、Duration、Metadata 用于日志与链路追踪'
weight: 9
---

## 可观测性与 Hooks

Hooks 在每个 DB 操作前后执行。可用于日志、指标、链路追踪和慢查询检测。

## HookEvent

`HookEvent` 携带操作详情：

| 字段 | 说明 |
|------|------|
| `Operation` | query, exec, query_row, begin_tx, commit_tx, rollback_tx, auto_migrate, validate_schema |
| `Statement` | 高层 statement 名称（若有） |
| `SQL` | 实际 SQL 字符串 |
| `Args` | 绑定参数 |
| `Table` | 目标表（若已知） |
| `StartedAt` | 操作开始时间 |
| `Duration` | 耗时（在 After 中设置） |
| `RowsAffected` | exec 操作影响行数 |
| `Err` | 错误（若有） |
| `Metadata` | 任意 key-value，用于 trace_id、request_id 等 |

## Duration 与 StartedAt

使用 `StartedAt` 和 `Duration` 做慢查询检测和耗时统计：

```go
dbx.NewWithOptions(raw, dialect,
    dbx.WithHooks(dbx.HookFuncs{
        AfterFunc: func(_ context.Context, event *dbx.HookEvent) {
            if event.Duration > 100*time.Millisecond {
                slog.Warn("slow query", "sql", event.SQL, "duration", event.Duration)
            }
        },
    }),
)
```

## Metadata 传递 Trace 与 Request ID

在 Before 中设置 `Metadata` 可传递 trace_id、request_id 或其他上下文。值会出现在 dbx 日志中：

```go
dbx.NewWithOptions(raw, dialect,
    dbx.WithHooks(dbx.HookFuncs{
        BeforeFunc: func(ctx context.Context, event *dbx.HookEvent) (context.Context, error) {
            if tid := ctx.Value("trace_id"); tid != nil {
                event.SetMetadata("trace_id", tid)
            }
            if rid := ctx.Value("request_id"); rid != nil {
                event.SetMetadata("request_id", rid)
            }
            return ctx, nil
        },
    }),
)
```

`SetMetadata` 会在需要时初始化 map，避免 nil map panic。

## Context

`Before` 和 `After` 会收到 `context.Context`。Hooks 可从 context 中读取 trace/request ID（例如来自中间件），并复制到 `event.Metadata` 供日志或指标使用。
