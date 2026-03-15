---
title: 'observabilityx'
linkTitle: 'observabilityx'
description: 'Optional Observability Abstraction (OTel/Prometheus)'
weight: 7
---

## observabilityx

`observabilityx` provides an optional unified facade for logging/tracing/metrics.

## Roadmap

- Module roadmap: [observabilityx roadmap](./roadmap)
- Global roadmap: [ArcGo roadmap](../roadmap)

## Why

- Keep `authx`, `eventx`, `configx` APIs stable.
- Make observability backends optional.
- Avoid forcing business code into one telemetry stack.

## Backends

- `observabilityx.Nop()` - Default no-op backend.
- `observabilityx/otel` - OpenTelemetry backend (trace + metric).
- `observabilityx/prometheus` - Prometheus backend (metrics + `/metrics` handler).

## Composing Multiple Backends

```go
otelObs := otelobs.New()
promObs := promobs.New()

obs := observabilityx.Multi(otelObs, promObs)
```

## Adopting Packages

```go
manager, _ := authx.NewManager(
    authx.WithObservability(obs),
    authx.WithProvider(provider),
)

bus := eventx.New(
    eventx.WithObservability(obs),
)

var cfg AppConfig
_ = configx.Load(&cfg,
    configx.WithObservability(obs),
    configx.WithFiles("config.yaml"),
)
```

## Prometheus Metrics Endpoint

```go
promObs := promobs.New()

metricsServer := httpx.NewServer(
    httpx.WithAdapter(std.New()),
    httpx.WithOpenAPIDocs(false),
)
metricsServer.Adapter().Handle(httpx.MethodGet, "/metrics", func(
    ctx context.Context,
    w http.ResponseWriter,
    r *http.Request,
) error {
    promObs.Handler().ServeHTTP(w, r)
    return nil
})
```

## Examples

- [multi](https://github.com/DaiYuANg/archgo/tree/main/observabilityx/examples/multi): Compose OTel + Prometheus backends.
