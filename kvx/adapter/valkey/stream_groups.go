package valkey

import (
	"context"
	"time"

	"github.com/DaiYuANg/arcgo/kvx"
)

// XGroupCreate creates a consumer group.
func (a *Adapter) XGroupCreate(ctx context.Context, key, group, startID string) error {
	return wrapValkeyError("create stream group", a.client.Do(ctx, a.client.B().Arbitrary("XGROUP", "CREATE").Args(key, group, startID).Build()).Error())
}

// XGroupDestroy destroys a consumer group.
func (a *Adapter) XGroupDestroy(ctx context.Context, key, group string) error {
	return wrapValkeyError("destroy stream group", a.client.Do(ctx, a.client.B().Arbitrary("XGROUP", "DESTROY").Args(key, group).Build()).Error())
}

// XGroupCreateConsumer creates a consumer in a group.
func (a *Adapter) XGroupCreateConsumer(ctx context.Context, key, group, consumer string) error {
	return wrapValkeyError("create stream consumer", a.client.Do(ctx, a.client.B().Arbitrary("XGROUP", "CREATECONSUMER").Args(key, group, consumer).Build()).Error())
}

// XGroupDelConsumer deletes a consumer from a group.
func (a *Adapter) XGroupDelConsumer(ctx context.Context, key, group, consumer string) error {
	return wrapValkeyError("delete stream consumer", a.client.Do(ctx, a.client.B().Arbitrary("XGROUP", "DELCONSUMER").Args(key, group, consumer).Build()).Error())
}

// XReadGroup reads entries as part of a consumer group.
func (a *Adapter) XReadGroup(ctx context.Context, group, consumer string, streams map[string]string, count int64, block time.Duration) (map[string][]kvx.StreamEntry, error) {
	if len(streams) == 0 {
		return map[string][]kvx.StreamEntry{}, nil
	}

	resp := a.client.Do(ctx, a.client.B().Arbitrary("XREADGROUP").Args(buildXReadGroupArgs(group, consumer, streams, count, block.Milliseconds())...).Build())
	entries, err := xReadEntriesFromResult("read stream group", resp)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return map[string][]kvx.StreamEntry{}, nil
	}

	return convertXReadEntries(entries), nil
}

// XAck acknowledges processing of stream entries.
func (a *Adapter) XAck(ctx context.Context, key, group string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	args := make([]string, 0, len(ids)+2)
	args = append(args, key, group)
	args = append(args, ids...)

	return wrapValkeyError("ack stream entries", a.client.Do(ctx, a.client.B().Arbitrary("XACK").Args(args...).Build()).Error())
}
