// Package stream provides Stream functionality.
package stream

import (
	"context"
	"encoding/json"
	"time"

	"github.com/DaiYuANg/archgo/kvx"
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
func (s *Stream) Add(ctx context.Context, streamKey string, values map[string]interface{}) (string, error) {
	byteValues := make(map[string][]byte, len(values))
	for k, v := range values {
		byteValues[k] = convertToBytes(v)
	}

	return s.client.XAdd(ctx, streamKey, "*", byteValues)
}

// AddWithID adds an entry with a specific ID to the stream.
func (s *Stream) AddWithID(ctx context.Context, streamKey string, id string, values map[string]interface{}) (string, error) {
	byteValues := make(map[string][]byte, len(values))
	for k, v := range values {
		byteValues[k] = convertToBytes(v)
	}

	return s.client.XAdd(ctx, streamKey, id, byteValues)
}

// AddEvent adds a typed event to the stream.
func (s *Stream) AddEvent(ctx context.Context, streamKey string, eventType string, payload interface{}) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	values := map[string]interface{}{
		"type":      eventType,
		"payload":   string(data),
		"timestamp": time.Now().UnixMilli(),
	}

	return s.Add(ctx, streamKey, values)
}

// Read reads entries from the stream.
func (s *Stream) Read(ctx context.Context, streamKey string, start string, count int64) ([]kvx.StreamEntry, error) {
	return s.client.XRead(ctx, streamKey, start, count)
}

// ReadMultiple reads entries from multiple streams.
func (s *Stream) ReadMultiple(ctx context.Context, streams map[string]string, count int64, block time.Duration) (map[string][]kvx.StreamEntry, error) {
	return s.client.XReadMultiple(ctx, streams, count, block)
}

// ReadLast reads the last N entries from the stream.
func (s *Stream) ReadLast(ctx context.Context, streamKey string, count int64) ([]kvx.StreamEntry, error) {
	return s.client.XRevRange(ctx, streamKey, "+", "-")
}

// ReadSince reads entries since a specific ID.
func (s *Stream) ReadSince(ctx context.Context, streamKey string, sinceID string, count int64) ([]kvx.StreamEntry, error) {
	return s.client.XRead(ctx, streamKey, sinceID, count)
}

// Range reads entries in a range.
func (s *Stream) Range(ctx context.Context, streamKey string, start, stop string) ([]kvx.StreamEntry, error) {
	return s.client.XRange(ctx, streamKey, start, stop)
}

// RevRange reads entries in reverse order.
func (s *Stream) RevRange(ctx context.Context, streamKey string, start, stop string) ([]kvx.StreamEntry, error) {
	return s.client.XRevRange(ctx, streamKey, start, stop)
}

// Len returns the number of entries in the stream.
func (s *Stream) Len(ctx context.Context, streamKey string) (int64, error) {
	return s.client.XLen(ctx, streamKey)
}

// Trim trims the stream to approximately maxLen entries.
func (s *Stream) Trim(ctx context.Context, streamKey string, maxLen int64) error {
	return s.client.XTrim(ctx, streamKey, maxLen)
}

// TrimApprox trims the stream to approximately maxLen entries (more efficient).
func (s *Stream) TrimApprox(ctx context.Context, streamKey string, maxLen int64) error {
	return s.client.XTrim(ctx, streamKey, maxLen)
}

// Delete deletes specific entries from the stream.
func (s *Stream) Delete(ctx context.Context, streamKey string, ids []string) error {
	return s.client.XDel(ctx, streamKey, ids)
}

// Info gets information about the stream.
func (s *Stream) Info(ctx context.Context, streamKey string) (*kvx.StreamInfo, error) {
	return s.client.XInfoStream(ctx, streamKey)
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
	data, err := json.Marshal(event)
	if err != nil {
		return "", err
	}

	values := map[string]interface{}{
		"data": string(data),
	}

	return e.stream.Add(ctx, e.streamKey, values)
}

// Subscribe subscribes to events from the stream.
func (e *EventStream[T]) Subscribe(ctx context.Context, start string, count int64) ([]T, string, error) {
	entries, err := e.stream.Read(ctx, e.streamKey, start, count)
	if err != nil {
		return nil, "", err
	}

	var events []T
	var lastID string

	for _, entry := range entries {
		if data, ok := entry.Values["data"]; ok {
			var event T
			if err := json.Unmarshal(data, &event); err != nil {
				continue
			}
			events = append(events, event)
			lastID = entry.ID
		}
	}

	return events, lastID, nil
}

// EventConsumer consumes typed events from a stream.
type EventConsumer[T any] struct {
	consumer *Consumer
}

// NewEventConsumer creates a new EventConsumer.
func NewEventConsumer[T any](group *ConsumerGroup, handler func(ctx context.Context, event T) error, opts ConsumerOptions) *EventConsumer[T] {
	messageHandler := func(ctx context.Context, entry kvx.StreamEntry) error {
		if data, ok := entry.Values["data"]; ok {
			var event T
			if err := json.Unmarshal(data, &event); err != nil {
				return err
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

// Helper functions
func convertToBytes(v interface{}) []byte {
	switch val := v.(type) {
	case []byte:
		return val
	case string:
		return []byte(val)
	case nil:
		return []byte("")
	default:
		data, _ := json.Marshal(v)
		return data
	}
}
