---
title: 'observabilityx 快速开始'
linkTitle: 'getting-started'
description: '创建可观测性门面、启动 span，并记录声明式指标'
weight: 2
---

## 快速开始

把 `observabilityx.Nop()` 当作安全默认值；当需要接入真实后端（OTel、Prometheus，或 `Multi`）时，在应用/模块初始化阶段把 backend 注入进去即可。

`observabilityx v0.2.0` 改成了声明式 metric spec。推荐流程是：

1. 先声明 metric spec
2. 再从 backend 拿到 instrument
3. 最后通过 instrument 记录值

## 示例（通过 Multi 组合 OTel + Prometheus）

```go
package main

import (
	"context"

	"github.com/DaiYuANg/arcgo/observabilityx"
	otelobs "github.com/DaiYuANg/arcgo/observabilityx/otel"
	promobs "github.com/DaiYuANg/arcgo/observabilityx/prometheus"
)

func main() {
	otelBackend := otelobs.New()
	promBackend := promobs.New(promobs.WithNamespace("app"))

	obs := observabilityx.Multi(otelBackend, promBackend)

	ctx, span := obs.StartSpan(context.Background(), "demo.operation", observabilityx.String("feature", "multi"))
	defer span.End()

	requests := obs.Counter(
		observabilityx.NewCounterSpec(
			"demo_counter_total",
			observabilityx.WithDescription("Demo 请求总次数。"),
			observabilityx.WithLabelKeys("result"),
		),
	)
	inflight := obs.UpDownCounter(
		observabilityx.NewUpDownCounterSpec(
			"demo_inflight",
			observabilityx.WithDescription("当前进行中的 demo 请求数。"),
			observabilityx.WithLabelKeys("result"),
		),
	)
	duration := obs.Histogram(
		observabilityx.NewHistogramSpec(
			"demo_duration_ms",
			observabilityx.WithDescription("Demo 请求耗时，单位毫秒。"),
			observabilityx.WithUnit("ms"),
			observabilityx.WithLabelKeys("result"),
		),
	)
	queueDepth := obs.Gauge(
		observabilityx.NewGaugeSpec(
			"demo_queue_depth",
			observabilityx.WithDescription("当前 demo 队列深度。"),
			observabilityx.WithLabelKeys("result"),
		),
	)

	inflight.Add(ctx, 1, observabilityx.String("result", "ok"))
	defer inflight.Add(ctx, -1, observabilityx.String("result", "ok"))

	requests.Add(ctx, 1, observabilityx.String("result", "ok"))
	duration.Record(ctx, 12, observabilityx.String("result", "ok"))
	queueDepth.Set(ctx, 3, observabilityx.String("result", "ok"))
}
```

## 可运行示例（仓库）

- [examples/observabilityx/multi](https://github.com/DaiYuANg/arcgo/tree/main/examples/observabilityx/multi)
