---
title: 'eventx'
linkTitle: 'eventx'
description: '进程内强类型事件总线'
weight: 4
---

## eventx

`eventx` 是一个用于 Go 服务的内存强类型事件总线。

## 路线图

- 模块路线图：[eventx roadmap](./roadmap)
- 全局路线图：[ArcGo roadmap](../roadmap)

## 核心能力

- 泛型类型订阅：`Subscribe[T Event]`
- 同步发布：`Publish`
- 带队列/工作者的异步发布：`PublishAsync`
- 可选的同一事件类型处理器的并行分发
- 中间件管道（全局 + 每订阅者）
- 优雅关闭和进行中排空 (`Close`)

## 事件契约

```go
type Event interface {
    Name() string
}
```

`Name()` 用于语义/可观测性。路由基于 Go 运行时类型。

## 快速开始

```go
type UserCreated struct { ID int }
func (e UserCreated) Name() string { return "user.created" }

bus := eventx.New()
defer bus.Close()

unsub, err := eventx.Subscribe(bus, func(ctx context.Context, evt UserCreated) error {
    fmt.Println(evt.ID)
    return nil
})
if err != nil { panic(err) }
defer unsub()

_ = bus.Publish(context.Background(), UserCreated{ID: 42})
```

## 分发模式

### 1) 确定性串行分发（默认）

```go
bus := eventx.New()
```

### 2) 每事件并行处理器分发

```go
bus := eventx.New(eventx.WithParallelDispatch(true))
```

## 异步发布

```go
bus := eventx.New(
    eventx.WithAsyncWorkers(8),
    eventx.WithAsyncQueueSize(1024),
    eventx.WithAsyncErrorHandler(func(ctx context.Context, evt eventx.Event, err error) {
        // log/metric/report
    }),
)

err := bus.PublishAsync(ctx, UserCreated{ID: 1})
if errors.Is(err, eventx.ErrAsyncQueueFull) {
    // 应用背压或回退策略
}
```

## 可选可观测性

```go
otelObs := otelobs.New()
promObs := promobs.New()
obs := observabilityx.Multi(otelObs, promObs)

bus := eventx.New(
    eventx.WithObservability(obs),
)
```

行为说明：

- 如果异步队列/工作者被禁用，`PublishAsync` 回退到同步 `Publish`。
- 当队列满时，`PublishAsync` 返回 `ErrAsyncQueueFull`。

## 中间件

### 全局中间件

```go
bus := eventx.New(
    eventx.WithMiddleware(
        eventx.RecoverMiddleware(),
        eventx.ObserveMiddleware(func(ctx context.Context, evt eventx.Event, d time.Duration, err error) {
            // metrics
        }),
    ),
)
```

### 每订阅者中间件

```go
_, _ = eventx.Subscribe(
    bus,
    handler,
    eventx.WithSubscriberMiddleware(mySubscriberMw),
)
```

执行顺序：

- 全局中间件包装订阅者中间件。
- 中间件顺序按提供顺序保持。

## 错误处理

- `Publish` 返回聚合的处理器错误（`errors.Join` 语义）。
- 处理器中的 panic 可以通过 `RecoverMiddleware` 转换为错误。
- 异步错误可以通过 `WithAsyncErrorHandler` 观察。

## 取消订阅和生命周期

- `Subscribe` 返回一个幂等的 `unsubscribe` 函数。
- `Close` 停止新发布，排空异步队列，并等待进行中的分发。
- 多次调用 `Close` 是安全的。

## 有用的 API

- `bus.SubscriberCount()` 检查活动订阅。
- `eventx.ErrBusClosed`、`eventx.ErrNilEvent`、`eventx.ErrNilBus`、`eventx.ErrNilHandler` 用于类型化错误分支。

## 测试技巧

- 在单元测试中使用串行分发以获得确定性排序。
- 在每个测试中调用 `defer bus.Close()` 避免工作者泄漏。
- 在测试中使用显式事件类型以避免意外的共享订阅。

## 常见问题

### `Event.Name()` 用于路由吗？

不。路由基于事件的具体 Go 类型。
`Name()` 主要用于日志/指标/追踪的语义元数据。

### 一个订阅者可以接收多种事件类型吗？

对每种类型使用单独的 `Subscribe[T]` 调用。
每个订阅绑定到一个泛型类型 `T`。

### 我可以从处理器恢复 panic 吗？

可以。全局或每订阅者添加 `RecoverMiddleware()`。

## 故障排除

### `PublishAsync` 返回 `ErrAsyncQueueFull`

选项：

- 增加队列大小 (`WithAsyncQueueSize`)。
- 增加工作者 (`WithAsyncWorkers`)。
- 添加上游背压/重试策略。
- 对关键事件回退到 `Publish`。

### 处理器似乎在意外顺序下运行

- 串行模式保留快照迭代顺序。
- 并行模式 (`WithParallelDispatch(true)`) 不保证排序。
- 如果顺序很重要，对该总线保持并行分发禁用。

### `Close` 在关闭时挂起

通常由长时间运行的处理器或阻塞的下游调用引起。
在处理器中传递可取消的 context 并强制执行超时。

## 反模式

- 使用一个全局总线用于所有域而没有清晰的拥有权边界。
- 发布高容量 firehose 流量而没有队列/背压规划。
- 在需要严格顺序保证的同时启用并行分发。
- 当涉及业务关键事件时忽略异步错误。

## 示例

- [observability](https://github.com/DaiYuANg/archgo/tree/main/eventx/examples/observability): 带有可选 OTel + Prometheus 可观测性的事件总线。
