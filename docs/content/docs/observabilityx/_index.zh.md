---
title: 'observabilityx'
linkTitle: 'observabilityx'
description: '可选可观测性抽象（OTel/Prometheus）'
weight: 7
---

## observabilityx

`observabilityx` 为日志/追踪/指标提供可选的统一门面。

## 路线图

- 模块路线图：[observabilityx roadmap](./roadmap)
- 全局路线图：[ArcGo roadmap](../roadmap)

## 为什么

- 保持 `authx`、`eventx`、`configx` API 稳定。
- 使可观测性后端可选。
- 避免强制业务代码使用一个遥测栈。

## 后端

- `observabilityx.Nop()` - 默认无操作后端。
- `observabilityx/otel` - OpenTelemetry 后端（trace + metric）。
- `observabilityx/prometheus` - Prometheus 后端（指标 + `/metrics` 处理器）。

## 组合多个后端

```go
otelObs := otelobs.New()
promObs := promobs.New()

obs := observabilityx.Multi(otelObs, promObs)
```

## 接入包

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

## Prometheus 指标端点

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

## 示例

- [multi](https://github.com/DaiYuANg/archgo/tree/main/observabilityx/examples/multi): 组合 OTel + Prometheus 后端。
