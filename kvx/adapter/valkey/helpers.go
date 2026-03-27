package valkey

import (
	"strconv"

	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/valkey-io/valkey-go"
)

func binaryArgs(args [][]byte) []string {
	values := make([]string, len(args))
	for i, arg := range args {
		values[i] = valkey.BinaryString(arg)
	}

	return values
}

func newHSetCommand(client valkey.Client, key string, values map[string][]byte) valkey.Completed {
	cmd := client.B().Hset().Key(key).FieldValue()
	for field, value := range values {
		cmd = cmd.FieldValue(field, valkey.BinaryString(value))
	}

	return cmd.Build()
}

func newXAddCommand(client valkey.Client, key, id string, values map[string][]byte) valkey.Completed {
	cmd := client.B().Xadd().Key(key).Id(id).FieldValue()
	for field, value := range values {
		cmd = cmd.FieldValue(field, valkey.BinaryString(value))
	}

	return cmd.Build()
}

func newXReadCommand(client valkey.Client, key, start string, count int64) valkey.Completed {
	if count > 0 {
		return client.B().Xread().Count(count).Block(0).Streams().Key(key).Id(start).Build()
	}

	return client.B().Xread().Block(0).Streams().Key(key).Id(start).Build()
}

func streamNamesAndIDs(streams map[string]string) ([]string, []string) {
	names := make([]string, 0, len(streams))
	ids := make([]string, 0, len(streams))
	for name, id := range streams {
		names = append(names, name)
		ids = append(ids, id)
	}

	return names, ids
}

func convertStringMapToBytes(values map[string]string) map[string][]byte {
	result := make(map[string][]byte, len(values))
	for key, value := range values {
		result[key] = []byte(value)
	}

	return result
}

func convertXRangeEntries(entries []valkey.XRangeEntry) []kvx.StreamEntry {
	result := make([]kvx.StreamEntry, len(entries))
	for i, entry := range entries {
		result[i] = kvx.StreamEntry{
			ID:     entry.ID,
			Values: convertStringMapToBytes(entry.FieldValues),
		}
	}

	return result
}

func convertXReadEntries(entries map[string][]valkey.XRangeEntry) map[string][]kvx.StreamEntry {
	result := make(map[string][]kvx.StreamEntry, len(entries))
	for stream, items := range entries {
		result[stream] = convertXRangeEntries(items)
	}

	return result
}

func searchDocsToKeys(docs []valkey.FtSearchDoc) []string {
	keys := make([]string, len(docs))
	for i, doc := range docs {
		keys[i] = doc.Key
	}

	return keys
}

func aggregateDocsToRows(docs []map[string]string) []map[string]any {
	rows := make([]map[string]any, len(docs))
	for i, doc := range docs {
		row := make(map[string]any, len(doc))
		for key, value := range doc {
			row[key] = value
		}
		rows[i] = row
	}

	return rows
}

func formatInt64(value int64) string {
	return strconv.FormatInt(value, 10)
}

func buildXReadGroupArgs(group, consumer string, streams map[string]string, count, block int64) []string {
	keys, ids := streamNamesAndIDs(streams)
	args := make([]string, 0, len(keys)*2+7)
	args = append(args, "GROUP", group, consumer)
	if count > 0 {
		args = append(args, "COUNT", strconv.FormatInt(count, 10))
	}
	if block > 0 {
		args = append(args, "BLOCK", strconv.FormatInt(block, 10))
	}
	args = append(args, "STREAMS")
	args = append(args, keys...)
	args = append(args, ids...)

	return args
}
