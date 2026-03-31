package prometheus

import (
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/DaiYuANg/arcgo/observabilityx"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/samber/lo"
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

	entries := lo.FilterMap(attrs, func(attr observabilityx.Attribute, _ int) (lo.Entry[string, string], bool) {
		labelKey := normalizeLabelKey(attr.Key)
		if labelKey == "" {
			return lo.Entry[string, string]{}, false
		}
		return lo.Entry[string, string]{
			Key:   labelKey,
			Value: fmt.Sprint(attr.Value),
		}, true
	})
	labels := lo.Associate(entries, func(entry lo.Entry[string, string]) (string, string) {
		return entry.Key, entry.Value
	})
	if len(labels) == 0 {
		return nil
	}
	return labels
}

func sortedLabelKeys(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	keys := lo.Keys(values)
	slices.Sort(keys)
	return keys
}

func toPromLabels(labelNames []string, values map[string]string) prom.Labels {
	if len(labelNames) == 0 {
		return prom.Labels{}
	}
	return prom.Labels(lo.Associate(labelNames, func(labelName string) (string, string) {
		return labelName, values[labelName]
	}))
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
