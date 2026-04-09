package otel_test

import (
	"context"
	"testing"

	"github.com/DaiYuANg/arcgo/observabilityx"
	otelobs "github.com/DaiYuANg/arcgo/observabilityx/otel"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	obs := otelobs.New()
	require.NotNil(t, obs)
	require.NotNil(t, obs.Logger())
}

func TestAdapterMethods(t *testing.T) {
	t.Parallel()

	obs := otelobs.New()

	ctx, span := obs.StartSpan(context.Background(), "test.operation", observabilityx.String("k", "v"))
	require.NotNil(t, ctx)
	require.NotNil(t, span)

	obs.Counter(observabilityx.NewCounterSpec("test_counter_total", observabilityx.WithLabelKeys("result"))).
		Add(ctx, 1, observabilityx.String("result", "ok"))
	obs.UpDownCounter(observabilityx.NewUpDownCounterSpec("test_inflight", observabilityx.WithLabelKeys("result"))).
		Add(ctx, 1, observabilityx.String("result", "ok"))
	obs.Histogram(observabilityx.NewHistogramSpec("test_duration_ms", observabilityx.WithLabelKeys("result"))).
		Record(ctx, 12, observabilityx.String("result", "ok"))
	obs.Gauge(observabilityx.NewGaugeSpec("test_queue_depth", observabilityx.WithLabelKeys("result"))).
		Set(ctx, 3, observabilityx.String("result", "ok"))

	span.SetAttributes(observabilityx.Bool("done", true))
	span.End()
}
