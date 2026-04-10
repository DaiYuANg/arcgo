---
title: 'httpx Middleware and Observability'
linkTitle: 'middleware-and-observability'
description: 'httpx 的 Prometheus / OpenTelemetry middleware'
weight: 6
---

## Middleware 与可观测性

`httpx/middleware` 提供了一些围绕标准 `net/http` handler 的轻量能力。

当前主要的可观测性 helper 有：

- `PrometheusMiddleware`
- `OpenTelemetryMiddleware`

## 面向路由模板的标签与 span name

如果指标和 trace 直接使用原始请求路径，动态 ID 很容易造成高基数：

- `/users/1`
- `/users/2`
- `/users/3`

对于 `httpx` 服务，通常更希望聚合到路由模板：

- `/users/{id}`

`httpx/middleware` 通过 `WithHTTPXRoutePattern(server)` 支持这一点。

## Prometheus 示例

```go
server := httpx.New(...)

handler := middleware.PrometheusMiddleware(
	serverRouter,
	middleware.WithHTTPXRoutePattern(server),
)
```

开启 route pattern 解析后，指标会使用注册时的路由模板，而不是原始 path。

## OpenTelemetry 示例

```go
handler := middleware.OpenTelemetryMiddleware(
	serverRouter,
	middleware.WithHTTPXRoutePattern(server),
)
```

开启 route pattern 解析后：

- span name 使用路由模板
- span 上会带 `http.route`

## 为什么重要

- 降低指标标签基数
- dashboard 更干净
- trace 聚合更稳定
- 与服务端路由注册语义保持一致

## 说明

- middleware 包是可选的。
- 它包装的是标准 handler，不是直接改写 `httpx` typed route 注册。
- route pattern 解析是 opt-in 的；不传这个 option，原有行为不变。
