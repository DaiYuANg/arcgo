package stream

import (
	"fmt"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/samber/lo"
)

func buildByteValues(values map[string]any) (map[string][]byte, error) {
	byteValues, err := lo.ReduceErr(lo.Entries(values), func(byteValues map[string][]byte, entry lo.Entry[string, any], _ int) (map[string][]byte, error) {
		data, err := convertToBytes(entry.Value)
		if err != nil {
			return nil, err
		}
		byteValues[entry.Key] = data
		return byteValues, nil
	}, make(map[string][]byte, len(values)))
	if err != nil {
		return nil, fmt.Errorf("build stream values: %w", err)
	}
	return byteValues, nil
}

func convertToBytes(v any) ([]byte, error) {
	switch val := v.(type) {
	case []byte:
		return val, nil
	case string:
		return []byte(val), nil
	case nil:
		return []byte(""), nil
	default:
		return marshalJSON(v, "marshal stream value")
	}
}

func limitEntries(entries collectionx.List[kvx.StreamEntry], count int64) collectionx.List[kvx.StreamEntry] {
	if entries == nil || count <= 0 || count >= int64(entries.Len()) {
		return entries
	}

	return entries.Take(int(count))
}

func streamEntriesFromMultiMap(results collectionx.MultiMap[string, kvx.StreamEntry], streamKey string) collectionx.List[kvx.StreamEntry] {
	entries := results.Get(streamKey)
	if len(entries) == 0 {
		return collectionx.NewList[kvx.StreamEntry]()
	}
	return collectionx.NewListWithCapacity(len(entries), entries...)
}
