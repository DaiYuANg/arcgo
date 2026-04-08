---
title: 'dix 指标与可观测性'
linkTitle: 'metrics-and-observability'
description: '使用 dix/metrics 将内部运行时事件接入 Prometheus 或 OpenTelemetry'
weight: 4
---

## Metrics And Observability

`dix` 核心本身不会内置 exporter、HTTP server 或 OTel pipeline。  
新加的独立子包 `github.com/DaiYuANg/arcgo/dix/metrics` 只负责把 `dix` 内部运行时事件转换成 `observabilityx` 指标：

- build
- start
- stop
- health check
- state transition

这意味着：

- 如果你已经有 Prometheus backend，就把 `dix/metrics` 接到 `observabilityx/prometheus`
- 如果你已经有 OTel meter，就把 `dix/metrics` 接到 `observabilityx/otel`
- `/metrics`、OTLP exporter、collector、HTTP 路由都继续由你的外部应用自己管理

## Install

```bash
go get github.com/DaiYuANg/arcgo/dix@latest
go get github.com/DaiYuANg/arcgo/dix/metrics@latest
go get github.com/DaiYuANg/arcgo/observabilityx/prometheus@latest
go get github.com/DaiYuANg/arcgo/observabilityx/otel@latest
```

## 最小接入：Prometheus

下面的示例里，`dix/metrics` 只负责向 `promobs.Adapter` 写指标；`/metrics` 仍然挂在你自己的 HTTP server 上。

```go
package main

import (
	"context"
	"net/http"

	"github.com/DaiYuANg/arcgo/dix"
	dixmetrics "github.com/DaiYuANg/arcgo/dix/metrics"
	promobs "github.com/DaiYuANg/arcgo/observabilityx/prometheus"
)

func main() {
	prom := promobs.New(promobs.WithNamespace("arcgo"))

	app := dix.New(
		"orders",
		dixmetrics.WithObservability(prom),
	)

	http.Handle("/metrics", prom.Handler())

	rt, err := app.Start(context.Background())
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = rt.Stop(context.Background())
	}()
}
```

如果你已经在用 `httpx`、`chi`、`gin`、`echo`、`fiber` 等 server，也一样只是把 `prom.Handler()` 挂进去，不需要 `dix` 额外启动一个 metrics server。

## 最小接入：OpenTelemetry

如果你已经有自己的 OTel exporter / SDK 初始化，只需要把 meter 封装成 `observabilityx/otel` backend，然后接入 `dix/metrics`：

```go
package main

import (
	"context"

	"github.com/DaiYuANg/arcgo/dix"
	dixmetrics "github.com/DaiYuANg/arcgo/dix/metrics"
	otelobs "github.com/DaiYuANg/arcgo/observabilityx/otel"
)

func main() {
	obs := otelobs.New()

	app := dix.New(
		"orders",
		dixmetrics.WithObservability(obs),
	)

	rt, err := app.Start(context.Background())
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = rt.Stop(context.Background())
	}()
}
```

更底层一点时，你也可以不用 `dix/metrics.WithObservability(...)`，而是直接传自定义 observer：

```go
app := dix.New(
	"orders",
	dix.WithObserver(dixmetrics.NewObserver(obs)),
)
```

## 默认指标名

默认前缀是 `dix_`。

- `dix_build_total`
- `dix_build_duration_ms`
- `dix_build_modules`
- `dix_build_providers`
- `dix_build_hooks`
- `dix_build_setups`
- `dix_build_invokes`
- `dix_start_total`
- `dix_start_duration_ms`
- `dix_start_registered_hooks`
- `dix_start_completed_hooks`
- `dix_start_rollback_total`
- `dix_stop_total`
- `dix_stop_duration_ms`
- `dix_stop_registered_hooks`
- `dix_stop_shutdown_errors`
- `dix_stop_hook_error_total`
- `dix_health_check_total`
- `dix_health_check_duration_ms`
- `dix_state_transition_total`

## 默认标签

常见标签包括：

- `app`
- `profile`
- `version`
- `result`

额外标签：

- health check：`kind`，可选 `check`
- state transition：`from`、`to`

## 自定义前缀与标签

`dix/metrics` 支持几个常见定制项：

- `dixmetrics.WithMetricPrefix("arc_dix")`
- `dixmetrics.WithVersionAttribute(false)`
- `dixmetrics.WithHealthCheckNameAttribute(false)`

例如：

```go
app := dix.New(
	"orders",
	dixmetrics.WithObservability(
		prom,
		dixmetrics.WithMetricPrefix("arc_dix"),
		dixmetrics.WithHealthCheckNameAttribute(false),
	),
)
```

## 设计边界

- `dix` 核心只发出 observer event，不依赖具体监控后端
- `dix/metrics` 只做事件到指标的翻译
- Prometheus `/metrics` handler 仍由 `observabilityx/prometheus` 提供
- OTel exporter / SDK 初始化仍由你的应用负责

如果你需要更细粒度或更定制的埋点，可以直接实现自己的 `dix.Observer`，再通过 `dix.WithObserver(...)` 挂进去。

## Next

- `observabilityx` 总览：[observabilityx](../observabilityx)
- Prometheus `/metrics` handler：[observabilityx Prometheus 指标端点](../observabilityx/prometheus-metrics)
- OTel backend：[observabilityx OpenTelemetry 后端](../observabilityx/otel-backend)
