---
title: 'observabilityx OpenTelemetry 后端'
linkTitle: 'otel-backend'
description: '使用 OTel 后端，并配置自定义 tracer/meter 与声明式 metric spec'
weight: 4
---

## OpenTelemetry 后端

`observabilityx/otel` 是基于 OpenTelemetry 的 `observabilityx.Observability` 实现。

从 `v0.2.0` 开始，指标改成先声明 typed spec，再通过 instrument 记录值。

默认会使用：

- `otel.Tracer("github.com/DaiYuANg/arcgo")`
- `otel.Meter("github.com/DaiYuANg/arcgo")`

如果你的应用自行初始化了 OTel SDK provider/exporter，也可以把自定义 tracer/meter 注入进去。

## 示例（自定义 tracer/meter）

```go
package main

import (
	"context"

	"github.com/DaiYuANg/arcgo/observabilityx"
	otelobs "github.com/DaiYuANg/arcgo/observabilityx/otel"
	"go.opentelemetry.io/otel"
)

func main() {
	obs := otelobs.New(
		otelobs.WithTracer(otel.Tracer("my-service")),
		otelobs.WithMeter(otel.Meter("my-service")),
	)

	ctx, span := obs.StartSpan(context.Background(), "db.query", observabilityx.String("table", "users"))
	defer span.End()

	queries := obs.Counter(
		observabilityx.NewCounterSpec(
			"db_queries_total",
			observabilityx.WithDescription("数据库查询总次数。"),
			observabilityx.WithLabelKeys("result"),
		),
	)
	queryDuration := obs.Histogram(
		observabilityx.NewHistogramSpec(
			"db_query_duration_ms",
			observabilityx.WithDescription("数据库查询耗时，单位毫秒。"),
			observabilityx.WithUnit("ms"),
			observabilityx.WithLabelKeys("result"),
		),
	)

	queries.Add(ctx, 1, observabilityx.String("result", "ok"))
	queryDuration.Record(ctx, 12, observabilityx.String("result", "ok"))
}
```
