// Package stream provides Stream functionality.
package stream

import (
	"context"

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
	// Convert interface{} values to []byte
	byteValues := make(map[string][]byte, len(values))
	for k, v := range values {
		switch val := v.(type) {
		case []byte:
			byteValues[k] = val
		case string:
			byteValues[k] = []byte(val)
		case nil:
			byteValues[k] = []byte("")
		default:
			// Default to string conversion
			byteValues[k] = []byte(string(val.(string)))
		}
	}

	return s.client.XAdd(ctx, streamKey, "*", byteValues)
}

// AddWithID adds an entry with a specific ID to the stream.
func (s *Stream) AddWithID(ctx context.Context, streamKey string, id string, values map[string]interface{}) (string, error) {
	byteValues := make(map[string][]byte, len(values))
	for k, v := range values {
		switch val := v.(type) {
		case []byte:
			byteValues[k] = val
		case string:
			byteValues[k] = []byte(val)
		case nil:
			byteValues[k] = []byte("")
		default:
			byteValues[k] = []byte(string(val.(string)))
		}
	}

	return s.client.XAdd(ctx, streamKey, id, byteValues)
}

// Read reads entries from the stream.
func (s *Stream) Read(ctx context.Context, streamKey string, start string, count int64) ([]kvx.StreamEntry, error) {
	return s.client.XRead(ctx, streamKey, start, count)
}

// ReadLast reads the last N entries from the stream.
func (s *Stream) ReadLast(ctx context.Context, streamKey string, count int64) ([]kvx.StreamEntry, error) {
	return s.client.XRead(ctx, streamKey, "-", count)
}

// Range reads entries in a range.
func (s *Stream) Range(ctx context.Context, streamKey string, start, stop string) ([]kvx.StreamEntry, error) {
	return s.client.XRange(ctx, streamKey, start, stop)
}

// Len returns the number of entries in the stream.
func (s *Stream) Len(ctx context.Context, streamKey string) (int64, error) {
	return s.client.XLen(ctx, streamKey)
}

// Trim trims the stream to approximately maxLen entries.
func (s *Stream) Trim(ctx context.Context, streamKey string, maxLen int64) error {
	return s.client.XTrim(ctx, streamKey, maxLen)
}

// Consumer provides stream consumer functionality.
type Consumer struct {
	stream       *Stream
	groupName    string
	consumerName string
}

// NewConsumer creates a new Consumer.
func NewConsumer(stream *Stream, groupName string, consumerName string) *Consumer {
	return &Consumer{
		stream:       stream,
		groupName:    groupName,
		consumerName: consumerName,
	}
}

// TODO: Add consumer group support (XGROUP, XREADGROUP, XACK, etc.)
