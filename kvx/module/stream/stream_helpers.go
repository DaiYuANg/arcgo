package stream

import "github.com/DaiYuANg/arcgo/kvx"

func buildByteValues(values map[string]any) (map[string][]byte, error) {
	byteValues := make(map[string][]byte, len(values))
	for key, value := range values {
		data, err := convertToBytes(value)
		if err != nil {
			return nil, err
		}
		byteValues[key] = data
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

func limitEntries(entries []kvx.StreamEntry, count int64) []kvx.StreamEntry {
	if count <= 0 || count >= int64(len(entries)) {
		return entries
	}

	return entries[:count]
}
