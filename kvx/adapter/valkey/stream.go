package valkey

import (
	"context"
	"errors"
	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/valkey-io/valkey-go"
	"strconv"
	"time"
)

// ============== Stream Interface ==============

// XAdd adds an entry to a stream.
func (a *Adapter) XAdd(ctx context.Context, key string, id string, values map[string][]byte) (string, error) {
	// Build the command with FieldValue chain
	cmd := a.client.B().Xadd().Key(key).Id(id).FieldValue()
	for k, v := range values {
		cmd = cmd.FieldValue(k, valkey.BinaryString(v))
	}

	resp := a.client.Do(ctx, cmd.Build())
	if resp.Error() != nil {
		return "", resp.Error()
	}
	return resp.ToString()
}

// XRead reads entries from a stream.
func (a *Adapter) XRead(ctx context.Context, key string, start string, count int64) ([]kvx.StreamEntry, error) {
	var cmd valkey.Completed
	if count > 0 {
		cmd = a.client.B().Xread().Count(count).Block(0).Streams().Key(key).Id(start).Build()
	} else {
		cmd = a.client.B().Xread().Block(0).Streams().Key(key).Id(start).Build()
	}

	resp := a.client.Do(ctx, cmd)
	if resp.Error() != nil {
		if valkey.IsValkeyNil(resp.Error()) {
			return nil, nil
		}
		return nil, resp.Error()
	}

	// Parse XREAD response using AsXRead
	xreadResult, err := resp.AsXRead()
	if err != nil {
		return nil, err
	}

	entries := make([]kvx.StreamEntry, 0)
	for _, streamEntries := range xreadResult {
		for _, entry := range streamEntries {
			values := make(map[string][]byte)
			for f, v := range entry.FieldValues {
				values[f] = []byte(v)
			}
			entries = append(entries, kvx.StreamEntry{
				ID:     entry.ID,
				Values: values,
			})
		}
	}

	return entries, nil
}

// XRange reads entries in a range.
func (a *Adapter) XRange(ctx context.Context, key string, start, stop string) ([]kvx.StreamEntry, error) {
	resp := a.client.Do(ctx, a.client.B().Xrange().Key(key).Start(start).End(stop).Build())
	if resp.Error() != nil {
		return nil, resp.Error()
	}

	// Parse XRANGE response using AsXRange
	xrangeEntries, err := resp.AsXRange()
	if err != nil {
		return nil, err
	}

	entries := make([]kvx.StreamEntry, len(xrangeEntries))
	for i, entry := range xrangeEntries {
		values := make(map[string][]byte)
		for f, v := range entry.FieldValues {
			values[f] = []byte(v)
		}
		entries[i] = kvx.StreamEntry{
			ID:     entry.ID,
			Values: values,
		}
	}

	return entries, nil
}

// XLen gets the number of entries in a stream.
func (a *Adapter) XLen(ctx context.Context, key string) (int64, error) {
	resp := a.client.Do(ctx, a.client.B().Xlen().Key(key).Build())
	if resp.Error() != nil {
		return 0, resp.Error()
	}
	return resp.AsInt64()
}

// XTrim trims the stream to approximately maxLen entries.
func (a *Adapter) XTrim(ctx context.Context, key string, maxLen int64) error {
	return a.client.Do(ctx, a.client.B().Xtrim().Key(key).Maxlen().Threshold(strconv.FormatInt(maxLen, 10)).Build()).Error()
}

// XReadMultiple reads entries from multiple streams.
func (a *Adapter) XReadMultiple(ctx context.Context, streams map[string]string, count int64, _ time.Duration) (map[string][]kvx.StreamEntry, error) {
	result := make(map[string][]kvx.StreamEntry, len(streams))
	for key, start := range streams {
		entries, err := a.XRead(ctx, key, start, count)
		if err != nil {
			return nil, err
		}
		result[key] = entries
	}
	return result, nil
}

// XRevRange reads entries in reverse order.
func (a *Adapter) XRevRange(ctx context.Context, key string, start, stop string) ([]kvx.StreamEntry, error) {
	resp := a.client.Do(ctx, a.client.B().Arbitrary("XREVRANGE").Args(key, start, stop).Build())
	if resp.Error() != nil {
		return nil, resp.Error()
	}
	xrangeEntries, err := resp.AsXRange()
	if err != nil {
		return nil, err
	}
	entries := make([]kvx.StreamEntry, len(xrangeEntries))
	for i, entry := range xrangeEntries {
		values := make(map[string][]byte)
		for f, v := range entry.FieldValues {
			values[f] = []byte(v)
		}
		entries[i] = kvx.StreamEntry{ID: entry.ID, Values: values}
	}
	return entries, nil
}

// XDel deletes specific entries from a stream.
func (a *Adapter) XDel(ctx context.Context, key string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	args := make([]string, 0, len(ids)+1)
	args = append(args, key)
	args = append(args, ids...)
	return a.client.Do(ctx, a.client.B().Arbitrary("XDEL").Args(args...).Build()).Error()
}

// XGroupCreate creates a consumer group.
func (a *Adapter) XGroupCreate(ctx context.Context, key string, group string, startID string) error {
	return a.client.Do(ctx, a.client.B().Arbitrary("XGROUP", "CREATE").Args(key, group, startID).Build()).Error()
}

// XGroupDestroy destroys a consumer group.
func (a *Adapter) XGroupDestroy(ctx context.Context, key string, group string) error {
	return a.client.Do(ctx, a.client.B().Arbitrary("XGROUP", "DESTROY").Args(key, group).Build()).Error()
}

// XGroupCreateConsumer creates a consumer in a group.
func (a *Adapter) XGroupCreateConsumer(ctx context.Context, key string, group string, consumer string) error {
	return a.client.Do(ctx, a.client.B().Arbitrary("XGROUP", "CREATECONSUMER").Args(key, group, consumer).Build()).Error()
}

// XGroupDelConsumer deletes a consumer from a group.
func (a *Adapter) XGroupDelConsumer(ctx context.Context, key string, group string, consumer string) error {
	return a.client.Do(ctx, a.client.B().Arbitrary("XGROUP", "DELCONSUMER").Args(key, group, consumer).Build()).Error()
}

// XReadGroup reads entries as part of a consumer group.
func (a *Adapter) XReadGroup(ctx context.Context, group string, consumer string, streams map[string]string, count int64, block time.Duration) (map[string][]kvx.StreamEntry, error) {
	if len(streams) == 0 {
		return map[string][]kvx.StreamEntry{}, nil
	}
	args := []string{"GROUP", group, consumer}
	if count > 0 {
		args = append(args, "COUNT", strconv.FormatInt(count, 10))
	}
	if block > 0 {
		args = append(args, "BLOCK", strconv.FormatInt(block.Milliseconds(), 10))
	}
	args = append(args, "STREAMS")
	for key := range streams {
		args = append(args, key)
	}
	for _, start := range streams {
		args = append(args, start)
	}
	resp := a.client.Do(ctx, a.client.B().Arbitrary("XREADGROUP").Args(args...).Build())
	if resp.Error() != nil {
		if valkey.IsValkeyNil(resp.Error()) {
			return map[string][]kvx.StreamEntry{}, nil
		}
		return nil, resp.Error()
	}
	xreadResult, err := resp.AsXRead()
	if err != nil {
		return nil, err
	}
	out := make(map[string][]kvx.StreamEntry, len(xreadResult))
	for streamKey, streamEntries := range xreadResult {
		items := make([]kvx.StreamEntry, 0, len(streamEntries))
		for _, entry := range streamEntries {
			values := make(map[string][]byte)
			for f, v := range entry.FieldValues {
				values[f] = []byte(v)
			}
			items = append(items, kvx.StreamEntry{ID: entry.ID, Values: values})
		}
		out[streamKey] = items
	}
	return out, nil
}

// XAck acknowledges processing of stream entries.
func (a *Adapter) XAck(ctx context.Context, key string, group string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	args := []string{key, group}
	args = append(args, ids...)
	return a.client.Do(ctx, a.client.B().Arbitrary("XACK").Args(args...).Build()).Error()
}

// XPending gets pending entries information.
func (a *Adapter) XPending(ctx context.Context, key string, group string) (*kvx.PendingInfo, error) {
	return nil, errors.New("kvx/valkey: XPending not implemented")
}

// XPendingRange gets pending entries in a range.
func (a *Adapter) XPendingRange(ctx context.Context, key string, group string, start string, stop string, count int64) ([]kvx.PendingEntry, error) {
	return nil, errors.New("kvx/valkey: XPendingRange not implemented")
}

// XClaim claims pending entries for a consumer.
func (a *Adapter) XClaim(ctx context.Context, key string, group string, consumer string, minIdleTime time.Duration, ids []string) ([]kvx.StreamEntry, error) {
	return nil, errors.New("kvx/valkey: XClaim not implemented")
}

// XAutoClaim auto-claims pending entries.
func (a *Adapter) XAutoClaim(ctx context.Context, key string, group string, consumer string, minIdleTime time.Duration, start string, count int64) (string, []kvx.StreamEntry, error) {
	return "", nil, errors.New("kvx/valkey: XAutoClaim not implemented")
}

// XInfoGroups gets info about consumer groups.
func (a *Adapter) XInfoGroups(ctx context.Context, key string) ([]kvx.GroupInfo, error) {
	return nil, errors.New("kvx/valkey: XInfoGroups not implemented")
}

// XInfoConsumers gets info about consumers in a group.
func (a *Adapter) XInfoConsumers(ctx context.Context, key string, group string) ([]kvx.ConsumerInfo, error) {
	return nil, errors.New("kvx/valkey: XInfoConsumers not implemented")
}

// XInfoStream gets info about a stream.
func (a *Adapter) XInfoStream(ctx context.Context, key string) (*kvx.StreamInfo, error) {
	return nil, errors.New("kvx/valkey: XInfoStream not implemented")
}
