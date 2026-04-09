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

	counters       *collectionmapping.ConcurrentMap[string, *counterInstrument]
	upDownCounters *collectionmapping.ConcurrentMap[string, *gaugeInstrument]
	histograms     *collectionmapping.ConcurrentMap[string, *histInstrument]
	gauges         *collectionmapping.ConcurrentMap[string, *gaugeInstrument]
}

type counterInstrument struct {
	spec   observabilityx.CounterSpec
	labels []string
	vec    *prom.CounterVec
}

type gaugeInstrument struct {
	spec   observabilityx.MetricSpec
	labels []string
	vec    *prom.GaugeVec
}

type histInstrument struct {
	spec   observabilityx.HistogramSpec
	labels []string
	vec    *prom.HistogramVec
}

// New creates a Prometheus adapter.
func New(opts ...Option) *Adapter {
	cfg := defaultConfig()
	option.Apply(&cfg, opts...)

	return &Adapter{
		logger:         observabilityx.NormalizeLogger(cfg.logger),
		namespace:      normalizeMetricSegment(cfg.namespace, defaultNamespace),
		register:       cfg.register,
		gatherer:       cfg.gatherer,
		buckets:        cfg.buckets.Values(),
		counters:       collectionmapping.NewConcurrentMap[string, *counterInstrument](),
		upDownCounters: collectionmapping.NewConcurrentMap[string, *gaugeInstrument](),
		histograms:     collectionmapping.NewConcurrentMap[string, *histInstrument](),
		gauges:         collectionmapping.NewConcurrentMap[string, *gaugeInstrument](),
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

// Counter returns a declared Prometheus counter handle.
func (a *Adapter) Counter(spec observabilityx.CounterSpec) observabilityx.Counter {
	instrument, err := a.counter(spec)
	if err != nil {
		a.Logger().Warn("prometheus counter setup failed", "metric", spec.Name, "error", err.Error())
		return nopCounter{}
	}
	return promCounter{logger: a.Logger(), instrument: instrument}
}

// UpDownCounter returns a declared Prometheus up-down counter handle.
func (a *Adapter) UpDownCounter(spec observabilityx.UpDownCounterSpec) observabilityx.UpDownCounter {
	instrument, err := a.upDownCounter(spec)
	if err != nil {
		a.Logger().Warn("prometheus up-down counter setup failed", "metric", spec.Name, "error", err.Error())
		return nopUpDownCounter{}
	}
	return promUpDownCounter{logger: a.Logger(), instrument: instrument}
}

// Histogram returns a declared Prometheus histogram handle.
func (a *Adapter) Histogram(spec observabilityx.HistogramSpec) observabilityx.Histogram {
	instrument, err := a.histogram(spec)
	if err != nil {
		a.Logger().Warn("prometheus histogram setup failed", "metric", spec.Name, "error", err.Error())
		return nopHistogram{}
	}
	return promHistogram{logger: a.Logger(), instrument: instrument}
}

// Gauge returns a declared Prometheus gauge handle.
func (a *Adapter) Gauge(spec observabilityx.GaugeSpec) observabilityx.Gauge {
	instrument, err := a.gauge(spec)
	if err != nil {
		a.Logger().Warn("prometheus gauge setup failed", "metric", spec.Name, "error", err.Error())
		return nopGauge{}
	}
	return promGauge{logger: a.Logger(), instrument: instrument}
}

// Handler returns HTTP metrics handler for the configured gatherer.
func (a *Adapter) Handler() http.Handler {
	return promhttp.HandlerFor(a.gatherer, promhttp.HandlerOpts{})
}

type promCounter struct {
	logger     *slog.Logger
	instrument *counterInstrument
}

func (c promCounter) Add(ctx context.Context, value int64, attrs ...observabilityx.Attribute) {
	_ = ctx
	if value <= 0 {
		if value < 0 {
			observabilityx.NormalizeLogger(c.logger).Warn("prometheus counter rejected negative value", "metric", c.instrument.spec.Name, "value", value)
		}
		return
	}

	labelValues := observabilityx.MetricLabelMap(c.instrument.spec.LabelKeys, attrs...)
	counter, err := c.instrument.vec.GetMetricWith(toPromLabels(c.instrument.labels, labelValues))
	if err != nil {
		observabilityx.NormalizeLogger(c.logger).Warn("prometheus counter labels mismatch", "metric", c.instrument.spec.Name, "error", err.Error())
		return
	}
	counter.Add(float64(value))
}

type promUpDownCounter struct {
	logger     *slog.Logger
	instrument *gaugeInstrument
}

func (c promUpDownCounter) Add(ctx context.Context, value int64, attrs ...observabilityx.Attribute) {
	_ = ctx
	if value == 0 {
		return
	}

	labelValues := observabilityx.MetricLabelMap(c.instrument.spec.LabelKeys, attrs...)
	gauge, err := c.instrument.vec.GetMetricWith(toPromLabels(c.instrument.labels, labelValues))
	if err != nil {
		observabilityx.NormalizeLogger(c.logger).Warn("prometheus up-down counter labels mismatch", "metric", c.instrument.spec.Name, "error", err.Error())
		return
	}
	gauge.Add(float64(value))
}

type promHistogram struct {
	logger     *slog.Logger
	instrument *histInstrument
}

func (h promHistogram) Record(ctx context.Context, value float64, attrs ...observabilityx.Attribute) {
	_ = ctx
	labelValues := observabilityx.MetricLabelMap(h.instrument.spec.LabelKeys, attrs...)
	histogram, err := h.instrument.vec.GetMetricWith(toPromLabels(h.instrument.labels, labelValues))
	if err != nil {
		observabilityx.NormalizeLogger(h.logger).Warn("prometheus histogram labels mismatch", "metric", h.instrument.spec.Name, "error", err.Error())
		return
	}
	histogram.Observe(value)
}

type promGauge struct {
	logger     *slog.Logger
	instrument *gaugeInstrument
}

func (g promGauge) Set(ctx context.Context, value float64, attrs ...observabilityx.Attribute) {
	_ = ctx
	labelValues := observabilityx.MetricLabelMap(g.instrument.spec.LabelKeys, attrs...)
	gauge, err := g.instrument.vec.GetMetricWith(toPromLabels(g.instrument.labels, labelValues))
	if err != nil {
		observabilityx.NormalizeLogger(g.logger).Warn("prometheus gauge labels mismatch", "metric", g.instrument.spec.Name, "error", err.Error())
		return
	}
	gauge.Set(value)
}

type nopCounter struct{}

func (nopCounter) Add(context.Context, int64, ...observabilityx.Attribute) {}

type nopUpDownCounter struct{}

func (nopUpDownCounter) Add(context.Context, int64, ...observabilityx.Attribute) {}

type nopHistogram struct{}

func (nopHistogram) Record(context.Context, float64, ...observabilityx.Attribute) {}

type nopGauge struct{}

func (nopGauge) Set(context.Context, float64, ...observabilityx.Attribute) {}
