package observabilityx

import (
	"context"
	"log/slog"

	"github.com/DaiYuANg/arcgo/collectionx"
)

// Multi combines multiple observability backends into one.
//
// Use this to send telemetry to more than one backend (for example OTel + Prometheus).
func Multi(backends ...Observability) Observability {
	filtered := collectionx.NewList(backends...).Reject(func(_ int, backend Observability) bool {
		return backend == nil
	})
	if filtered.IsEmpty() {
		return Nop()
	}

	firstBackend, _ := filtered.GetFirst()
	logger := firstBackend.Logger()
	if logger == nil {
		logger = slog.Default()
	}

	return &multiObservability{
		backends: filtered,
		logger:   logger,
	}
}

type multiObservability struct {
	backends collectionx.List[Observability]
	logger   *slog.Logger
}

func (m *multiObservability) Logger() *slog.Logger {
	return NormalizeLogger(m.logger)
}

func (m *multiObservability) StartSpan(
	ctx context.Context,
	name string,
	attrs ...Attribute,
) (context.Context, Span) {
	if ctx == nil {
		ctx = context.Background()
	}

	firstBackend, _ := m.backends.GetFirst()
	nextCtx, firstSpan := firstBackend.StartSpan(ctx, name, attrs...)
	spans := collectionx.NewListWithCapacity[Span](m.backends.Len())
	if firstSpan != nil {
		spans.Add(firstSpan)
	}
	m.backends.Drop(1).Range(func(_ int, backend Observability) bool {
		_, span := backend.StartSpan(nextCtx, name, attrs...)
		if span != nil {
			spans.Add(span)
		}
		return true
	})
	if spans.Len() == 0 {
		return nextCtx, nopSpan{}
	}
	return nextCtx, multiSpan{spans: spans}
}

func (m *multiObservability) Counter(spec CounterSpec) Counter {
	counters := collectionx.NewListWithCapacity[Counter](m.backends.Len())
	m.backends.Each(func(_ int, backend Observability) {
		counter := backend.Counter(spec)
		if counter != nil {
			counters.Add(counter)
		}
	})
	if counters.IsEmpty() {
		return nopCounter{}
	}
	return multiCounter{counters: counters}
}

func (m *multiObservability) UpDownCounter(spec UpDownCounterSpec) UpDownCounter {
	counters := collectionx.NewListWithCapacity[UpDownCounter](m.backends.Len())
	m.backends.Each(func(_ int, backend Observability) {
		counter := backend.UpDownCounter(spec)
		if counter != nil {
			counters.Add(counter)
		}
	})
	if counters.IsEmpty() {
		return nopUpDownCounter{}
	}
	return multiUpDownCounter{counters: counters}
}

func (m *multiObservability) Histogram(spec HistogramSpec) Histogram {
	histograms := collectionx.NewListWithCapacity[Histogram](m.backends.Len())
	m.backends.Each(func(_ int, backend Observability) {
		histogram := backend.Histogram(spec)
		if histogram != nil {
			histograms.Add(histogram)
		}
	})
	if histograms.IsEmpty() {
		return nopHistogram{}
	}
	return multiHistogram{histograms: histograms}
}

func (m *multiObservability) Gauge(spec GaugeSpec) Gauge {
	gauges := collectionx.NewListWithCapacity[Gauge](m.backends.Len())
	m.backends.Each(func(_ int, backend Observability) {
		gauge := backend.Gauge(spec)
		if gauge != nil {
			gauges.Add(gauge)
		}
	})
	if gauges.IsEmpty() {
		return nopGauge{}
	}
	return multiGauge{gauges: gauges}
}

type multiCounter struct {
	counters collectionx.List[Counter]
}

func (m multiCounter) Add(ctx context.Context, value int64, attrs ...Attribute) {
	m.counters.Each(func(_ int, counter Counter) {
		counter.Add(ctx, value, attrs...)
	})
}

type multiUpDownCounter struct {
	counters collectionx.List[UpDownCounter]
}

func (m multiUpDownCounter) Add(ctx context.Context, value int64, attrs ...Attribute) {
	m.counters.Each(func(_ int, counter UpDownCounter) {
		counter.Add(ctx, value, attrs...)
	})
}

type multiHistogram struct {
	histograms collectionx.List[Histogram]
}

func (m multiHistogram) Record(ctx context.Context, value float64, attrs ...Attribute) {
	m.histograms.Each(func(_ int, histogram Histogram) {
		histogram.Record(ctx, value, attrs...)
	})
}

type multiGauge struct {
	gauges collectionx.List[Gauge]
}

func (m multiGauge) Set(ctx context.Context, value float64, attrs ...Attribute) {
	m.gauges.Each(func(_ int, gauge Gauge) {
		gauge.Set(ctx, value, attrs...)
	})
}

type multiSpan struct {
	spans collectionx.List[Span]
}

func (s multiSpan) End() {
	s.spans.Each(func(_ int, span Span) {
		span.End()
	})
}

func (s multiSpan) RecordError(err error) {
	if err == nil {
		return
	}
	s.spans.Each(func(_ int, span Span) {
		span.RecordError(err)
	})
}

func (s multiSpan) SetAttributes(attrs ...Attribute) {
	s.spans.Each(func(_ int, span Span) {
		span.SetAttributes(attrs...)
	})
}
