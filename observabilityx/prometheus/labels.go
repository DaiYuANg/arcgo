package prometheus

import (
	"fmt"
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

func attrsToLabelMap(attrs []observabilityx.Attribute) map[string]string {
	if len(attrs) == 0 {
		return nil
	}

	labels := make(map[string]string, len(attrs))
	for _, attr := range attrs {
		labelKey := normalizeLabelKey(attr.Key)
		if labelKey == "" {
			continue
		}
		labels[labelKey] = fmt.Sprint(attr.Value)
	}
	if len(labels) == 0 {
		return nil
	}
	return labels
}

func sortedLabelKeys(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
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

func normalizeLabelKey(raw string) string {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return ""
	}

	replaced := strings.Map(func(r rune) rune {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), r == '_':
			return unicode.ToLower(r)
		default:
			return '_'
		}
	}, clean)
	replaced = strings.Trim(replaced, "_")
	if replaced == "" {
		return ""
	}

	firstRune := rune(replaced[0])
	if !unicode.IsLetter(firstRune) && firstRune != '_' {
		replaced = "_" + replaced
	}
	return replaced
}
