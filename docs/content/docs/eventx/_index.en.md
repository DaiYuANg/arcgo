---
title: 'eventx'
linkTitle: 'eventx'
description: 'In-Process Strongly Typed Event Bus'
weight: 4
---

## eventx

`eventx` is an in-memory strongly typed event bus for Go services.

## Roadmap

- Module roadmap: [eventx roadmap](./roadmap)
- Global roadmap: [ArcGo roadmap](../roadmap)

## Core Capabilities

- Generic type subscription: `Subscribe[T Event]`
- Synchronous publishing: `Publish`
- Asynchronous publishing with queue/workers: `PublishAsync`
- Optional parallel dispatch for handlers of the same event type
- Middleware pipeline (global + per-subscriber)
- Graceful shutdown and in-flight draining (`Close`)

## Event Contract

```go
type Event interface {
    Name() string
}
```

`Name()` is for semantics/observability. Routing is based on Go runtime type.

## Quick Start

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

## Dispatch Modes

### 1) Deterministic Serial Dispatch (Default)

```go
bus := eventx.New()
```

### 2) Per-Event Parallel Handler Dispatch

```go
bus := eventx.New(eventx.WithParallelDispatch(true))
```

## Async Publishing

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
    // apply backpressure or fallback
}
```

## Optional Observability

```go
otelObs := otelobs.New()
promObs := promobs.New()
obs := observabilityx.Multi(otelObs, promObs)

bus := eventx.New(
    eventx.WithObservability(obs),
)
```

Behavior notes:

- If async queue/workers are disabled, `PublishAsync` falls back to synchronous `Publish`.
- When queue is full, `PublishAsync` returns `ErrAsyncQueueFull`.

## Middleware

### Global Middleware

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

### Per-Subscriber Middleware

```go
_, _ = eventx.Subscribe(
    bus,
    handler,
    eventx.WithSubscriberMiddleware(mySubscriberMw),
)
```

Execution order:

- Global middleware wraps subscriber middleware.
- Middleware order is preserved as provided.

## Error Handling

- `Publish` returns aggregated handler errors (`errors.Join` semantics).
- Panics in handlers can be converted to errors via `RecoverMiddleware`.
- Async errors can be observed via `WithAsyncErrorHandler`.

## Unsubscription and Lifecycle

- `Subscribe` returns an idempotent `unsubscribe` function.
- `Close` stops new publishes, drains async queue, and waits for in-flight dispatches.
- Multiple calls to `Close` are safe.

## Useful API

- `bus.SubscriberCount()` to check active subscriptions.
- `eventx.ErrBusClosed`, `eventx.ErrNilEvent`, `eventx.ErrNilBus`, `eventx.ErrNilHandler` for typed error branches.

## Testing Tips

- Use serial dispatch in unit tests for deterministic ordering.
- Call `defer bus.Close()` in each test to avoid worker leaks.
- Use explicit event types in tests to avoid accidental shared subscriptions.

## FAQ

### Is `Event.Name()` used for routing?

No. Routing is based on the event's concrete Go type.
`Name()` is primarily for semantic metadata in logs/metrics/traces.

### Can a subscriber receive multiple event types?

Use separate `Subscribe[T]` calls for each type.
Each subscription binds to one generic type `T`.

### Can I recover panics from handlers?

Yes. Add `RecoverMiddleware()` globally or per-subscriber.

## Troubleshooting

### `PublishAsync` returns `ErrAsyncQueueFull`

Options:

- Increase queue size (`WithAsyncQueueSize`).
- Increase workers (`WithAsyncWorkers`).
- Add upstream backpressure/retry policy.
- Fall back to `Publish` for critical events.

### Handlers seem to run in unexpected order

- Serial mode preserves snapshot iteration order.
- Parallel mode (`WithParallelDispatch(true)`) doesn't guarantee ordering.
- If order matters, keep parallel dispatch disabled for that bus.

### `Close` hangs on shutdown

Usually caused by long-running handlers or blocking downstream calls.
Pass cancellable contexts in handlers and enforce timeouts.

## Anti-Patterns

- Using one global bus for all domains without clear ownership boundaries.
- Publishing high-volume firehose traffic without queue/backpressure planning.
- Enabling parallel dispatch when strict ordering guarantees are required.
- Ignoring async errors when business-critical events are involved.

## Examples

- [observability](https://github.com/DaiYuANg/archgo/tree/main/eventx/examples/observability): Event bus with optional OTel + Prometheus observability.
