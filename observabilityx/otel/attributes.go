package otel

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
)

func toOTelAttributes(attrs []observabilityx.Attribute) []attribute.KeyValue {
	if len(attrs) == 0 {
		return nil
	}

	return lo.FilterMap(attrs, func(attr observabilityx.Attribute, _ int) (attribute.KeyValue, bool) {
		key := strings.TrimSpace(attr.Key)
		if key == "" {
			return attribute.KeyValue{}, false
		}
		return toOTelAttribute(key, attr.Value), true
	})
}

func toOTelAttribute(key string, value any) attribute.KeyValue {
	k := attribute.Key(key)

	if text, ok := value.(string); ok {
		return k.String(text)
	}
	if flag, ok := value.(bool); ok {
		return k.Bool(flag)
	}
	if duration, ok := value.(time.Duration); ok {
		return k.Int64(duration.Milliseconds())
	}
	if attr, ok := toSignedIntAttribute(k, value); ok {
		return attr
	}
	if attr, ok := toFloatAttribute(k, value); ok {
		return attr
	}
	if attr, ok := toUnsignedAttribute(k, value); ok {
		return attr
	}

	return k.String(fmt.Sprint(value))
}

func toSignedIntAttribute(key attribute.Key, value any) (attribute.KeyValue, bool) {
	switch typed := value.(type) {
	case int:
		return key.Int(typed), true
	case int8:
		return key.Int64(int64(typed)), true
	case int16:
		return key.Int64(int64(typed)), true
	case int32:
		return key.Int64(int64(typed)), true
	case int64:
		return key.Int64(typed), true
	default:
		return attribute.KeyValue{}, false
	}
}

func toFloatAttribute(key attribute.Key, value any) (attribute.KeyValue, bool) {
	switch typed := value.(type) {
	case float32:
		return key.Float64(float64(typed)), true
	case float64:
		return key.Float64(typed), true
	default:
		return attribute.KeyValue{}, false
	}
}

func toUnsignedAttribute(key attribute.Key, value any) (attribute.KeyValue, bool) {
	switch typed := value.(type) {
	case uint:
		return key.String(strconv.FormatUint(uint64(typed), 10)), true
	case uint8:
		return key.String(strconv.FormatUint(uint64(typed), 10)), true
	case uint16:
		return key.String(strconv.FormatUint(uint64(typed), 10)), true
	case uint32:
		return key.String(strconv.FormatUint(uint64(typed), 10)), true
	case uint64:
		return key.String(strconv.FormatUint(typed, 10)), true
	default:
		return attribute.KeyValue{}, false
	}
}