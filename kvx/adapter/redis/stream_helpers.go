package redis

import (
	"github.com/DaiYuANg/arcgo/kvx"
	goredis "github.com/redis/go-redis/v9"
)

func buildStreamPairs(streams map[string]string) []string {
	pairs := make([]string, 0, len(streams)*2)
	for key, start := range streams {
		pairs = append(pairs, key, start)
	}

	return pairs
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
	entries := make([]kvx.StreamEntry, len(messages))
	for i, msg := range messages {
		entries[i] = kvx.StreamEntry{
			ID:     msg.ID,
			Values: convertInterfaceMapToBytes(msg.Values),
		}
	}

	return entries
}

func convertStreams(streams []goredis.XStream) map[string][]kvx.StreamEntry {
	entries := make(map[string][]kvx.StreamEntry, len(streams))
	for _, stream := range streams {
		entries[stream.Stream] = convertStreamMessages(stream.Messages)
	}

	return entries
}

func convertPendingEntries(pending []goredis.XPendingExt) []kvx.PendingEntry {
	entries := make([]kvx.PendingEntry, len(pending))
	for i, item := range pending {
		entries[i] = kvx.PendingEntry{
			ID:         item.ID,
			Consumer:   item.Consumer,
			IdleTime:   item.Idle,
			Deliveries: item.RetryCount,
		}
	}

	return entries
}

func convertGroupInfos(groups []goredis.XInfoGroup) []kvx.GroupInfo {
	result := make([]kvx.GroupInfo, len(groups))
	for i, group := range groups {
		result[i] = kvx.GroupInfo{
			Name:            group.Name,
			Consumers:       group.Consumers,
			Pending:         group.Pending,
			LastDeliveredID: group.LastDeliveredID,
		}
	}

	return result
}

func convertConsumerInfos(consumers []goredis.XInfoConsumer) []kvx.ConsumerInfo {
	result := make([]kvx.ConsumerInfo, len(consumers))
	for i, consumer := range consumers {
		result[i] = kvx.ConsumerInfo{
			Name:    consumer.Name,
			Pending: consumer.Pending,
			Idle:    consumer.Idle,
		}
	}

	return result
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
