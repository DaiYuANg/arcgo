// Package stream provides Stream functionality.
package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/samber/lo"
)

// Stream provides high-level stream operations.
type Stream struct {
	client kvx.Stream
}

// NewStream creates a new Stream instance.
func NewStream(client kvx.Stream) *Stream {
	return &Stream{client: client}
}

// Add adds an entry to the stream.
func (s *Stream) Add(ctx context.Context, streamKey string, values map[string]any) (string, error) {
	byteValues, err := buildByteValues(values)
	if err != nil {
		return "", fmt.Errorf("serialize stream values for %s: %w", streamKey, err)
	}

	id, addErr := s.client.XAdd(ctx, streamKey, "*", byteValues)
	return wrapResult(id, addErr, "add stream entry to "+streamKey)
}

// AddWithID adds an entry with a specific ID to the stream.
func (s *Stream) AddWithID(ctx context.Context, streamKey, id string, values map[string]any) (string, error) {
	byteValues, err := buildByteValues(values)
	if err != nil {
		return "", fmt.Errorf("serialize stream values for %s: %w", streamKey, err)
	}

	entryID, addErr := s.client.XAdd(ctx, streamKey, id, byteValues)
	return wrapResult(entryID, addErr, "add stream entry with ID to "+streamKey)
}

// AddEvent adds a typed event to the stream.
func (s *Stream) AddEvent(ctx context.Context, streamKey, eventType string, payload any) (string, error) {
	data, err := marshalJSON(payload, "marshal stream event payload")
	if err != nil {
		return "", err
	}

	values := map[string]any{
		"type":      eventType,
		"payload":   string(data),
		"timestamp": time.Now().UnixMilli(),
	}

	return s.Add(ctx, streamKey, values)
}

// Read reads entries from the stream.
func (s *Stream) Read(ctx context.Context, streamKey, start string, count int64) (collectionx.List[kvx.StreamEntry], error) {
	entries, err := s.client.XRead(ctx, streamKey, start, count)
	return wrapResult(entries, err, "read stream entries from "+streamKey)
}

// ReadMultiple reads entries from multiple streams.
func (s *Stream) ReadMultiple(ctx context.Context, streams map[string]string, count int64, block time.Duration) (collectionx.MultiMap[string, kvx.StreamEntry], error) {
	entries, err := s.client.XReadMultiple(ctx, streams, count, block)
	return wrapResult(entries, err, "read multiple streams")
}

// ReadLast reads the last N entries from the stream.
func (s *Stream) ReadLast(ctx context.Context, streamKey string, count int64) (collectionx.List[kvx.StreamEntry], error) {
	entries, rangeErr := s.client.XRevRange(ctx, streamKey, "+", "-")
	entries, err := wrapResult(entries, rangeErr, "read latest stream entries from "+streamKey)
	if err != nil {
		return nil, err
	}

	return limitEntries(entries, count), nil
}

// ReadSince reads entries since a specific ID.
func (s *Stream) ReadSince(ctx context.Context, streamKey, sinceID string, count int64) (collectionx.List[kvx.StreamEntry], error) {
	entries, err := s.client.XRead(ctx, streamKey, sinceID, count)
	return wrapResult(entries, err, "read stream entries since ID from "+streamKey)
}

// Range reads entries in a range.
func (s *Stream) Range(ctx context.Context, streamKey, start, stop string) (collectionx.List[kvx.StreamEntry], error) {
	entries, err := s.client.XRange(ctx, streamKey, start, stop)
	return wrapResult(entries, err, "read stream range from "+streamKey)
}

// RevRange reads entries in reverse order.
func (s *Stream) RevRange(ctx context.Context, streamKey, start, stop string) (collectionx.List[kvx.StreamEntry], error) {
	entries, err := s.client.XRevRange(ctx, streamKey, start, stop)
	return wrapResult(entries, err, "read stream reverse range from "+streamKey)
}

// Len returns the number of entries in the stream.
func (s *Stream) Len(ctx context.Context, streamKey string) (int64, error) {
	length, err := s.client.XLen(ctx, streamKey)
	return wrapResult(length, err, "read stream length for "+streamKey)
}

// Trim trims the stream to approximately maxLen entries.
func (s *Stream) Trim(ctx context.Context, streamKey string, maxLen int64) error {
	return wrapError(s.client.XTrim(ctx, streamKey, maxLen), "trim stream "+streamKey)
}

// TrimApprox trims the stream to approximately maxLen entries (more efficient).
func (s *Stream) TrimApprox(ctx context.Context, streamKey string, maxLen int64) error {
	return wrapError(s.client.XTrim(ctx, streamKey, maxLen), "trim stream approximately "+streamKey)
}

// Delete deletes specific entries from the stream.
func (s *Stream) Delete(ctx context.Context, streamKey string, ids []string) error {
	return wrapError(s.client.XDel(ctx, streamKey, ids), "delete stream entries from "+streamKey)
}

// Info gets information about the stream.
func (s *Stream) Info(ctx context.Context, streamKey string) (*kvx.StreamInfo, error) {
	info, err := s.client.XInfoStream(ctx, streamKey)
	return wrapResult(info, err, "read stream info for "+streamKey)
}

// ConsumerGroup creates a ConsumerGroup instance for this stream.
func (s *Stream) ConsumerGroup(streamKey, groupName, consumerName string) *ConsumerGroup {
	return NewConsumerGroup(s.client, streamKey, groupName, consumerName)
}

// ConsumerGroupManager creates a ConsumerGroupManager for this stream.
func (s *Stream) ConsumerGroupManager(streamKey string) *ConsumerGroupManager {
	return NewConsumerGroupManager(s.client, streamKey)
}

// EventStream provides typed event streaming.
type EventStream[T any] struct {
	stream    *Stream
	streamKey string
}

// NewEventStream creates a new EventStream.
func NewEventStream[T any](client kvx.Stream, streamKey string) *EventStream[T] {
	return &EventStream[T]{
		stream:    NewStream(client),
		streamKey: streamKey,
	}
}

// Publish publishes an event to the stream.
func (e *EventStream[T]) Publish(ctx context.Context, event T) (string, error) {
	data, err := marshalJSON(event, "marshal typed stream event")
	if err != nil {
		return "", err
	}

	values := map[string]any{
		"data": string(data),
	}

	return e.stream.Add(ctx, e.streamKey, values)
}

// Subscribe subscribes to events from the stream.
func (e *EventStream[T]) Subscribe(ctx context.Context, start string, count int64) (collectionx.List[T], string, error) {
	entries, err := e.stream.Read(ctx, e.streamKey, start, count)
	if err != nil {
		return nil, "", err
	}

	decoded := collectionx.FilterMapList(entries, func(_ int, entry kvx.StreamEntry) (lo.Entry[string, T], bool) {
		data, ok := entry.Values.Get("data")
		if !ok {
			return lo.Entry[string, T]{}, false
		}
		var event T
		if err := json.Unmarshal(data, &event); err != nil {
			return lo.Entry[string, T]{}, false
		}
		return lo.Entry[string, T]{Key: entry.ID, Value: event}, true
	})
	if decoded.IsEmpty() {
		return collectionx.NewList[T](), "", nil
	}

	last, _ := decoded.GetLast()
	return collectionx.MapList(decoded, func(_ int, item lo.Entry[string, T]) T {
		return item.Value
	}), last.Key, nil
}

// EventConsumer consumes typed events from a stream.
type EventConsumer[T any] struct {
	consumer *Consumer
}

// NewEventConsumer creates a new EventConsumer.
func NewEventConsumer[T any](group *ConsumerGroup, handler func(ctx context.Context, event T) error, opts ConsumerOptions) *EventConsumer[T] {
	messageHandler := func(ctx context.Context, entry kvx.StreamEntry) error {
		if data, ok := entry.Values.Get("data"); ok {
			var event T
			if err := json.Unmarshal(data, &event); err != nil {
				return fmt.Errorf("unmarshal stream event: %w", err)
			}
			return handler(ctx, event)
		}
		return nil
	}

	return &EventConsumer[T]{
		consumer: NewConsumer(group, messageHandler, opts),
	}
}

// Run starts the event consumer.
func (e *EventConsumer[T]) Run(ctx context.Context) error {
	return e.consumer.Run(ctx)
}
