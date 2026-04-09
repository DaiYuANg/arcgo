package prometheus

import (
	"errors"
	"fmt"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/observabilityx"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/samber/oops"
)

func (a *Adapter) counter(spec observabilityx.CounterSpec) (*counterInstrument, error) {
	if a == nil {
		return nil, oops.In("observabilityx/prometheus").
			With("op", "create_counter", "metric", spec.Name).
			New("adapter is nil")
	}

	spec = observabilityx.NormalizeCounterSpec(spec)
	spec.Name = a.normalizeMetricName(spec.Name)
	cacheKey := observabilityx.NormalizeCounterSpec(spec)
	key := cacheKey.MetricSpec.Name + "|" + metricSpecKey(cacheKey.MetricSpec)
	if existing, ok := a.counters.Get(key); ok {
		return existing, nil
	}

	labelNames := sortedLabelKeys(spec.MetricSpec)
	vec := prom.NewCounterVec(prom.CounterOpts{
		Name: spec.Name,
		Help: helpText(spec.Description, spec.Name),
	}, labelNames)

	if err := a.register.Register(vec); err != nil {
		alreadyRegisteredError, ok := errors.AsType[*prom.AlreadyRegisteredError](err)
		if !ok || alreadyRegisteredError == nil {
			return nil, wrapRegistrationError("create_counter", spec.Name, len(labelNames), err)
		}
		existingVec, ok := alreadyRegisteredError.ExistingCollector.(*prom.CounterVec)
		if !ok {
			return nil, unexpectedCollectorType("create_counter", spec.Name, len(labelNames), alreadyRegisteredError.ExistingCollector)
		}
		vec = existingVec
	}

	inst := &counterInstrument{
		spec:   spec,
		labels: labelNames,
		vec:    vec,
	}
	actual, _ := a.counters.GetOrStore(key, inst)
	return actual, nil
}

func (a *Adapter) upDownCounter(spec observabilityx.UpDownCounterSpec) (*gaugeInstrument, error) {
	if a == nil {
		return nil, oops.In("observabilityx/prometheus").
			With("op", "create_up_down_counter", "metric", spec.Name).
			New("adapter is nil")
	}

	normalized := observabilityx.NormalizeUpDownCounterSpec(spec)
	normalized.Name = a.normalizeMetricName(normalized.Name)
	key := normalized.Name + "|" + metricSpecKey(normalized.MetricSpec)
	if existing, ok := a.upDownCounters.Get(key); ok {
		return existing, nil
	}

	inst, err := a.registerGaugeInstrument("create_up_down_counter", normalized.MetricSpec)
	if err != nil {
		return nil, err
	}
	actual, _ := a.upDownCounters.GetOrStore(key, inst)
	return actual, nil
}

func (a *Adapter) histogram(spec observabilityx.HistogramSpec) (*histInstrument, error) {
	if a == nil {
		return nil, oops.In("observabilityx/prometheus").
			With("op", "create_histogram", "metric", spec.Name).
			New("adapter is nil")
	}

	spec = observabilityx.NormalizeHistogramSpec(spec)
	spec.Name = a.normalizeMetricName(spec.Name)
	key := spec.Name + "|" + histogramSpecKey(spec)
	if existing, ok := a.histograms.Get(key); ok {
		return existing, nil
	}

	labelNames := sortedLabelKeys(spec.MetricSpec)
	vec := prom.NewHistogramVec(prom.HistogramOpts{
		Name:    spec.Name,
		Help:    helpText(spec.Description, spec.Name),
		Buckets: histogramBuckets(a.buckets, spec.Buckets),
	}, labelNames)

	if err := a.register.Register(vec); err != nil {
		alreadyRegisteredError, ok := errors.AsType[*prom.AlreadyRegisteredError](err)
		if !ok || alreadyRegisteredError == nil {
			return nil, wrapRegistrationError("create_histogram", spec.Name, len(labelNames), err)
		}
		existingVec, ok := alreadyRegisteredError.ExistingCollector.(*prom.HistogramVec)
		if !ok {
			return nil, unexpectedCollectorType("create_histogram", spec.Name, len(labelNames), alreadyRegisteredError.ExistingCollector)
		}
		vec = existingVec
	}

	inst := &histInstrument{
		spec:   spec,
		labels: labelNames,
		vec:    vec,
	}
	actual, _ := a.histograms.GetOrStore(key, inst)
	return actual, nil
}

func (a *Adapter) gauge(spec observabilityx.GaugeSpec) (*gaugeInstrument, error) {
	if a == nil {
		return nil, oops.In("observabilityx/prometheus").
			With("op", "create_gauge", "metric", spec.Name).
			New("adapter is nil")
	}

	normalized := observabilityx.NormalizeGaugeSpec(spec)
	normalized.Name = a.normalizeMetricName(normalized.Name)
	key := normalized.Name + "|" + metricSpecKey(normalized.MetricSpec)
	if existing, ok := a.gauges.Get(key); ok {
		return existing, nil
	}

	inst, err := a.registerGaugeInstrument("create_gauge", normalized.MetricSpec)
	if err != nil {
		return nil, err
	}
	actual, _ := a.gauges.GetOrStore(key, inst)
	return actual, nil
}

func (a *Adapter) registerGaugeInstrument(op string, spec observabilityx.MetricSpec) (*gaugeInstrument, error) {
	labelNames := sortedLabelKeys(spec)
	vec := prom.NewGaugeVec(prom.GaugeOpts{
		Name: spec.Name,
		Help: helpText(spec.Description, spec.Name),
	}, labelNames)

	if err := a.register.Register(vec); err != nil {
		alreadyRegisteredError, ok := errors.AsType[*prom.AlreadyRegisteredError](err)
		if !ok || alreadyRegisteredError == nil {
			return nil, wrapRegistrationError(op, spec.Name, len(labelNames), err)
		}
		existingVec, ok := alreadyRegisteredError.ExistingCollector.(*prom.GaugeVec)
		if !ok {
			return nil, unexpectedCollectorType(op, spec.Name, len(labelNames), alreadyRegisteredError.ExistingCollector)
		}
		vec = existingVec
	}

	return &gaugeInstrument{
		spec:   spec,
		labels: labelNames,
		vec:    vec,
	}, nil
}

func helpText(description string, metricName string) string {
	if description != "" {
		return description
	}
	return "Metric for " + metricName
}

func histogramBuckets(defaultBuckets []float64, specBuckets collectionx.List[float64]) []float64 {
	if specBuckets == nil || specBuckets.IsEmpty() {
		return defaultBuckets
	}
	return specBuckets.Values()
}

func wrapRegistrationError(op string, metricName string, labelCount int, err error) error {
	return oops.In("observabilityx/prometheus").
		With("op", op, "metric", metricName, "label_count", labelCount).
		Wrapf(err, "register prometheus metric")
}

func unexpectedCollectorType(op string, metricName string, labelCount int, collector any) error {
	return oops.In("observabilityx/prometheus").
		With("op", op, "metric", metricName, "label_count", labelCount, "collector_type", fmt.Sprintf("%T", collector)).
		Errorf("prometheus metric has unexpected collector type")
}

func metricSpecKey(spec observabilityx.MetricSpec) string {
	return fmt.Sprintf("%s|%s|%s", spec.Name, spec.Description, spec.Unit)
}

func histogramSpecKey(spec observabilityx.HistogramSpec) string {
	return fmt.Sprintf("%s|%v", metricSpecKey(spec.MetricSpec), spec.Buckets.Values())
}
