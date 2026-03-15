package prometheus

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/DaiYuANg/archgo/observabilityx"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestAdapterMetrics(t *testing.T) {
	t.Parallel()

	registry := prom.NewRegistry()
	obs := New(
		WithRegisterer(registry),
		WithGatherer(registry),
		WithNamespace("arcgo_test"),
	)

	obs.AddCounter(context.Background(), "authx_authenticate_total", 1, observabilityx.String("result", "ok"))
	obs.RecordHistogram(context.Background(), "authx_authenticate_duration_ms", 10, observabilityx.String("result", "ok"))

	metrics, err := registry.Gather()
	require.NoError(t, err)
	require.NotEmpty(t, metrics)
}

func TestAdapterHandler(t *testing.T) {
	t.Parallel()

	registry := prom.NewRegistry()
	obs := New(WithRegisterer(registry), WithGatherer(registry))

	obs.AddCounter(context.Background(), "eventx_publish_total", 1)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	obs.Handler().ServeHTTP(w, req)

	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), "eventx_publish_total")
}
