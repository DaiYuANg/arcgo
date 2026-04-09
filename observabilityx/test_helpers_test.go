package observabilityx_test

import (
	"context"
	"log/slog"
	"sync/atomic"

	"github.com/DaiYuANg/arcgo/observabilityx"
)

type testBackend struct {
	logger *slog.Logger

	spanCount          atomic.Int64
	counterCount       atomic.Int64
	upDownCounterCount atomic.Int64
	histogramCount     atomic.Int64
	gaugeCount         atomic.Int64
}

func newTestBackend() *testBackend {
	return &testBackend{}
}

func (t *testBackend) Logger() *slog.Logger {
	return observabilityx.NormalizeLogger(t.logger)
}

func (t *testBackend) StartSpan(ctx context.Context, _ string, _ ...observabilityx.Attribute) (context.Context, observabilityx.Span) {
	t.spanCount.Add(1)
	if ctx == nil {
		ctx = context.Background()
	}
	return ctx, testSpan{}
}

func (t *testBackend) Counter(spec observabilityx.CounterSpec) observabilityx.Counter {
	_ = spec
	return testCounter{backend: t}
}

func (t *testBackend) UpDownCounter(spec observabilityx.UpDownCounterSpec) observabilityx.UpDownCounter {
	_ = spec
	return testUpDownCounter{backend: t}
}

func (t *testBackend) Histogram(spec observabilityx.HistogramSpec) observabilityx.Histogram {
	_ = spec
	return testHistogram{backend: t}
}

func (t *testBackend) Gauge(spec observabilityx.GaugeSpec) observabilityx.Gauge {
	_ = spec
	return testGauge{backend: t}
}

type testCounter struct {
	backend *testBackend
}

func (t testCounter) Add(context.Context, int64, ...observabilityx.Attribute) {
	t.backend.counterCount.Add(1)
}

type testUpDownCounter struct {
	backend *testBackend
}

func (t testUpDownCounter) Add(context.Context, int64, ...observabilityx.Attribute) {
	t.backend.upDownCounterCount.Add(1)
}

type testHistogram struct {
	backend *testBackend
}

func (t testHistogram) Record(context.Context, float64, ...observabilityx.Attribute) {
	t.backend.histogramCount.Add(1)
}

type testGauge struct {
	backend *testBackend
}

func (t testGauge) Set(context.Context, float64, ...observabilityx.Attribute) {
	t.backend.gaugeCount.Add(1)
}

type testSpan struct{}

func (testSpan) End() {}

func (testSpan) RecordError(error) {}

func (testSpan) SetAttributes(...observabilityx.Attribute) {}
