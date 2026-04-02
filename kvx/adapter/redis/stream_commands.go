package redis

import (
	"context"
	"errors"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/kvx"
	goredis "github.com/redis/go-redis/v9"
)

// XAdd adds an entry to a stream.
func (a *Adapter) XAdd(ctx context.Context, key, id string, values map[string][]byte) (string, error) {
	entryID, err := a.client.XAdd(ctx, newXAddArgs(key, id, values)).Result()
	return wrapRedisResult("add stream entry", entryID, err)
}

// XRead reads entries from a stream.
func (a *Adapter) XRead(ctx context.Context, key, start string, count int64) (collectionx.List[kvx.StreamEntry], error) {
	result, err := a.client.XRead(ctx, &goredis.XReadArgs{
		Streams: []string{key, start},
		Count:   count,
		Block:   0,
	}).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return collectionx.NewList[kvx.StreamEntry](), nil
		}

		return nil, wrapRedisError("read stream", err)
	}

	if len(result) == 0 {
		return collectionx.NewList[kvx.StreamEntry](), nil
	}

	return convertStreamMessages(result[0].Messages), nil
}

// XReadMultiple reads entries from multiple streams.
func (a *Adapter) XReadMultiple(ctx context.Context, streams map[string]string, count int64, block time.Duration) (collectionx.MultiMap[string, kvx.StreamEntry], error) {
	result, err := a.client.XRead(ctx, &goredis.XReadArgs{
		Streams: buildStreamPairs(streams),
		Count:   count,
		Block:   block,
	}).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return collectionx.NewMultiMap[string, kvx.StreamEntry](), nil
		}

		return nil, wrapRedisError("read multiple streams", err)
	}

	return convertStreams(result), nil
}

// XRange reads entries in a range.
func (a *Adapter) XRange(ctx context.Context, key, start, stop string) (collectionx.List[kvx.StreamEntry], error) {
	result, err := a.client.XRange(ctx, key, start, stop).Result()
	result, err = wrapRedisResult("range stream", result, err)
	if err != nil {
		return nil, err
	}

	return convertStreamMessages(result), nil
}

// XRevRange reads entries in reverse order.
func (a *Adapter) XRevRange(ctx context.Context, key, start, stop string) (collectionx.List[kvx.StreamEntry], error) {
	result, err := a.client.XRevRange(ctx, key, start, stop).Result()
	result, err = wrapRedisResult("reverse range stream", result, err)
	if err != nil {
		return nil, err
	}

	return convertStreamMessages(result), nil
}

// XLen gets the number of entries in a stream.
func (a *Adapter) XLen(ctx context.Context, key string) (int64, error) {
	length, err := a.client.XLen(ctx, key).Result()
	return wrapRedisResult("get stream length", length, err)
}

// XTrim trims the stream to approximately maxLen entries.
func (a *Adapter) XTrim(ctx context.Context, key string, maxLen int64) error {
	return wrapRedisError("trim stream", a.client.XTrimMaxLen(ctx, key, maxLen).Err())
}

// XDel deletes specific entries from a stream.
func (a *Adapter) XDel(ctx context.Context, key string, ids []string) error {
	return wrapRedisError("delete stream entries", a.client.XDel(ctx, key, ids...).Err())
}
