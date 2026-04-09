// Package main demonstrates combining multiple observability backends in one application.
package main

import (
	"context"
	"log/slog"

	"github.com/DaiYuANg/arcgo/httpx"
	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/DaiYuANg/arcgo/httpx/adapter/std"
	"github.com/DaiYuANg/arcgo/observabilityx"
	otelobs "github.com/DaiYuANg/arcgo/observabilityx/otel"
	promobs "github.com/DaiYuANg/arcgo/observabilityx/prometheus"
)

func main() {
	prom := promobs.New(promobs.WithNamespace("observability_example"))
	obs := observabilityx.Multi(otelobs.New(), prom)

	ctx, span := obs.StartSpan(context.TODO(), "demo.operation", observabilityx.String("feature", "multi-backend"))
	defer span.End()

	obs.Counter(observabilityx.NewCounterSpec("demo_counter_total", observabilityx.WithLabelKeys("result"))).
		Add(ctx, 1, observabilityx.String("result", "ok"))
	obs.UpDownCounter(observabilityx.NewUpDownCounterSpec("demo_inflight", observabilityx.WithLabelKeys("result"))).
		Add(ctx, 1, observabilityx.String("result", "ok"))
	obs.Histogram(observabilityx.NewHistogramSpec("demo_duration_ms", observabilityx.WithUnit("ms"), observabilityx.WithLabelKeys("result"))).
		Record(ctx, 12, observabilityx.String("result", "ok"))
	obs.Gauge(observabilityx.NewGaugeSpec("demo_queue_depth", observabilityx.WithLabelKeys("result"))).
		Set(ctx, 3, observabilityx.String("result", "ok"))

	stdAdapter := std.New(nil, adapter.HumaOptions{DisableDocsRoutes: true})
	metricsServer := httpx.New(
		httpx.WithAdapter(stdAdapter),
	)
	stdAdapter.Router().Handle("/metrics", prom.Handler())

	slog.Info("httpx metrics route registered", "route", "GET /metrics")
	_ = metricsServer
}
