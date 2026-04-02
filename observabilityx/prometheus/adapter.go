package prometheus

import (
	"context"
	"log/slog"
	"net/http"

	collectionmapping "github.com/DaiYuANg/arcgo/collectionx/mapping"
	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/DaiYuANg/arcgo/pkg/option"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Adapter is a Prometheus-backed observability adapter.
type Adapter struct {
	logger    *slog.Logger
	namespace string
	register  prom.Registerer
	gatherer  prom.Gatherer
	buckets   []float64

	counters   *collectionmapping.ConcurrentMap[string, *counterInstrument]
	histograms *collectionmapping.ConcurrentMap[string, *histInstrument]
}

type counterInstrument struct {
	labels []string
	vec    *prom.CounterVec
}

type histInstrument struct {
	labels []string
	vec    *prom.HistogramVec
}

// New creates a Prometheus adapter.
func New(opts ...Option) *Adapter {
	cfg := defaultConfig()
	option.Apply(&cfg, opts...)

	return &Adapter{
		logger:     observabilityx.NormalizeLogger(cfg.logger),
		namespace:  normalizeMetricSegment(cfg.namespace, defaultNamespace),
		register:   cfg.register,
		gatherer:   cfg.gatherer,
		buckets:    cfg.buckets.Values(),
		counters:   collectionmapping.NewConcurrentMap[string, *counterInstrument](),
		histograms: collectionmapping.NewConcurrentMap[string, *histInstrument](),
	}
}

// Logger returns logger for this adapter.
func (a *Adapter) Logger() *slog.Logger {
	return observabilityx.NormalizeLogger(a.logger)
}

// StartSpan is a no-op for Prometheus adapter.
func (a *Adapter) StartSpan(
	ctx context.Context,
	name string,
	attrs ...observabilityx.Attribute,
) (context.Context, observabilityx.Span) {
	_ = name
	_ = attrs
	if ctx == nil {
		ctx = context.Background()
	}
	return ctx, noopSpan{}
}

// AddCounter records an increment for a named counter.
func (a *Adapter) AddCounter(
	ctx context.Context,
	name string,
	value int64,
	attrs ...observabilityx.Attribute,
) {
	_ = ctx
	if value == 0 {
		return
	}

	metricName := a.normalizeMetricName(name)
	metricLabels := attrsToLabelMap(attrs)
	instrument, err := a.counter(metricName, metricLabels)
	if err != nil {
		a.Logger().Warn("prometheus counter setup failed", "metric", metricName, "error", err.Error())
		return
	}

	labelValues := toPromLabels(instrument.labels, metricLabels)
	counter, err := instrument.vec.GetMetricWith(labelValues)
	if err != nil {
		a.Logger().Warn("prometheus counter labels mismatch", "metric", metricName, "error", err.Error())
		return
	}
	counter.Add(float64(value))
}

// RecordHistogram records a value for a named histogram.
func (a *Adapter) RecordHistogram(
	ctx context.Context,
	name string,
	value float64,
	attrs ...observabilityx.Attribute,
) {
	_ = ctx
	metricName := a.normalizeMetricName(name)
	metricLabels := attrsToLabelMap(attrs)
	instrument, err := a.histogram(metricName, metricLabels)
	if err != nil {
		a.Logger().Warn("prometheus histogram setup failed", "metric", metricName, "error", err.Error())
		return
	}

	labelValues := toPromLabels(instrument.labels, metricLabels)
	histogram, err := instrument.vec.GetMetricWith(labelValues)
	if err != nil {
		a.Logger().Warn("prometheus histogram labels mismatch", "metric", metricName, "error", err.Error())
		return
	}
	histogram.Observe(value)
}

// Handler returns HTTP metrics handler for the configured gatherer.
func (a *Adapter) Handler() http.Handler {
	return promhttp.HandlerFor(a.gatherer, promhttp.HandlerOpts{})
}
