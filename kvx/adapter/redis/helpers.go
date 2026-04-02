package redis

import (
	"fmt"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/lo"
)

func convertBytesMapToAny(values map[string][]byte) map[string]any {
	return lo.MapValues(values, func(value []byte, _ string) any {
		return value
	})
}

func convertInterfaceMapToBytes(m map[string]any) map[string][]byte {
	return lo.MapValues(m, func(value any, _ string) []byte {
		switch val := value.(type) {
		case []byte:
			return val
		case string:
			return []byte(val)
		default:
			return fmt.Append(nil, val)
		}
	})
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

func parseFTSearchResponse(val any) collectionx.List[string] {
	arr, ok := val.([]any)
	if !ok || len(arr) < 1 {
		return collectionx.NewList[string]()
	}

	return collectionx.FilterMapList(collectionx.NewListWithCapacity(len(arr)-1, arr[1:]...), func(index int, item any) (string, bool) {
		if index%2 != 0 {
			return "", false
		}
		key, ok := item.(string)
		return key, ok
	})
}

func parseFTAggregateResponse(val any) []map[string]any {
	arr, ok := val.([]any)
	if !ok || len(arr) < 1 {
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

	return lo.Reduce(lo.Range(len(values)/2), func(result map[string]any, pairIndex int, _ int) map[string]any {
		key, ok := values[pairIndex*2].(string)
		if ok {
			result[key] = values[pairIndex*2+1]
		}
		return result
	}, make(map[string]any, len(values)/2))
}
