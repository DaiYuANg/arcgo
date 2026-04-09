---
title: 'observabilityx Getting Started'
linkTitle: 'getting-started'
description: 'Create an observability facade, start spans, and record declared metrics'
weight: 2
---

## Getting started

Use `observabilityx.Nop()` as a safe default, and switch to a real backend (OTel, Prometheus, or `Multi`) by wiring the backend into your app/module.

`observabilityx v0.2.0` uses declared metric specs. The normal flow is:

1. declare a metric spec
2. ask the backend for an instrument
3. record values through the instrument

## Example (OTel + Prometheus via Multi)

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
			observabilityx.WithDescription("Total number of demo requests."),
			observabilityx.WithLabelKeys("result"),
		),
	)
	inflight := obs.UpDownCounter(
		observabilityx.NewUpDownCounterSpec(
			"demo_inflight",
			observabilityx.WithDescription("Current number of in-flight demo requests."),
			observabilityx.WithLabelKeys("result"),
		),
	)
	duration := obs.Histogram(
		observabilityx.NewHistogramSpec(
			"demo_duration_ms",
			observabilityx.WithDescription("Demo request duration in milliseconds."),
			observabilityx.WithUnit("ms"),
			observabilityx.WithLabelKeys("result"),
		),
	)
	queueDepth := obs.Gauge(
		observabilityx.NewGaugeSpec(
			"demo_queue_depth",
			observabilityx.WithDescription("Current demo queue depth."),
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

## Runnable example (repository)

- [examples/observabilityx/multi](https://github.com/DaiYuANg/arcgo/tree/main/examples/observabilityx/multi)
