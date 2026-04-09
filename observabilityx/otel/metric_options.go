package otel

import (
	"fmt"

	"github.com/DaiYuANg/arcgo/observabilityx"
	"go.opentelemetry.io/otel/metric"
)

func counterOptions(spec observabilityx.CounterSpec) []metric.Int64CounterOption {
	return []metric.Int64CounterOption{
		metric.WithDescription(spec.Description),
		metric.WithUnit(spec.Unit),
	}
}

func upDownCounterOptions(spec observabilityx.UpDownCounterSpec) []metric.Int64UpDownCounterOption {
	return []metric.Int64UpDownCounterOption{
		metric.WithDescription(spec.Description),
		metric.WithUnit(spec.Unit),
	}
}

func histogramOptions(spec observabilityx.HistogramSpec) []metric.Float64HistogramOption {
	options := []metric.Float64HistogramOption{
		metric.WithDescription(spec.Description),
		metric.WithUnit(spec.Unit),
	}
	if spec.Buckets != nil && !spec.Buckets.IsEmpty() {
		options = append(options, metric.WithExplicitBucketBoundaries(spec.Buckets.Values()...))
	}
	return options
}

func gaugeOptions(spec observabilityx.GaugeSpec) []metric.Float64GaugeOption {
	return []metric.Float64GaugeOption{
		metric.WithDescription(spec.Description),
		metric.WithUnit(spec.Unit),
	}
}

func cacheMetricSpecKey(kind string, spec observabilityx.MetricSpec) string {
	return fmt.Sprintf("%s|%s|%s|%s|%v", kind, spec.Name, spec.Description, spec.Unit, spec.LabelKeys.Values())
}

func cacheHistogramSpecKey(spec observabilityx.HistogramSpec) string {
	return fmt.Sprintf("%s|%v", cacheMetricSpecKey("histogram", spec.MetricSpec), spec.Buckets.Values())
}
