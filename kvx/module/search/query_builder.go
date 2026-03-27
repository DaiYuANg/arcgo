package search

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/samber/lo"
)

// QueryBuilder helps build search queries.
type QueryBuilder struct {
	parts []string
}

// NewQueryBuilder creates a new QueryBuilder.
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		parts: make([]string, 0),
	}
}

// Text adds a text search condition.
func (qb *QueryBuilder) Text(field, value string) *QueryBuilder {
	if field == "" {
		qb.parts = append(qb.parts, value)
		return qb
	}

	qb.parts = append(qb.parts, fmt.Sprintf("@%s:%s", field, value))
	return qb
}

// Tag adds a tag search condition (exact match).
func (qb *QueryBuilder) Tag(field, value string) *QueryBuilder {
	qb.parts = append(qb.parts, fmt.Sprintf("@%s:{%s}", field, escapeTag(value)))
	return qb
}

// Tags adds a tag search condition with multiple values (OR).
func (qb *QueryBuilder) Tags(field string, values []string) *QueryBuilder {
	escaped := lo.Map(values, func(value string, _ int) string {
		return escapeTag(value)
	})
	qb.parts = append(qb.parts, fmt.Sprintf("@%s:{%s}", field, strings.Join(escaped, "|")))
	return qb
}

// Range adds a numeric range condition.
func (qb *QueryBuilder) Range(field string, lower, upper float64) *QueryBuilder {
	qb.parts = append(qb.parts, fmt.Sprintf("@%s:[%v %v]", field, formatNumber(lower), formatNumber(upper)))
	return qb
}

// GreaterThan adds a greater than condition.
func (qb *QueryBuilder) GreaterThan(field string, value float64) *QueryBuilder {
	qb.parts = append(qb.parts, fmt.Sprintf("@%s:[(%v +inf]", field, formatNumber(value)))
	return qb
}

// LessThan adds a less than condition.
func (qb *QueryBuilder) LessThan(field string, value float64) *QueryBuilder {
	qb.parts = append(qb.parts, fmt.Sprintf("@%s:[-inf (%v]", field, formatNumber(value)))
	return qb
}

// And combines conditions with AND.
func (qb *QueryBuilder) And() *QueryBuilder {
	if len(qb.parts) > 1 {
		lastTwo := qb.parts[len(qb.parts)-2:]
		qb.parts = qb.parts[:len(qb.parts)-2]
		qb.parts = append(qb.parts, fmt.Sprintf("(%s) (%s)", lastTwo[0], lastTwo[1]))
	}
	return qb
}

// Or combines conditions with OR.
func (qb *QueryBuilder) Or() *QueryBuilder {
	if len(qb.parts) > 1 {
		lastTwo := qb.parts[len(qb.parts)-2:]
		qb.parts = qb.parts[:len(qb.parts)-2]
		qb.parts = append(qb.parts, fmt.Sprintf("(%s)|(%s)", lastTwo[0], lastTwo[1]))
	}
	return qb
}

// Not negates the last condition.
func (qb *QueryBuilder) Not() *QueryBuilder {
	if len(qb.parts) > 0 {
		last := qb.parts[len(qb.parts)-1]
		qb.parts[len(qb.parts)-1] = fmt.Sprintf("-(%s)", last)
	}
	return qb
}

// Build builds the query string.
func (qb *QueryBuilder) Build() string {
	if len(qb.parts) == 0 {
		return "*"
	}
	if len(qb.parts) == 1 {
		return qb.parts[0]
	}
	return strings.Join(qb.parts, " ")
}

// escapeTag escapes special characters in tag values.
func escapeTag(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, ",", "\\,")
	value = strings.ReplaceAll(value, ".", "\\.")
	value = strings.ReplaceAll(value, "<", "\\<")
	value = strings.ReplaceAll(value, ">", "\\>")
	value = strings.ReplaceAll(value, "{", "\\{")
	value = strings.ReplaceAll(value, "}", "\\}")
	value = strings.ReplaceAll(value, "[", "\\[")
	value = strings.ReplaceAll(value, "]", "\\]")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	value = strings.ReplaceAll(value, "'", "\\'")
	value = strings.ReplaceAll(value, ":", "\\:")
	value = strings.ReplaceAll(value, ";", "\\;")
	value = strings.ReplaceAll(value, "!", "\\!")
	value = strings.ReplaceAll(value, "@", "\\@")
	value = strings.ReplaceAll(value, "~", "\\~")
	value = strings.ReplaceAll(value, "$", "\\$")
	value = strings.ReplaceAll(value, "%", "\\%")
	value = strings.ReplaceAll(value, "^", "\\^")
	value = strings.ReplaceAll(value, "&", "\\&")
	value = strings.ReplaceAll(value, "*", "\\*")
	value = strings.ReplaceAll(value, "(", "\\(")
	value = strings.ReplaceAll(value, ")", "\\)")
	value = strings.ReplaceAll(value, "-", "\\-")
	value = strings.ReplaceAll(value, "+", "\\+")
	value = strings.ReplaceAll(value, "=", "\\=")
	value = strings.ReplaceAll(value, "|", "\\|")
	return value
}

func formatNumber(n float64) string {
	if n == float64(int64(n)) {
		return strconv.FormatInt(int64(n), 10)
	}
	return strconv.FormatFloat(n, 'f', -1, 64)
}
