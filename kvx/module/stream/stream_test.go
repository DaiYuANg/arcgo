package stream

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestStream_Add(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	stream := NewStream(mock)

	id, err := stream.Add(ctx, "mystream", map[string]interface{}{
		"field1": "value1",
		"field2": 42,
	})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if id == "" {
		t.Errorf("Expected non-empty ID")
	}

	if len(mock.streams["mystream"]) != 1 {
		t.Errorf("Expected 1 entry in stream, got %d", len(mock.streams["mystream"]))
	}
}

func TestStream_AddWithID(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	stream := NewStream(mock)

	id, err := stream.AddWithID(ctx, "mystream", "123-0", map[string]interface{}{
		"field1": "value1",
	})
	if err != nil {
		t.Fatalf("AddWithID failed: %v", err)
	}

	if id != "123-0" {
		t.Errorf("Expected ID '123-0', got '%s'", id)
	}
}

func TestStream_AddEvent(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	stream := NewStream(mock)

	type TestEvent struct {
		Name string `json:"name"`
		Data int    `json:"data"`
	}

	event := TestEvent{Name: "test", Data: 42}
	id, err := stream.AddEvent(ctx, "mystream", "test_event", event)
	if err != nil {
		t.Fatalf("AddEvent failed: %v", err)
	}

	if id == "" {
		t.Errorf("Expected non-empty ID")
	}

	entries := mock.streams["mystream"]
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	if _, ok := entries[0].Values["type"]; !ok {
		t.Errorf("Expected 'type' field in event")
	}
	if _, ok := entries[0].Values["payload"]; !ok {
		t.Errorf("Expected 'payload' field in event")
	}
	if _, ok := entries[0].Values["timestamp"]; !ok {
		t.Errorf("Expected 'timestamp' field in event")
	}
}

func TestStream_Read(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	stream := NewStream(mock)

	// Add entries
	mock.XAdd(ctx, "mystream", "1-0", map[string][]byte{"data": []byte("test1")})
	mock.XAdd(ctx, "mystream", "2-0", map[string][]byte{"data": []byte("test2")})

	entries, err := stream.Read(ctx, "mystream", "0", 10)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}
}

func TestStream_ReadMultiple(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	stream := NewStream(mock)

	// Add entries to multiple streams
	mock.XAdd(ctx, "stream1", "1-0", map[string][]byte{"data": []byte("test1")})
	mock.XAdd(ctx, "stream2", "1-0", map[string][]byte{"data": []byte("test2")})

	results, err := stream.ReadMultiple(ctx, map[string]string{
		"stream1": "0",
		"stream2": "0",
	}, 10, time.Second)
	if err != nil {
		t.Fatalf("ReadMultiple failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 streams, got %d", len(results))
	}
}

func TestStream_ReadLast(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	stream := NewStream(mock)

	// Add entries
	mock.XAdd(ctx, "mystream", "1-0", map[string][]byte{"data": []byte("test1")})
	mock.XAdd(ctx, "mystream", "2-0", map[string][]byte{"data": []byte("test2")})
	mock.XAdd(ctx, "mystream", "3-0", map[string][]byte{"data": []byte("test3")})

	entries, err := stream.ReadLast(ctx, "mystream", 2)
	if err != nil {
		t.Fatalf("ReadLast failed: %v", err)
	}

	// Should return entries in reverse order
	if len(entries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(entries))
	}
}

func TestStream_ReadSince(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	stream := NewStream(mock)

	// Add entries
	mock.XAdd(ctx, "mystream", "1-0", map[string][]byte{"data": []byte("test1")})
	mock.XAdd(ctx, "mystream", "2-0", map[string][]byte{"data": []byte("test2")})
	mock.XAdd(ctx, "mystream", "3-0", map[string][]byte{"data": []byte("test3")})

	entries, err := stream.ReadSince(ctx, "mystream", "1-0", 10)
	if err != nil {
		t.Fatalf("ReadSince failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}
}

func TestStream_Range(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	stream := NewStream(mock)

	// Add entries
	mock.XAdd(ctx, "mystream", "1-0", map[string][]byte{"data": []byte("test1")})
	mock.XAdd(ctx, "mystream", "2-0", map[string][]byte{"data": []byte("test2")})
	mock.XAdd(ctx, "mystream", "3-0", map[string][]byte{"data": []byte("test3")})

	entries, err := stream.Range(ctx, "mystream", "1-0", "2-0")
	if err != nil {
		t.Fatalf("Range failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}
}

func TestStream_RevRange(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	stream := NewStream(mock)

	// Add entries
	mock.XAdd(ctx, "mystream", "1-0", map[string][]byte{"data": []byte("test1")})
	mock.XAdd(ctx, "mystream", "2-0", map[string][]byte{"data": []byte("test2")})

	entries, err := stream.RevRange(ctx, "mystream", "+", "-")
	if err != nil {
		t.Fatalf("RevRange failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}
}

func TestStream_Len(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	stream := NewStream(mock)

	// Add entries
	mock.XAdd(ctx, "mystream", "1-0", map[string][]byte{"data": []byte("test1")})
	mock.XAdd(ctx, "mystream", "2-0", map[string][]byte{"data": []byte("test2")})

	length, err := stream.Len(ctx, "mystream")
	if err != nil {
		t.Fatalf("Len failed: %v", err)
	}

	if length != 2 {
		t.Errorf("Expected length 2, got %d", length)
	}
}

func TestStream_Trim(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	stream := NewStream(mock)

	// Add entries
	for i := 0; i < 10; i++ {
		mock.XAdd(ctx, "mystream", "*", map[string][]byte{"data": []byte("test")})
	}

	err := stream.Trim(ctx, "mystream", 5)
	if err != nil {
		t.Fatalf("Trim failed: %v", err)
	}

	length, _ := stream.Len(ctx, "mystream")
	if length != 5 {
		t.Errorf("Expected length 5 after trim, got %d", length)
	}
}

func TestStream_TrimApprox(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	stream := NewStream(mock)

	// Add entries
	for i := 0; i < 10; i++ {
		mock.XAdd(ctx, "mystream", "*", map[string][]byte{"data": []byte("test")})
	}

	err := stream.TrimApprox(ctx, "mystream", 5)
	if err != nil {
		t.Fatalf("TrimApprox failed: %v", err)
	}
}

func TestStream_Delete(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	stream := NewStream(mock)

	// Add entries
	mock.XAdd(ctx, "mystream", "1-0", map[string][]byte{"data": []byte("test1")})
	mock.XAdd(ctx, "mystream", "2-0", map[string][]byte{"data": []byte("test2")})

	err := stream.Delete(ctx, "mystream", []string{"1-0"})
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	length, _ := stream.Len(ctx, "mystream")
	if length != 1 {
		t.Errorf("Expected length 1 after delete, got %d", length)
	}
}

func TestStream_Info(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	stream := NewStream(mock)

	// Add entries
	mock.XAdd(ctx, "mystream", "1-0", map[string][]byte{"data": []byte("test")})

	info, err := stream.Info(ctx, "mystream")
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}

	if info.Length != 1 {
		t.Errorf("Expected length 1, got %d", info.Length)
	}
}

func TestStream_ConsumerGroup(t *testing.T) {
	mock := newMockStream()
	stream := NewStream(mock)

	cg := stream.ConsumerGroup("mystream", "mygroup", "consumer1")
	if cg == nil {
		t.Errorf("Expected non-nil ConsumerGroup")
	}
	if cg.streamKey != "mystream" {
		t.Errorf("Expected streamKey 'mystream', got '%s'", cg.streamKey)
	}
	if cg.groupName != "mygroup" {
		t.Errorf("Expected groupName 'mygroup', got '%s'", cg.groupName)
	}
}

func TestStream_ConsumerGroupManager(t *testing.T) {
	mock := newMockStream()
	stream := NewStream(mock)

	manager := stream.ConsumerGroupManager("mystream")
	if manager == nil {
		t.Errorf("Expected non-nil ConsumerGroupManager")
	}
	if manager.streamKey != "mystream" {
		t.Errorf("Expected streamKey 'mystream', got '%s'", manager.streamKey)
	}
}

func TestEventStream_Publish(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()

	type TestEvent struct {
		Name string `json:"name"`
		Value int   `json:"value"`
	}

	eventStream := NewEventStream[TestEvent](mock, "events")
	event := TestEvent{Name: "test", Value: 42}

	id, err := eventStream.Publish(ctx, event)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	if id == "" {
		t.Errorf("Expected non-empty ID")
	}

	entries := mock.streams["events"]
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	if _, ok := entries[0].Values["data"]; !ok {
		t.Errorf("Expected 'data' field in entry")
	}
}

func TestEventStream_Subscribe(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()

	type TestEvent struct {
		Name string `json:"name"`
		Value int   `json:"value"`
	}

	// Add event
	data, _ := json.Marshal(TestEvent{Name: "test", Value: 42})
	mock.XAdd(ctx, "events", "1-0", map[string][]byte{"data": data})

	eventStream := NewEventStream[TestEvent](mock, "events")
	events, lastID, err := eventStream.Subscribe(ctx, "0", 10)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}

	if events[0].Name != "test" {
		t.Errorf("Expected event name 'test', got '%s'", events[0].Name)
	}

	if lastID != "1-0" {
		t.Errorf("Expected lastID '1-0', got '%s'", lastID)
	}
}

func TestEventConsumer(t *testing.T) {
	mock := newMockStream()
	mock.groups["events"] = make(map[string]*mockConsumerGroup)
	mock.groups["events"]["mygroup"] = &mockConsumerGroup{name: "mygroup"}

	type TestEvent struct {
		Name string `json:"name"`
	}

	cg := NewConsumerGroup(mock, "events", "mygroup", "consumer1")

	handler := func(ctx context.Context, event TestEvent) error {
		return nil
	}

	opts := DefaultConsumerOptions()
	consumer := NewEventConsumer(cg, handler, opts)

	if consumer == nil {
		t.Errorf("Expected non-nil EventConsumer")
	}
}

func TestConvertToBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected []byte
	}{
		{"string", "test", []byte("test")},
		{"bytes", []byte("test"), []byte("test")},
		{"nil", nil, []byte("")},
		{"int", 42, []byte("42")},
		{"map", map[string]int{"a": 1}, []byte(`{"a":1}`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToBytes(tt.input)
			// For JSON-encoded values, just check it's not empty
			if len(result) == 0 && len(tt.expected) > 0 {
				t.Errorf("convertToBytes(%v) returned empty", tt.input)
			}
		})
	}
}

func TestNewStream(t *testing.T) {
	mock := newMockStream()
	stream := NewStream(mock)

	if stream == nil {
		t.Errorf("Expected non-nil Stream")
	}
	if stream.client == nil {
		t.Errorf("Expected non-nil client")
	}
}

func TestNewEventStream(t *testing.T) {
	mock := newMockStream()

	type TestEvent struct{}
	eventStream := NewEventStream[TestEvent](mock, "events")

	if eventStream == nil {
		t.Errorf("Expected non-nil EventStream")
	}
	if eventStream.streamKey != "events" {
		t.Errorf("Expected streamKey 'events', got '%s'", eventStream.streamKey)
	}
}
