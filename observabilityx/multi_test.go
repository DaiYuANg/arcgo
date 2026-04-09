package observabilityx_test

import (
	"context"
	"testing"

	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/stretchr/testify/require"
)

func TestMulti(t *testing.T) {
	t.Parallel()

	a := newTestBackend()
	b := newTestBackend()

	obs := observabilityx.Multi(a, b)
	require.NotNil(t, obs)
	require.NotNil(t, obs.Logger())

	ctx, span := obs.StartSpan(context.Background(), "test")
	require.NotNil(t, ctx)
	require.NotNil(t, span)

	obs.Counter(observabilityx.NewCounterSpec("counter")).Add(ctx, 1)
	obs.UpDownCounter(observabilityx.NewUpDownCounterSpec("inflight")).Add(ctx, 1)
	obs.Histogram(observabilityx.NewHistogramSpec("histogram")).Record(ctx, 1)
	obs.Gauge(observabilityx.NewGaugeSpec("queue_depth")).Set(ctx, 1)
	span.End()

	require.EqualValues(t, 1, a.spanCount.Load())
	require.EqualValues(t, 1, b.spanCount.Load())
	require.EqualValues(t, 1, a.counterCount.Load())
	require.EqualValues(t, 1, b.counterCount.Load())
	require.EqualValues(t, 1, a.upDownCounterCount.Load())
	require.EqualValues(t, 1, b.upDownCounterCount.Load())
	require.EqualValues(t, 1, a.histogramCount.Load())
	require.EqualValues(t, 1, b.histogramCount.Load())
	require.EqualValues(t, 1, a.gaugeCount.Load())
	require.EqualValues(t, 1, b.gaugeCount.Load())
}
