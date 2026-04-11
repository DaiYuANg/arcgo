package prometheus

import (
	"slices"
	"strings"
	"unicode"

	"github.com/DaiYuANg/arcgo/observabilityx"
	prom "github.com/prometheus/client_golang/prometheus"
)

func (a *Adapter) normalizeMetricName(name string) string {
	metricSegment := normalizeMetricSegment(name, "metric")
	return normalizeMetricSegment(a.namespace+"_"+metricSegment, "arcgo_metric")
}

func normalizeMetricSegment(raw, fallback string) string {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		clean = fallback
	}
	replaced := strings.Map(func(r rune) rune {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), r == '_', r == ':':
			return unicode.ToLower(r)
		default:
			return '_'
		}
	}, clean)
	replaced = strings.Trim(replaced, "_")
	if replaced == "" {
		replaced = fallback
	}
	firstRune := rune(replaced[0])
	if !unicode.IsLetter(firstRune) && firstRune != '_' && firstRune != ':' {
		replaced = "_" + replaced
	}
	return replaced
}

func sortedLabelKeys(spec observabilityx.MetricSpec) []string {
	labelKeys := spec.LabelKeys.Values()
	slices.Sort(labelKeys)
	return labelKeys
}

func toPromLabels(labelNames []string, values map[string]string) prom.Labels {
	if len(labelNames) == 0 {
		return prom.Labels{}
	}

	labels := make(prom.Labels, len(labelNames))
	for _, labelName := range labelNames {
		labels[labelName] = values[labelName]
	}
	return labels
}
