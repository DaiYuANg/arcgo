package otel

import (
	"context"
	"testing"

	"github.com/DaiYuANg/archgo/observabilityx"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	obs := New()
	require.NotNil(t, obs)
	require.NotNil(t, obs.Logger())
}

func TestAdapterMethods(t *testing.T) {
	t.Parallel()

	obs := New()

	ctx, span := obs.StartSpan(context.Background(), "test.operation", observabilityx.String("k", "v"))
	require.NotNil(t, ctx)
	require.NotNil(t, span)

	obs.AddCounter(ctx, "test_counter_total", 1, observabilityx.String("result", "ok"))
	obs.RecordHistogram(ctx, "test_duration_ms", 12, observabilityx.String("result", "ok"))

	span.SetAttributes(observabilityx.Bool("done", true))
	span.End()
}
