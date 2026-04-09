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
	if a == nil {
		return nil, oops.In("observabilityx/otel").
			With("op", "create_counter").
			New("adapter is nil")
	}
	clean := strings.TrimSpace(spec.Name)
	if clean == "" {
		return nil, oops.In("observabilityx/otel").
			With("op", "create_counter").
			New("metric counter name is empty")
	}
	if a.meter == nil {
		return nil, oops.In("observabilityx/otel").
			With("op", "create_counter", "metric", clean).
			New("meter is nil")
	}

	key := observabilityx.NormalizeCounterSpec(spec)
	cacheKey := cacheMetricSpecKey("counter", key.MetricSpec)
	if existing, ok := a.counters.Get(cacheKey); ok {
		return existing, nil
	}

	created, err := a.meter.Int64Counter(clean, counterOptions(spec)...)
	if err != nil {
		return nil, oops.In("observabilityx/otel").
			With("op", "create_counter", "metric", clean).
			Wrapf(err, "create OTel counter")
	}

	actual, _ := a.counters.GetOrStore(cacheKey, created)
	return actual, nil
}

func (a *adapter) upDownCounter(spec observabilityx.UpDownCounterSpec) (metric.Int64UpDownCounter, error) {
	if a == nil {
		return nil, oops.In("observabilityx/otel").
			With("op", "create_up_down_counter").
			New("adapter is nil")
	}
	clean := strings.TrimSpace(spec.Name)
	if clean == "" {
		return nil, oops.In("observabilityx/otel").
			With("op", "create_up_down_counter").
			New("metric up-down counter name is empty")
	}
	if a.meter == nil {
		return nil, oops.In("observabilityx/otel").
			With("op", "create_up_down_counter", "metric", clean).
			New("meter is nil")
	}

	key := observabilityx.NormalizeUpDownCounterSpec(spec)
	cacheKey := cacheMetricSpecKey("up_down_counter", key.MetricSpec)
	if existing, ok := a.upDownCounters.Get(cacheKey); ok {
		return existing, nil
	}

	created, err := a.meter.Int64UpDownCounter(clean, upDownCounterOptions(spec)...)
	if err != nil {
		return nil, oops.In("observabilityx/otel").
			With("op", "create_up_down_counter", "metric", clean).
			Wrapf(err, "create OTel up-down counter")
	}

	actual, _ := a.upDownCounters.GetOrStore(cacheKey, created)
	return actual, nil
}

func (a *adapter) histogram(spec observabilityx.HistogramSpec) (metric.Float64Histogram, error) {
	if a == nil {
		return nil, oops.In("observabilityx/otel").
			With("op", "create_histogram").
			New("adapter is nil")
	}
	clean := strings.TrimSpace(spec.Name)
	if clean == "" {
		return nil, oops.In("observabilityx/otel").
			With("op", "create_histogram").
			New("metric histogram name is empty")
	}
	if a.meter == nil {
		return nil, oops.In("observabilityx/otel").
			With("op", "create_histogram", "metric", clean).
			New("meter is nil")
	}

	key := observabilityx.NormalizeHistogramSpec(spec)
	cacheKey := cacheHistogramSpecKey(key)
	if existing, ok := a.histograms.Get(cacheKey); ok {
		return existing, nil
	}

	created, err := a.meter.Float64Histogram(clean, histogramOptions(spec)...)
	if err != nil {
		return nil, oops.In("observabilityx/otel").
			With("op", "create_histogram", "metric", clean).
			Wrapf(err, "create OTel histogram")
	}

	actual, _ := a.histograms.GetOrStore(cacheKey, created)
	return actual, nil
}

func (a *adapter) gauge(spec observabilityx.GaugeSpec) (metric.Float64Gauge, error) {
	if a == nil {
		return nil, oops.In("observabilityx/otel").
			With("op", "create_gauge").
			New("adapter is nil")
	}
	clean := strings.TrimSpace(spec.Name)
	if clean == "" {
		return nil, oops.In("observabilityx/otel").
			With("op", "create_gauge").
			New("metric gauge name is empty")
	}
	if a.meter == nil {
		return nil, oops.In("observabilityx/otel").
			With("op", "create_gauge", "metric", clean).
			New("meter is nil")
	}

	key := observabilityx.NormalizeGaugeSpec(spec)
	cacheKey := cacheMetricSpecKey("gauge", key.MetricSpec)
	if existing, ok := a.gauges.Get(cacheKey); ok {
		return existing, nil
	}

	created, err := a.meter.Float64Gauge(clean, gaugeOptions(spec)...)
	if err != nil {
		return nil, oops.In("observabilityx/otel").
			With("op", "create_gauge", "metric", clean).
			Wrapf(err, "create OTel gauge")
	}

	actual, _ := a.gauges.GetOrStore(cacheKey, created)
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
