---
title: 'observabilityx OpenTelemetry backend'
linkTitle: 'otel-backend'
description: 'Use the OTel backend with custom tracer/meter and declared metric specs'
weight: 4
---

## OpenTelemetry backend

`observabilityx/otel` is an OTel-backed implementation of `observabilityx.Observability`.

With `v0.2.0`, metrics are declared through typed specs before values are recorded.

By default it uses:

- `otel.Tracer("github.com/DaiYuANg/arcgo")`
- `otel.Meter("github.com/DaiYuANg/arcgo")`

You can also supply your own tracer/meter if your application sets up an SDK provider/exporter.

## Example (custom tracer/meter)

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
			observabilityx.WithDescription("Total number of database queries."),
			observabilityx.WithLabelKeys("result"),
		),
	)
	queryDuration := obs.Histogram(
		observabilityx.NewHistogramSpec(
			"db_query_duration_ms",
			observabilityx.WithDescription("Database query duration in milliseconds."),
			observabilityx.WithUnit("ms"),
			observabilityx.WithLabelKeys("result"),
		),
	)

	queries.Add(ctx, 1, observabilityx.String("result", "ok"))
	queryDuration.Record(ctx, 12, observabilityx.String("result", "ok"))
}
```
