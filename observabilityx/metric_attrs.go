package observabilityx

import (
	"fmt"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/lo"
)

// FilterMetricAttributes returns attributes constrained to the declared label schema.
func FilterMetricAttributes(labelKeys collectionx.List[string], attrs ...Attribute) []Attribute {
	normalizedKeys := valuesOrEmpty(normalizeLabelKeys(labelKeys))
	if len(normalizedKeys) == 0 || len(attrs) == 0 {
		return nil
	}

	labelMap := MetricLabelMap(labelKeys, attrs...)
	return lo.FilterMap(normalizedKeys, func(labelKey string, _ int) (Attribute, bool) {
		value, ok := labelMap[labelKey]
		if !ok {
			return Attribute{}, false
		}
		return String(labelKey, value), true
	})
}

// MetricLabelMap returns a normalized label map constrained to the declared label schema.
func MetricLabelMap(labelKeys collectionx.List[string], attrs ...Attribute) map[string]string {
	normalizedKeys := valuesOrEmpty(normalizeLabelKeys(labelKeys))
	if len(normalizedKeys) == 0 || len(attrs) == 0 {
		return nil
	}

	allowed := collectionx.NewList(normalizedKeys...).Values()
	return lo.PickBy(buildMetricLabelMap(attrs), func(key string, _ string) bool {
		return lo.Contains(allowed, key)
	})
}

func buildMetricLabelMap(attrs []Attribute) map[string]string {
	if len(attrs) == 0 {
		return nil
	}

	entries := lo.FilterMap(attrs, func(attr Attribute, _ int) (lo.Entry[string, string], bool) {
		labelKey := normalizeMetricLabelKey(attr.Key)
		if labelKey == "" {
			return lo.Entry[string, string]{}, false
		}
		return lo.Entry[string, string]{
			Key:   labelKey,
			Value: fmt.Sprint(attr.Value),
		}, true
	})
	if len(entries) == 0 {
		return nil
	}
	return lo.Associate(entries, func(entry lo.Entry[string, string]) (string, string) {
		return entry.Key, entry.Value
	})
}
