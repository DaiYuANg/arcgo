package valkey

import (
	"context"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/samber/lo"
)

// XAdd adds an entry to a stream.
func (a *Adapter) XAdd(ctx context.Context, key, id string, values map[string][]byte) (string, error) {
	resp := a.client.Do(ctx, newXAddCommand(a.client, key, id, values))

	return stringFromResult("add stream entry", resp)
}

// XRead reads entries from a stream.
func (a *Adapter) XRead(ctx context.Context, key, start string, count int64) ([]kvx.StreamEntry, error) {
	resp := a.client.Do(ctx, newXReadCommand(a.client, key, start, count))
	entries, err := xReadEntriesFromResult("read stream", resp)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, nil
	}

	return convertXRangeEntries(entries[key]), nil
}

// XReadMultiple reads entries from multiple streams.
func (a *Adapter) XReadMultiple(ctx context.Context, streams map[string]string, count int64, _ time.Duration) (map[string][]kvx.StreamEntry, error) {
	result := make(map[string][]kvx.StreamEntry, len(streams))
	for _, entry := range lo.Entries(streams) {
		entries, err := a.XRead(ctx, entry.Key, entry.Value, count)
		if err != nil {
			return nil, err
		}
		result[entry.Key] = entries
	}

	return result, nil
}

// XRange reads entries in a range.
func (a *Adapter) XRange(ctx context.Context, key, start, stop string) ([]kvx.StreamEntry, error) {
	resp := a.client.Do(ctx, a.client.B().Xrange().Key(key).Start(start).End(stop).Build())
	entries, err := xRangeEntriesFromResult("range stream", resp)
	if err != nil {
		return nil, err
	}

	return convertXRangeEntries(entries), nil
}

// XRevRange reads entries in reverse order.
func (a *Adapter) XRevRange(ctx context.Context, key, start, stop string) ([]kvx.StreamEntry, error) {
	resp := a.client.Do(ctx, a.client.B().Arbitrary("XREVRANGE").Args(key, start, stop).Build())
	entries, err := xRangeEntriesFromResult("reverse range stream", resp)
	if err != nil {
		return nil, err
	}

	return convertXRangeEntries(entries), nil
}

// XLen gets the number of entries in a stream.
func (a *Adapter) XLen(ctx context.Context, key string) (int64, error) {
	resp := a.client.Do(ctx, a.client.B().Xlen().Key(key).Build())

	return int64FromResult("get stream length", resp)
}

// XTrim trims the stream to approximately maxLen entries.
func (a *Adapter) XTrim(ctx context.Context, key string, maxLen int64) error {
	return wrapValkeyError("trim stream", a.client.Do(ctx, a.client.B().Xtrim().Key(key).Maxlen().Threshold(formatInt64(maxLen)).Build()).Error())
}

// XDel deletes specific entries from a stream.
func (a *Adapter) XDel(ctx context.Context, key string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	args := collectionx.NewListWithCapacity[string](len(ids)+1, key)
	args.Add(ids...)
	return wrapValkeyError("delete stream entries", a.client.Do(ctx, a.client.B().Arbitrary("XDEL").Args(args.Values()...).Build()).Error())
}
