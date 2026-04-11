//revive:disable:file-length-limit OTel adapter methods are kept together as one cohesive integration surface.

package otel

import (
	"context"
	"log/slog"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	collectionmapping "github.com/DaiYuANg/arcgo/collectionx/mapping"
	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/DaiYuANg/arcgo/pkg/option"
	"github.com/samber/oops"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	defaultTracerName = "github.com/DaiYuANg/arcgo"
	defaultMeterName  = "github.com/DaiYuANg/arcgo"
)

// Option configures OTel observability integration.
type Option func(*config)

type config struct {
	logger *slog.Logger
	tracer trace.Tracer
	meter  metric.Meter
}

// WithLogger sets logger used by this adapter.
func WithLogger(logger *slog.Logger) Option {
	return func(cfg *config) {
		cfg.logger = logger
	}
}

// WithTracer sets tracer used by this adapter.
func WithTracer(tracer trace.Tracer) Option {
	return func(cfg *config) {
		cfg.tracer = tracer
	}
}

// WithMeter sets meter used by this adapter.
func WithMeter(meter metric.Meter) Option {
	return func(cfg *config) {
		cfg.meter = meter
	}
}

// New creates an OTel-backed observability adapter.
func New(opts ...Option) observabilityx.Observability {
	cfg := config{
		logger: slog.Default(),
		tracer: otel.Tracer(defaultTracerName),
		meter:  otel.Meter(defaultMeterName),
	}
	option.Apply(&cfg, opts...)

	return &adapter{
		logger:         observabilityx.NormalizeLogger(cfg.logger),
		tracer:         cfg.tracer,
		meter:          cfg.meter,
		counters:       collectionmapping.NewConcurrentMap[string, metric.Int64Counter](),
		upDownCounters: collectionmapping.NewConcurrentMap[string, metric.Int64UpDownCounter](),
		histograms:     collectionmapping.NewConcurrentMap[string, metric.Float64Histogram](),
		gauges:         collectionmapping.NewConcurrentMap[string, metric.Float64Gauge](),
	}
}

type adapter struct {
	logger *slog.Logger
	tracer trace.Tracer
	meter  metric.Meter

	counters       *collectionmapping.ConcurrentMap[string, metric.Int64Counter]
	upDownCounters *collectionmapping.ConcurrentMap[string, metric.Int64UpDownCounter]
	histograms     *collectionmapping.ConcurrentMap[string, metric.Float64Histogram]
	gauges         *collectionmapping.ConcurrentMap[string, metric.Float64Gauge]
}

func (a *adapter) Logger() *slog.Logger {
	return observabilityx.NormalizeLogger(a.logger)
}

func (a *adapter) StartSpan(
	ctx context.Context,
	name string,
	attrs ...observabilityx.Attribute,
) (context.Context, observabilityx.Span) {
	return startTraceSpan(normalizeContext(ctx), a.tracer, normalizeSpanName(name), attrs)
}

func (a *adapter) Counter(spec observabilityx.CounterSpec) observabilityx.Counter {
	spec = observabilityx.NormalizeCounterSpec(spec)
	counter, err := a.counter(spec)
	if err != nil {
		a.Logger().Warn("create metric counter failed", "name", spec.Name, "error", err.Error())
		return nopCounter{}
	}
	return otelCounter{counter: counter, labelKeys: spec.LabelKeys}
}

func (a *adapter) UpDownCounter(spec observabilityx.UpDownCounterSpec) observabilityx.UpDownCounter {
	spec = observabilityx.NormalizeUpDownCounterSpec(spec)
	counter, err := a.upDownCounter(spec)
	if err != nil {
		a.Logger().Warn("create metric up-down counter failed", "name", spec.Name, "error", err.Error())
		return nopUpDownCounter{}
	}
	return otelUpDownCounter{counter: counter, labelKeys: spec.LabelKeys}
}

func (a *adapter) Histogram(spec observabilityx.HistogramSpec) observabilityx.Histogram {
	spec = observabilityx.NormalizeHistogramSpec(spec)
	histogram, err := a.histogram(spec)
	if err != nil {
		a.Logger().Warn("create metric histogram failed", "name", spec.Name, "error", err.Error())
		return nopHistogram{}
	}
	return otelHistogram{histogram: histogram, labelKeys: spec.LabelKeys}
}

func (a *adapter) Gauge(spec observabilityx.GaugeSpec) observabilityx.Gauge {
	spec = observabilityx.NormalizeGaugeSpec(spec)
	gauge, err := a.gauge(spec)
	if err != nil {
		a.Logger().Warn("create metric gauge failed", "name", spec.Name, "error", err.Error())
		return nopGauge{}
	}
	return otelGauge{gauge: gauge, labelKeys: spec.LabelKeys}
}

func (a *adapter) counter(spec observabilityx.CounterSpec) (metric.Int64Counter, error) {
	key := observabilityx.NormalizeCounterSpec(spec)
	return metricInstrument(
		a,
		"create_counter",
		"counter",
		"metric counter name is empty",
		key.MetricSpec,
		cacheMetricSpecKey("counter", key.MetricSpec),
		func(adapter *adapter) *collectionmapping.ConcurrentMap[string, metric.Int64Counter] {
			return adapter.counters
		},
		func(adapter *adapter, clean string) (metric.Int64Counter, error) {
			return adapter.meter.Int64Counter(clean, counterOptions(key)...)
		},
	)
}

func (a *adapter) upDownCounter(spec observabilityx.UpDownCounterSpec) (metric.Int64UpDownCounter, error) {
	key := observabilityx.NormalizeUpDownCounterSpec(spec)
	return metricInstrument(
		a,
		"create_up_down_counter",
		"up-down counter",
		"metric up-down counter name is empty",
		key.MetricSpec,
		cacheMetricSpecKey("up_down_counter", key.MetricSpec),
		func(adapter *adapter) *collectionmapping.ConcurrentMap[string, metric.Int64UpDownCounter] {
			return adapter.upDownCounters
		},
		func(adapter *adapter, clean string) (metric.Int64UpDownCounter, error) {
			return adapter.meter.Int64UpDownCounter(clean, upDownCounterOptions(key)...)
		},
	)
}

func (a *adapter) histogram(spec observabilityx.HistogramSpec) (metric.Float64Histogram, error) {
	key := observabilityx.NormalizeHistogramSpec(spec)
	return metricInstrument(
		a,
		"create_histogram",
		"histogram",
		"metric histogram name is empty",
		key.MetricSpec,
		cacheHistogramSpecKey(key),
		func(adapter *adapter) *collectionmapping.ConcurrentMap[string, metric.Float64Histogram] {
			return adapter.histograms
		},
		func(adapter *adapter, clean string) (metric.Float64Histogram, error) {
			return adapter.meter.Float64Histogram(clean, histogramOptions(key)...)
		},
	)
}

func (a *adapter) gauge(spec observabilityx.GaugeSpec) (metric.Float64Gauge, error) {
	key := observabilityx.NormalizeGaugeSpec(spec)
	return metricInstrument(
		a,
		"create_gauge",
		"gauge",
		"metric gauge name is empty",
		key.MetricSpec,
		cacheMetricSpecKey("gauge", key.MetricSpec),
		func(adapter *adapter) *collectionmapping.ConcurrentMap[string, metric.Float64Gauge] {
			return adapter.gauges
		},
		func(adapter *adapter, clean string) (metric.Float64Gauge, error) {
			return adapter.meter.Float64Gauge(clean, gaugeOptions(key)...)
		},
	)
}

func metricInstrument[I any](
	a *adapter,
	op, kind, emptyNameMessage string,
	spec observabilityx.MetricSpec,
	cacheKey string,
	cacheFor func(*adapter) *collectionmapping.ConcurrentMap[string, I],
	create func(*adapter, string) (I, error),
) (I, error) {
	var zero I
	if a == nil {
		return zero, oops.In("observabilityx/otel").
			With("op", op).
			New("adapter is nil")
	}
	clean := strings.TrimSpace(spec.Name)
	if clean == "" {
		return zero, oops.In("observabilityx/otel").
			With("op", op).
			New(emptyNameMessage)
	}
	if a.meter == nil {
		return zero, oops.In("observabilityx/otel").
			With("op", op, "metric", clean).
			New("meter is nil")
	}
	cache := cacheFor(a)
	if existing, ok := cache.Get(cacheKey); ok {
		return existing, nil
	}

	created, err := create(a, clean)
	if err != nil {
		return zero, oops.In("observabilityx/otel").
			With("op", op, "metric", clean).
			Wrapf(err, "create OTel %s", kind)
	}

	actual, _ := cache.GetOrStore(cacheKey, created)
	return actual, nil
}

type otelCounter struct {
	counter   metric.Int64Counter
	labelKeys collectionx.List[string]
}

func (c otelCounter) Add(ctx context.Context, value int64, attrs ...observabilityx.Attribute) {
	if value <= 0 {
		return
	}
	c.counter.Add(normalizeContext(ctx), value, metric.WithAttributes(toOTelAttributes(observabilityx.FilterMetricAttributes(c.labelKeys, attrs...))...))
}

type otelUpDownCounter struct {
	counter   metric.Int64UpDownCounter
	labelKeys collectionx.List[string]
}

func (c otelUpDownCounter) Add(ctx context.Context, value int64, attrs ...observabilityx.Attribute) {
	if value == 0 {
		return
	}
	c.counter.Add(normalizeContext(ctx), value, metric.WithAttributes(toOTelAttributes(observabilityx.FilterMetricAttributes(c.labelKeys, attrs...))...))
}

type otelHistogram struct {
	histogram metric.Float64Histogram
	labelKeys collectionx.List[string]
}

func (h otelHistogram) Record(ctx context.Context, value float64, attrs ...observabilityx.Attribute) {
	h.histogram.Record(normalizeContext(ctx), value, metric.WithAttributes(toOTelAttributes(observabilityx.FilterMetricAttributes(h.labelKeys, attrs...))...))
}

type otelGauge struct {
	gauge     metric.Float64Gauge
	labelKeys collectionx.List[string]
}

func (g otelGauge) Set(ctx context.Context, value float64, attrs ...observabilityx.Attribute) {
	g.gauge.Record(normalizeContext(ctx), value, metric.WithAttributes(toOTelAttributes(observabilityx.FilterMetricAttributes(g.labelKeys, attrs...))...))
}

type nopCounter struct{}

func (nopCounter) Add(context.Context, int64, ...observabilityx.Attribute) {}

type nopUpDownCounter struct{}

func (nopUpDownCounter) Add(context.Context, int64, ...observabilityx.Attribute) {}

type nopHistogram struct{}

func (nopHistogram) Record(context.Context, float64, ...observabilityx.Attribute) {}

type nopGauge struct{}

func (nopGauge) Set(context.Context, float64, ...observabilityx.Attribute) {}

type otelSpan struct {
	span trace.Span
}

func (s otelSpan) End() {
	if s.span != nil {
		s.span.End()
	}
}

func (s otelSpan) RecordError(err error) {
	if s.span != nil && err != nil {
		s.span.RecordError(err)
	}
}

func (s otelSpan) SetAttributes(attrs ...observabilityx.Attribute) {
	if s.span == nil || len(attrs) == 0 {
		return
	}
	s.span.SetAttributes(toOTelAttributes(attrs)...)
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}

	return context.Background()
}

func normalizeSpanName(name string) string {
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return "operation"
	}

	return cleanName
}

//nolint:spancheck // span ownership is transferred to the returned observabilityx.Span wrapper.
func startTraceSpan(ctx context.Context, tracer trace.Tracer, name string, attrs []observabilityx.Attribute) (context.Context, observabilityx.Span) {
	nextCtx, span := tracer.Start(ctx, name, trace.WithAttributes(toOTelAttributes(attrs)...))
	return nextCtx, otelSpan{span: span}
}
