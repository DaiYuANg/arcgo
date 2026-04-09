package observabilityx

import (
	"context"
	"log/slog"
)

// Nop returns a no-op observability implementation.
func Nop() Observability {
	return NopWithLogger(nil)
}

// NopWithLogger returns a no-op observability implementation with logger support.
func NopWithLogger(logger *slog.Logger) Observability {
	return &nopObservability{
		logger: NormalizeLogger(logger),
	}
}

type nopObservability struct {
	logger *slog.Logger
}

func (n *nopObservability) Logger() *slog.Logger {
	return NormalizeLogger(n.logger)
}

func (n *nopObservability) StartSpan(ctx context.Context, name string, attrs ...Attribute) (context.Context, Span) {
	_ = name
	_ = attrs
	if ctx == nil {
		ctx = context.Background()
	}
	return ctx, nopSpan{}
}

func (n *nopObservability) Counter(spec CounterSpec) Counter {
	_ = spec
	return nopCounter{}
}

func (n *nopObservability) UpDownCounter(spec UpDownCounterSpec) UpDownCounter {
	_ = spec
	return nopUpDownCounter{}
}

func (n *nopObservability) Histogram(spec HistogramSpec) Histogram {
	_ = spec
	return nopHistogram{}
}

func (n *nopObservability) Gauge(spec GaugeSpec) Gauge {
	_ = spec
	return nopGauge{}
}

type nopCounter struct{}

func (nopCounter) Add(context.Context, int64, ...Attribute) {}

type nopUpDownCounter struct{}

func (nopUpDownCounter) Add(context.Context, int64, ...Attribute) {}

type nopHistogram struct{}

func (nopHistogram) Record(context.Context, float64, ...Attribute) {}

type nopGauge struct{}

func (nopGauge) Set(context.Context, float64, ...Attribute) {}

type nopSpan struct{}

func (nopSpan) End() {}

func (nopSpan) RecordError(err error) {
	_ = err
}

func (nopSpan) SetAttributes(attrs ...Attribute) {
	_ = attrs
}
