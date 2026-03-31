package valkey

import (
	"strconv"

	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/samber/lo"
	"github.com/valkey-io/valkey-go"
)

func binaryArgs(args [][]byte) []string {
	return lo.Map(args, func(arg []byte, _ int) string {
		return valkey.BinaryString(arg)
	})
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
	return lo.Map(entries, func(entry valkey.XRangeEntry, _ int) kvx.StreamEntry {
		return kvx.StreamEntry{
			ID:     entry.ID,
			Values: convertStringMapToBytes(entry.FieldValues),
		}
	})
}

func convertXReadEntries(entries map[string][]valkey.XRangeEntry) map[string][]kvx.StreamEntry {
	result := make(map[string][]kvx.StreamEntry, len(entries))
	for stream, items := range entries {
		result[stream] = convertXRangeEntries(items)
	}

	return result
}

func searchDocsToKeys(docs []valkey.FtSearchDoc) []string {
	return lo.Map(docs, func(doc valkey.FtSearchDoc, _ int) string {
		return doc.Key
	})
}

func aggregateDocsToRows(docs []map[string]string) []map[string]any {
	return lo.Map(docs, func(doc map[string]string, _ int) map[string]any {
		return lo.MapValues(doc, func(value string, _ string) any {
			return value
		})
	})
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
