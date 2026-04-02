package redis

import (
	"github.com/DaiYuANg/arcgo/collectionx"
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

func convertStreamMessages(messages []goredis.XMessage) collectionx.List[kvx.StreamEntry] {
	return collectionx.MapList(collectionx.NewListWithCapacity(len(messages), messages...), func(_ int, msg goredis.XMessage) kvx.StreamEntry {
		return kvx.StreamEntry{
			ID:     msg.ID,
			Values: collectionx.NewMapFrom(convertInterfaceMapToBytes(msg.Values)),
		}
	})
}

func convertStreams(streams []goredis.XStream) collectionx.MultiMap[string, kvx.StreamEntry] {
	result := collectionx.NewMultiMapWithCapacity[string, kvx.StreamEntry](len(streams))
	lo.ForEach(streams, func(stream goredis.XStream, _ int) {
		result.Set(stream.Stream, convertStreamMessages(stream.Messages).Values()...)
	})
	return result
}

func convertPendingEntries(pending []goredis.XPendingExt) collectionx.List[kvx.PendingEntry] {
	return collectionx.MapList(collectionx.NewListWithCapacity(len(pending), pending...), func(_ int, item goredis.XPendingExt) kvx.PendingEntry {
		return kvx.PendingEntry{
			ID:         item.ID,
			Consumer:   item.Consumer,
			IdleTime:   item.Idle,
			Deliveries: item.RetryCount,
		}
	})
}

func convertGroupInfos(groups []goredis.XInfoGroup) collectionx.List[kvx.GroupInfo] {
	return collectionx.MapList(collectionx.NewListWithCapacity(len(groups), groups...), func(_ int, group goredis.XInfoGroup) kvx.GroupInfo {
		return kvx.GroupInfo{
			Name:            group.Name,
			Consumers:       group.Consumers,
			Pending:         group.Pending,
			LastDeliveredID: group.LastDeliveredID,
		}
	})
}

func convertConsumerInfos(consumers []goredis.XInfoConsumer) collectionx.List[kvx.ConsumerInfo] {
	return collectionx.MapList(collectionx.NewListWithCapacity(len(consumers), consumers...), func(_ int, consumer goredis.XInfoConsumer) kvx.ConsumerInfo {
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
			Values: collectionx.NewMapFrom(convertInterfaceMapToBytes(info.FirstEntry.Values)),
		}
	}

	if info.LastEntry.ID != "" {
		result.LastEntry = &kvx.StreamEntry{
			ID:     info.LastEntry.ID,
			Values: collectionx.NewMapFrom(convertInterfaceMapToBytes(info.LastEntry.Values)),
		}
	}

	return result
}
