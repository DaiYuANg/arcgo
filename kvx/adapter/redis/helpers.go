package redis

import (
	"fmt"

	"github.com/samber/lo"
)

func convertBytesMapToAny(values map[string][]byte) map[string]any {
	result := make(map[string]any, len(values))
	for key, value := range values {
		result[key] = value
	}

	return result
}

func convertInterfaceMapToBytes(m map[string]any) map[string][]byte {
	result := make(map[string][]byte, len(m))
	for k, v := range m {
		switch val := v.(type) {
		case []byte:
			result[k] = val
		case string:
			result[k] = []byte(val)
		default:
			result[k] = fmt.Append(nil, val)
		}
	}

	return result
}

func valueToBytes(val any) []byte {
	switch v := val.(type) {
	case []byte:
		return v
	case string:
		return []byte(v)
	case nil:
		return nil
	default:
		return fmt.Append(nil, v)
	}
}

func parseFTSearchResponse(val any) []string {
	arr, ok := val.([]any)
	if !ok {
		return nil
	}

	if len(arr) < 1 {
		return nil
	}

	keys := make([]string, 0, len(arr)/2)
	for i := 1; i < len(arr); i += 2 {
		if key, ok := arr[i].(string); ok {
			keys = append(keys, key)
		}
	}

	return keys
}

func parseFTAggregateResponse(val any) []map[string]any {
	arr, ok := val.([]any)
	if !ok {
		return nil
	}

	if len(arr) < 1 {
		return nil
	}

	return lo.FilterMap(arr[1:], func(row any, _ int) (map[string]any, bool) {
		parsed := parseFTAggregateRow(row)
		return parsed, parsed != nil
	})
}

func parseFTAggregateRow(row any) map[string]any {
	values, ok := row.([]any)
	if !ok || len(values) == 0 {
		return nil
	}

	result := make(map[string]any, len(values)/2)
	for i := 0; i < len(values)-1; i += 2 {
		key, ok := values[i].(string)
		if !ok {
			continue
		}

		result[key] = values[i+1]
	}

	return result
}
