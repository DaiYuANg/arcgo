package redis

import (
	"context"
	"errors"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/kvx"
	goredis "github.com/redis/go-redis/v9"
)

// XGroupCreate creates a consumer group.
func (a *Adapter) XGroupCreate(ctx context.Context, key, group, startID string) error {
	return wrapRedisError("create stream group", a.client.XGroupCreate(ctx, key, group, startID).Err())
}

// XGroupDestroy destroys a consumer group.
func (a *Adapter) XGroupDestroy(ctx context.Context, key, group string) error {
	return wrapRedisError("destroy stream group", a.client.XGroupDestroy(ctx, key, group).Err())
}

// XGroupCreateConsumer creates a consumer in a group.
func (a *Adapter) XGroupCreateConsumer(ctx context.Context, key, group, consumer string) error {
	return wrapRedisError("create stream consumer", a.client.XGroupCreateConsumer(ctx, key, group, consumer).Err())
}

// XGroupDelConsumer deletes a consumer from a group.
func (a *Adapter) XGroupDelConsumer(ctx context.Context, key, group, consumer string) error {
	return wrapRedisError("delete stream consumer", a.client.XGroupDelConsumer(ctx, key, group, consumer).Err())
}

// XReadGroup reads entries as part of a consumer group.
func (a *Adapter) XReadGroup(ctx context.Context, group, consumer string, streams map[string]string, count int64, block time.Duration) (collectionx.MultiMap[string, kvx.StreamEntry], error) {
	result, err := a.client.XReadGroup(ctx, &goredis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  buildStreamPairs(streams),
		Count:    count,
		Block:    block,
	}).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return collectionx.NewMultiMap[string, kvx.StreamEntry](), nil
		}

		return nil, wrapRedisError("read stream group", err)
	}

	return convertStreams(result), nil
}

// XAck acknowledges processing of stream entries.
func (a *Adapter) XAck(ctx context.Context, key, group string, ids []string) error {
	return wrapRedisError("ack stream entries", a.client.XAck(ctx, key, group, ids...).Err())
}

// XPending gets pending entries information.
func (a *Adapter) XPending(ctx context.Context, key, group string) (*kvx.PendingInfo, error) {
	result, err := a.client.XPending(ctx, key, group).Result()
	result, err = wrapRedisResult("get pending stream info", result, err)
	if err != nil {
		return nil, err
	}

	return &kvx.PendingInfo{
		Count:     result.Count,
		StartID:   result.Lower,
		EndID:     result.Higher,
		Consumers: collectionx.NewMapFrom(result.Consumers),
	}, nil
}

// XPendingRange gets pending entries in a range.
func (a *Adapter) XPendingRange(ctx context.Context, key, group, start, stop string, count int64) (collectionx.List[kvx.PendingEntry], error) {
	result, err := a.client.XPendingExt(ctx, &goredis.XPendingExtArgs{
		Stream: key,
		Group:  group,
		Start:  start,
		End:    stop,
		Count:  count,
	}).Result()
	result, err = wrapRedisResult("get pending stream range", result, err)
	if err != nil {
		return nil, err
	}

	return convertPendingEntries(result), nil
}

// XClaim claims pending entries for a consumer.
func (a *Adapter) XClaim(ctx context.Context, key, group, consumer string, minIdleTime time.Duration, ids []string) (collectionx.List[kvx.StreamEntry], error) {
	result, err := a.client.XClaim(ctx, &goredis.XClaimArgs{
		Stream:   key,
		Group:    group,
		Consumer: consumer,
		MinIdle:  minIdleTime,
		Messages: ids,
	}).Result()
	result, err = wrapRedisResult("claim stream entries", result, err)
	if err != nil {
		return nil, err
	}

	return convertStreamMessages(result), nil
}

// XAutoClaim auto-claims pending entries.
func (a *Adapter) XAutoClaim(ctx context.Context, key, group, consumer string, minIdleTime time.Duration, start string, count int64) (string, collectionx.List[kvx.StreamEntry], error) {
	messages, next, err := a.client.XAutoClaim(ctx, &goredis.XAutoClaimArgs{
		Stream:   key,
		Group:    group,
		Consumer: consumer,
		MinIdle:  minIdleTime,
		Start:    start,
		Count:    count,
	}).Result()
	if err != nil {
		return "", nil, wrapRedisError("auto claim stream entries", err)
	}

	return next, convertStreamMessages(messages), nil
}
