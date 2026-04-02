package valkey

import (
	"strconv"

	"github.com/DaiYuANg/arcgo/collectionx"
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
	entries := lo.Entries(streams)
	return lo.Map(entries, func(entry lo.Entry[string, string], _ int) string {
			return entry.Key
		}),
		lo.Map(entries, func(entry lo.Entry[string, string], _ int) string {
			return entry.Value
		})
}

func convertStringMapToBytes(values map[string]string) map[string][]byte {
	return lo.MapValues(values, func(value string, _ string) []byte {
		return []byte(value)
	})
}

func convertXRangeEntries(entries []valkey.XRangeEntry) collectionx.List[kvx.StreamEntry] {
	return collectionx.MapList(collectionx.NewListWithCapacity(len(entries), entries...), func(_ int, entry valkey.XRangeEntry) kvx.StreamEntry {
		return kvx.StreamEntry{
			ID:     entry.ID,
			Values: collectionx.NewMapFrom(convertStringMapToBytes(entry.FieldValues)),
		}
	})
}

func convertXReadEntries(entries map[string][]valkey.XRangeEntry) collectionx.MultiMap[string, kvx.StreamEntry] {
	result := collectionx.NewMultiMapWithCapacity[string, kvx.StreamEntry](len(entries))
	lo.ForEach(lo.Entries(entries), func(entry lo.Entry[string, []valkey.XRangeEntry], _ int) {
		result.Set(entry.Key, convertXRangeEntries(entry.Value).Values()...)
	})
	return result
}

func searchDocsToKeys(docs []valkey.FtSearchDoc) collectionx.List[string] {
	return collectionx.MapList(collectionx.NewListWithCapacity(len(docs), docs...), func(_ int, doc valkey.FtSearchDoc) string {
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
	args := collectionx.NewListWithCapacity[string](len(keys)*2+7, "GROUP", group, consumer)
	if count > 0 {
		args.Add("COUNT", strconv.FormatInt(count, 10))
	}
	if block > 0 {
		args.Add("BLOCK", strconv.FormatInt(block, 10))
	}
	args.Add("STREAMS")
	args.Add(keys...)
	args.Add(ids...)

	return args.Values()
}
