package observabilityx_test

import (
	"context"
	"testing"

	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/stretchr/testify/require"
)

func TestNop(t *testing.T) {
	t.Parallel()

	obs := observabilityx.Nop()
	require.NotNil(t, obs)
	require.NotNil(t, obs.Logger())

	ctx, span := obs.StartSpan(context.TODO(), "test")
	require.NotNil(t, ctx)
	require.NotNil(t, span)

	obs.Counter(observabilityx.NewCounterSpec("counter", observabilityx.WithLabelKeys("result"))).
		Add(context.Background(), 1, observabilityx.String("result", "ok"))
	obs.UpDownCounter(observabilityx.NewUpDownCounterSpec("inflight", observabilityx.WithLabelKeys("result"))).
		Add(context.Background(), 1, observabilityx.String("result", "ok"))
	obs.Histogram(observabilityx.NewHistogramSpec("histogram", observabilityx.WithLabelKeys("result"))).
		Record(context.Background(), 1.0, observabilityx.String("result", "ok"))
	obs.Gauge(observabilityx.NewGaugeSpec("queue_depth", observabilityx.WithLabelKeys("result"))).
		Set(context.Background(), 1.0, observabilityx.String("result", "ok"))

	span.SetAttributes(observabilityx.String("k", "v"))
	span.RecordError(context.Canceled)
	span.End()
}
