package list

import (
	"fmt"
	"strings"

	"github.com/samber/lo"
)

// Join concatenates list items with sep.
// When formatter is omitted, string items are used as-is, []byte items are cast to string,
// and all other items fall back to fmt.Sprint.
func (l *List[T]) Join(sep string, formatters ...func(index int, item T) string) string {
	if l == nil || len(l.items) == 0 {
		return ""
	}

	formatter := defaultListJoinFormatter[T]
	if len(formatters) > 0 && formatters[0] != nil {
		formatter = formatters[0]
	}

	var builder strings.Builder
	lo.ForEach(l.items, func(item T, index int) {
		if index > 0 {
			builder.WriteString(sep)
		}
		builder.WriteString(formatter(index, item))
	})
	return builder.String()
}

func defaultListJoinFormatter[T any](_ int, item T) string {
	switch value := any(item).(type) {
	case string:
		return value
	case []byte:
		return string(value)
	default:
		return fmt.Sprint(value)
	}
}
