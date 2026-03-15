package otel

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	collectionmapping "github.com/DaiYuANg/archgo/collectionx/mapping"
	"github.com/DaiYuANg/archgo/observabilityx"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	defaultTracerName = "github.com/DaiYuANg/archgo"
	defaultMeterName  = "github.com/DaiYuANg/archgo"
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
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(&cfg)
	}

	return &adapter{
		logger:     observabilityx.NormalizeLogger(cfg.logger),
		tracer:     cfg.tracer,
		meter:      cfg.meter,
		counters:   collectionmapping.NewConcurrentMap[string, metric.Int64Counter](),
		histograms: collectionmapping.NewConcurrentMap[string, metric.Float64Histogram](),
	}
}

type adapter struct {
	logger *slog.Logger
	tracer trace.Tracer
	meter  metric.Meter

	counters   *collectionmapping.ConcurrentMap[string, metric.Int64Counter]
	histograms *collectionmapping.ConcurrentMap[string, metric.Float64Histogram]
}

func (a *adapter) Logger() *slog.Logger {
	return observabilityx.NormalizeLogger(a.logger)
}

func (a *adapter) StartSpan(
	ctx context.Context,
	name string,
	attrs ...observabilityx.Attribute,
) (context.Context, observabilityx.Span) {
	if ctx == nil {
		ctx = context.Background()
	}

	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		cleanName = "operation"
	}

	spanOptions := []trace.SpanStartOption{
		trace.WithAttributes(toOTelAttributes(attrs)...),
	}
	nextCtx, span := a.tracer.Start(ctx, cleanName, spanOptions...)
	return nextCtx, otelSpan{span: span}
}

func (a *adapter) AddCounter(
	ctx context.Context,
	name string,
	value int64,
	attrs ...observabilityx.Attribute,
) {
	if value == 0 {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	counter, err := a.counter(name)
	if err != nil {
		a.Logger().Warn("create metric counter failed", "name", name, "error", err.Error())
		return
	}
	counter.Add(ctx, value, metric.WithAttributes(toOTelAttributes(attrs)...))
}

func (a *adapter) RecordHistogram(
	ctx context.Context,
	name string,
	value float64,
	attrs ...observabilityx.Attribute,
) {
	if ctx == nil {
		ctx = context.Background()
	}

	histogram, err := a.histogram(name)
	if err != nil {
		a.Logger().Warn("create metric histogram failed", "name", name, "error", err.Error())
		return
	}
	histogram.Record(ctx, value, metric.WithAttributes(toOTelAttributes(attrs)...))
}

func (a *adapter) counter(name string) (metric.Int64Counter, error) {
	clean := strings.TrimSpace(name)
	if clean == "" {
		return nil, fmt.Errorf("metric counter name is empty")
	}

	if existing, ok := a.counters.Get(clean); ok {
		return existing, nil
	}

	created, err := a.meter.Int64Counter(clean)
	if err != nil {
		return nil, err
	}

	actual, _ := a.counters.GetOrStore(clean, created)
	return actual, nil
}

func (a *adapter) histogram(name string) (metric.Float64Histogram, error) {
	clean := strings.TrimSpace(name)
	if clean == "" {
		return nil, fmt.Errorf("metric histogram name is empty")
	}

	if existing, ok := a.histograms.Get(clean); ok {
		return existing, nil
	}

	created, err := a.meter.Float64Histogram(clean)
	if err != nil {
		return nil, err
	}

	actual, _ := a.histograms.GetOrStore(clean, created)
	return actual, nil
}

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

func toOTelAttributes(attrs []observabilityx.Attribute) []attribute.KeyValue {
	if len(attrs) == 0 {
		return nil
	}

	return lo.FilterMap(attrs, func(attr observabilityx.Attribute, _ int) (attribute.KeyValue, bool) {
		key := strings.TrimSpace(attr.Key)
		if key == "" {
			return attribute.KeyValue{}, false
		}
		return toOTelAttribute(key, attr.Value), true
	})
}

func toOTelAttribute(key string, value any) attribute.KeyValue {
	k := attribute.Key(key)

	switch typed := value.(type) {
	case string:
		return k.String(typed)
	case bool:
		return k.Bool(typed)
	case int:
		return k.Int(typed)
	case int8:
		return k.Int64(int64(typed))
	case int16:
		return k.Int64(int64(typed))
	case int32:
		return k.Int64(int64(typed))
	case int64:
		return k.Int64(typed)
	case uint:
		return k.Int64(int64(typed))
	case uint8:
		return k.Int64(int64(typed))
	case uint16:
		return k.Int64(int64(typed))
	case uint32:
		return k.Int64(int64(typed))
	case uint64:
		return k.Int64(int64(typed))
	case float32:
		return k.Float64(float64(typed))
	case float64:
		return k.Float64(typed)
	case time.Duration:
		return k.Int64(typed.Milliseconds())
	default:
		return k.String(fmt.Sprint(typed))
	}
}
