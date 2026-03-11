package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/DaiYuANg/arcgo/httpx"
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

	obs.AddCounter(ctx, "demo_counter_total", 1, observabilityx.String("result", "ok"))
	obs.RecordHistogram(ctx, "demo_duration_ms", 12, observabilityx.String("result", "ok"))

	metricsServer := httpx.New(
		httpx.WithAdapter(std.New()),
		httpx.WithOpenAPIDocs(false),
	)
	metricsServer.Adapter().Handle(httpx.MethodGet, "/metrics", func(
		ctx context.Context,
		w http.ResponseWriter,
		r *http.Request,
	) error {
		_ = ctx
		prom.Handler().ServeHTTP(w, r)
		return nil
	})

	fmt.Println("httpx metrics route registered: GET /metrics")
	_ = metricsServer
}
