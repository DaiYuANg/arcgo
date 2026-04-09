package prometheus_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DaiYuANg/arcgo/observabilityx"
	promobs "github.com/DaiYuANg/arcgo/observabilityx/prometheus"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestAdapterMetrics(t *testing.T) {
	t.Parallel()

	registry := prom.NewRegistry()
	obs := promobs.New(
		promobs.WithRegisterer(registry),
		promobs.WithGatherer(registry),
		promobs.WithNamespace("arcgo_test"),
	)

	obs.Counter(observabilityx.NewCounterSpec("authx_authenticate_total", observabilityx.WithLabelKeys("result"))).
		Add(context.Background(), 1, observabilityx.String("result", "ok"))
	obs.UpDownCounter(observabilityx.NewUpDownCounterSpec("authx_inflight", observabilityx.WithLabelKeys("result"))).
		Add(context.Background(), 1, observabilityx.String("result", "ok"))
	obs.Histogram(observabilityx.NewHistogramSpec("authx_authenticate_duration_ms", observabilityx.WithLabelKeys("result"))).
		Record(context.Background(), 10, observabilityx.String("result", "ok"))
	obs.Gauge(observabilityx.NewGaugeSpec("authx_queue_depth", observabilityx.WithLabelKeys("result"))).
		Set(context.Background(), 3, observabilityx.String("result", "ok"))

	metrics, err := registry.Gather()
	require.NoError(t, err)
	require.NotEmpty(t, metrics)
}

func TestAdapterHandler(t *testing.T) {
	t.Parallel()

	registry := prom.NewRegistry()
	obs := promobs.New(promobs.WithRegisterer(registry), promobs.WithGatherer(registry))

	obs.Counter(observabilityx.NewCounterSpec("eventx_publish_total")).Add(context.Background(), 1)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", http.NoBody)
	w := httptest.NewRecorder()
	obs.Handler().ServeHTTP(w, req)

	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), "eventx_publish_total")
}
