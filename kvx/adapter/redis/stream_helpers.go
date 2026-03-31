package redis

import (
	"github.com/DaiYuANg/arcgo/kvx"
	goredis "github.com/redis/go-redis/v9"
	"github.com/samber/lo"
)

func buildStreamPairs(streams map[string]string) []string {
	return lo.FlatMap(lo.Entries(streams), func(entry lo.Entry[string, string], _ int) []string {
		return []string{entry.Key, entry.Value}
	})
}

func newXAddArgs(key, id string, values map[string][]byte) *goredis.XAddArgs {
	args := &goredis.XAddArgs{
		Stream: key,
		Values: convertBytesMapToAny(values),
	}
	if id != "*" {
		args.ID = id
	}

	return args
}

func convertStreamMessages(messages []goredis.XMessage) []kvx.StreamEntry {
	return lo.Map(messages, func(msg goredis.XMessage, _ int) kvx.StreamEntry {
		return kvx.StreamEntry{
			ID:     msg.ID,
			Values: convertInterfaceMapToBytes(msg.Values),
		}
	})
}

func convertStreams(streams []goredis.XStream) map[string][]kvx.StreamEntry {
	return lo.Associate(streams, func(stream goredis.XStream) (string, []kvx.StreamEntry) {
		return stream.Stream, convertStreamMessages(stream.Messages)
	})
}

func convertPendingEntries(pending []goredis.XPendingExt) []kvx.PendingEntry {
	return lo.Map(pending, func(item goredis.XPendingExt, _ int) kvx.PendingEntry {
		return kvx.PendingEntry{
			ID:         item.ID,
			Consumer:   item.Consumer,
			IdleTime:   item.Idle,
			Deliveries: item.RetryCount,
		}
	})
}

func convertGroupInfos(groups []goredis.XInfoGroup) []kvx.GroupInfo {
	return lo.Map(groups, func(group goredis.XInfoGroup, _ int) kvx.GroupInfo {
		return kvx.GroupInfo{
			Name:            group.Name,
			Consumers:       group.Consumers,
			Pending:         group.Pending,
			LastDeliveredID: group.LastDeliveredID,
		}
	})
}

func convertConsumerInfos(consumers []goredis.XInfoConsumer) []kvx.ConsumerInfo {
	return lo.Map(consumers, func(consumer goredis.XInfoConsumer, _ int) kvx.ConsumerInfo {
		return kvx.ConsumerInfo{
			Name:    consumer.Name,
			Pending: consumer.Pending,
			Idle:    consumer.Idle,
		}
	})
}

func convertStreamInfo(info *goredis.XInfoStream) *kvx.StreamInfo {
	result := &kvx.StreamInfo{
		Length:          info.Length,
		RadixTreeKeys:   info.RadixTreeKeys,
		RadixTreeNodes:  info.RadixTreeNodes,
		Groups:          info.Groups,
		LastGeneratedID: info.LastGeneratedID,
	}

	if info.FirstEntry.ID != "" {
		result.FirstEntry = &kvx.StreamEntry{
			ID:     info.FirstEntry.ID,
			Values: convertInterfaceMapToBytes(info.FirstEntry.Values),
		}
	}

	if info.LastEntry.ID != "" {
		result.LastEntry = &kvx.StreamEntry{
			ID:     info.LastEntry.ID,
			Values: convertInterfaceMapToBytes(info.LastEntry.Values),
		}
	}

	return result
}
