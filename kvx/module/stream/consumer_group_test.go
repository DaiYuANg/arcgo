package stream

import (
	"context"
	"testing"
	"time"

	"github.com/DaiYuANg/archgo/kvx"
)

// Mock Stream implementation
type mockStream struct {
	streams       map[string][]kvx.StreamEntry
	groups        map[string]map[string]*mockConsumerGroup
	consumers     map[string]map[string]map[string]*mockConsumer
	pending       map[string]map[string][]kvx.PendingEntry
	lastID        map[string]string
}

type mockConsumerGroup struct {
	name            string
	consumers       int64
	pending         int64
	lastDeliveredID string
}

type mockConsumer struct {
	name    string
	pending int64
	idle    time.Duration
}

func newMockStream() *mockStream {
	return &mockStream{
		streams:   make(map[string][]kvx.StreamEntry),
		groups:    make(map[string]map[string]*mockConsumerGroup),
		consumers: make(map[string]map[string]map[string]*mockConsumer),
		pending:   make(map[string]map[string][]kvx.PendingEntry),
		lastID:    make(map[string]string),
	}
}

func (m *mockStream) XAdd(ctx context.Context, key string, id string, values map[string][]byte) (string, error) {
	if _, ok := m.streams[key]; !ok {
		m.streams[key] = []kvx.StreamEntry{}
	}

	actualID := id
	if id == "*" {
		actualID = time.Now().Format("20060102150405") + "-0"
	}

	entry := kvx.StreamEntry{
		ID:     actualID,
		Values: values,
	}
	m.streams[key] = append(m.streams[key], entry)
	m.lastID[key] = actualID
	return actualID, nil
}

func (m *mockStream) XRead(ctx context.Context, key string, start string, count int64) ([]kvx.StreamEntry, error) {
	entries, ok := m.streams[key]
	if !ok {
		return []kvx.StreamEntry{}, nil
	}

	var result []kvx.StreamEntry
	found := start == "0"
	for _, entry := range entries {
		if found {
			result = append(result, entry)
			if int64(len(result)) >= count {
				break
			}
		}
		if entry.ID == start {
			found = true
		}
	}
	return result, nil
}

func (m *mockStream) XReadMultiple(ctx context.Context, streams map[string]string, count int64, block time.Duration) (map[string][]kvx.StreamEntry, error) {
	result := make(map[string][]kvx.StreamEntry)
	for key, start := range streams {
		entries, err := m.XRead(ctx, key, start, count)
		if err != nil {
			return nil, err
		}
		result[key] = entries
	}
	return result, nil
}

func (m *mockStream) XRange(ctx context.Context, key string, start, stop string) ([]kvx.StreamEntry, error) {
	entries, ok := m.streams[key]
	if !ok {
		return []kvx.StreamEntry{}, nil
	}

	var result []kvx.StreamEntry
	inRange := false
	for _, entry := range entries {
		if entry.ID == start || start == "-" {
			inRange = true
		}
		if inRange {
			result = append(result, entry)
		}
		if entry.ID == stop || stop == "+" {
			break
		}
	}
	return result, nil
}

func (m *mockStream) XRevRange(ctx context.Context, key string, start, stop string) ([]kvx.StreamEntry, error) {
	entries, ok := m.streams[key]
	if !ok {
		return []kvx.StreamEntry{}, nil
	}

	var result []kvx.StreamEntry
	for i := len(entries) - 1; i >= 0; i-- {
		result = append(result, entries[i])
	}
	return result, nil
}

func (m *mockStream) XLen(ctx context.Context, key string) (int64, error) {
	return int64(len(m.streams[key])), nil
}

func (m *mockStream) XTrim(ctx context.Context, key string, maxLen int64) error {
	if entries, ok := m.streams[key]; ok {
		if int64(len(entries)) > maxLen {
			m.streams[key] = entries[int64(len(entries))-maxLen:]
		}
	}
	return nil
}

func (m *mockStream) XDel(ctx context.Context, key string, ids []string) error {
	if entries, ok := m.streams[key]; ok {
		idSet := make(map[string]bool)
		for _, id := range ids {
			idSet[id] = true
		}
		var newEntries []kvx.StreamEntry
		for _, entry := range entries {
			if !idSet[entry.ID] {
				newEntries = append(newEntries, entry)
			}
		}
		m.streams[key] = newEntries
	}
	return nil
}

func (m *mockStream) XGroupCreate(ctx context.Context, key string, group string, startID string) error {
	if _, ok := m.groups[key]; !ok {
		m.groups[key] = make(map[string]*mockConsumerGroup)
	}
	m.groups[key][group] = &mockConsumerGroup{
		name:            group,
		lastDeliveredID: startID,
	}
	return nil
}

func (m *mockStream) XGroupDestroy(ctx context.Context, key string, group string) error {
	if groups, ok := m.groups[key]; ok {
		delete(groups, group)
	}
	return nil
}

func (m *mockStream) XGroupCreateConsumer(ctx context.Context, key string, group string, consumer string) error {
	if _, ok := m.consumers[key]; !ok {
		m.consumers[key] = make(map[string]map[string]*mockConsumer)
	}
	if _, ok := m.consumers[key][group]; !ok {
		m.consumers[key][group] = make(map[string]*mockConsumer)
	}
	m.consumers[key][group][consumer] = &mockConsumer{
		name: consumer,
	}
	return nil
}

func (m *mockStream) XGroupDelConsumer(ctx context.Context, key string, group string, consumer string) error {
	if consumers, ok := m.consumers[key]; ok {
		if groupConsumers, ok := consumers[group]; ok {
			delete(groupConsumers, consumer)
		}
	}
	return nil
}

func (m *mockStream) XReadGroup(ctx context.Context, group string, consumer string, streams map[string]string, count int64, block time.Duration) (map[string][]kvx.StreamEntry, error) {
	result := make(map[string][]kvx.StreamEntry)
	for key, start := range streams {
		var entries []kvx.StreamEntry
		if start == ">" {
			// New messages only
			entries, _ = m.XRead(ctx, key, m.lastID[key], count)
		} else {
			// Pending messages
			entries, _ = m.XRead(ctx, key, start, count)
		}
		result[key] = entries

		// Track pending
		if _, ok := m.pending[key]; !ok {
			m.pending[key] = make(map[string][]kvx.PendingEntry)
		}
		for _, entry := range entries {
			m.pending[key][group] = append(m.pending[key][group], kvx.PendingEntry{
				ID:       entry.ID,
				Consumer: consumer,
				IdleTime: 0,
			})
		}
	}
	return result, nil
}

func (m *mockStream) XAck(ctx context.Context, key string, group string, ids []string) error {
	if pending, ok := m.pending[key]; ok {
		if groupPending, ok := pending[group]; ok {
			idSet := make(map[string]bool)
			for _, id := range ids {
				idSet[id] = true
			}
			var newPending []kvx.PendingEntry
			for _, entry := range groupPending {
				if !idSet[entry.ID] {
					newPending = append(newPending, entry)
				}
			}
			m.pending[key][group] = newPending
		}
	}
	return nil
}

func (m *mockStream) XPending(ctx context.Context, key string, group string) (*kvx.PendingInfo, error) {
	info := &kvx.PendingInfo{
		Consumers: make(map[string]int64),
	}
	if pending, ok := m.pending[key]; ok {
		if groupPending, ok := pending[group]; ok {
			info.Count = int64(len(groupPending))
			for _, entry := range groupPending {
				info.Consumers[entry.Consumer]++
			}
		}
	}
	return info, nil
}

func (m *mockStream) XPendingRange(ctx context.Context, key string, group string, start string, stop string, count int64) ([]kvx.PendingEntry, error) {
	if pending, ok := m.pending[key]; ok {
		if groupPending, ok := pending[group]; ok {
			return groupPending, nil
		}
	}
	return []kvx.PendingEntry{}, nil
}

func (m *mockStream) XClaim(ctx context.Context, key string, group string, consumer string, minIdleTime time.Duration, ids []string) ([]kvx.StreamEntry, error) {
	var result []kvx.StreamEntry
	if entries, ok := m.streams[key]; ok {
		idSet := make(map[string]bool)
		for _, id := range ids {
			idSet[id] = true
		}
		for _, entry := range entries {
			if idSet[entry.ID] {
				result = append(result, entry)
			}
		}
	}
	return result, nil
}

func (m *mockStream) XAutoClaim(ctx context.Context, key string, group string, consumer string, minIdleTime time.Duration, start string, count int64) (string, []kvx.StreamEntry, error) {
	entries, _ := m.XClaim(ctx, key, group, consumer, minIdleTime, []string{})
	return "", entries, nil
}

func (m *mockStream) XInfoGroups(ctx context.Context, key string) ([]kvx.GroupInfo, error) {
	var result []kvx.GroupInfo
	if groups, ok := m.groups[key]; ok {
		for _, group := range groups {
			result = append(result, kvx.GroupInfo{
				Name:            group.name,
				Consumers:       group.consumers,
				Pending:         group.pending,
				LastDeliveredID: group.lastDeliveredID,
			})
		}
	}
	return result, nil
}

func (m *mockStream) XInfoConsumers(ctx context.Context, key string, group string) ([]kvx.ConsumerInfo, error) {
	var result []kvx.ConsumerInfo
	if consumers, ok := m.consumers[key]; ok {
		if groupConsumers, ok := consumers[group]; ok {
			for _, consumer := range groupConsumers {
				result = append(result, kvx.ConsumerInfo{
					Name:    consumer.name,
					Pending: consumer.pending,
					Idle:    consumer.idle,
				})
			}
		}
	}
	return result, nil
}

func (m *mockStream) XInfoStream(ctx context.Context, key string) (*kvx.StreamInfo, error) {
	return &kvx.StreamInfo{
		Length:          int64(len(m.streams[key])),
		LastGeneratedID: m.lastID[key],
	}, nil
}

func TestConsumerGroup_Create(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	cg := NewConsumerGroup(mock, "mystream", "mygroup", "consumer1")

	err := cg.Create(ctx, "0")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if _, ok := mock.groups["mystream"]["mygroup"]; !ok {
		t.Errorf("Consumer group not created")
	}
}

func TestConsumerGroup_CreateFromBeginning(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	cg := NewConsumerGroup(mock, "mystream", "mygroup", "consumer1")

	err := cg.CreateFromBeginning(ctx)
	if err != nil {
		t.Fatalf("CreateFromBeginning failed: %v", err)
	}

	group := mock.groups["mystream"]["mygroup"]
	if group.lastDeliveredID != "0" {
		t.Errorf("Expected start ID '0', got '%s'", group.lastDeliveredID)
	}
}

func TestConsumerGroup_CreateFromLatest(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	cg := NewConsumerGroup(mock, "mystream", "mygroup", "consumer1")

	err := cg.CreateFromLatest(ctx)
	if err != nil {
		t.Fatalf("CreateFromLatest failed: %v", err)
	}

	group := mock.groups["mystream"]["mygroup"]
	if group.lastDeliveredID != "$" {
		t.Errorf("Expected start ID '$', got '%s'", group.lastDeliveredID)
	}
}

func TestConsumerGroup_Destroy(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	mock.groups["mystream"] = make(map[string]*mockConsumerGroup)
	mock.groups["mystream"]["mygroup"] = &mockConsumerGroup{name: "mygroup"}

	cg := NewConsumerGroup(mock, "mystream", "mygroup", "consumer1")
	err := cg.Destroy(ctx)
	if err != nil {
		t.Fatalf("Destroy failed: %v", err)
	}

	if _, ok := mock.groups["mystream"]["mygroup"]; ok {
		t.Errorf("Consumer group not destroyed")
	}
}

func TestConsumerGroup_Read(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()

	// Add some entries
	mock.XAdd(ctx, "mystream", "*", map[string][]byte{"data": []byte("test1")})
	mock.XAdd(ctx, "mystream", "*", map[string][]byte{"data": []byte("test2")})

	mock.groups["mystream"] = make(map[string]*mockConsumerGroup)
	mock.groups["mystream"]["mygroup"] = &mockConsumerGroup{name: "mygroup"}

	cg := NewConsumerGroup(mock, "mystream", "mygroup", "consumer1")
	entries, err := cg.Read(ctx, 10, time.Second)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}
}

func TestConsumerGroup_Ack(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()

	// Add and read entry
	id, _ := mock.XAdd(ctx, "mystream", "*", map[string][]byte{"data": []byte("test")})
	mock.groups["mystream"] = make(map[string]*mockConsumerGroup)
	mock.groups["mystream"]["mygroup"] = &mockConsumerGroup{name: "mygroup"}
	mock.pending["mystream"] = make(map[string][]kvx.PendingEntry)
	mock.pending["mystream"]["mygroup"] = []kvx.PendingEntry{
		{ID: id, Consumer: "consumer1"},
	}

	cg := NewConsumerGroup(mock, "mystream", "mygroup", "consumer1")
	err := cg.Ack(ctx, []string{id})
	if err != nil {
		t.Fatalf("Ack failed: %v", err)
	}

	if len(mock.pending["mystream"]["mygroup"]) != 0 {
		t.Errorf("Expected pending to be empty after ack")
	}
}

func TestConsumerGroup_Pending(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()

	mock.pending["mystream"] = make(map[string][]kvx.PendingEntry)
	mock.pending["mystream"]["mygroup"] = []kvx.PendingEntry{
		{ID: "1", Consumer: "consumer1"},
		{ID: "2", Consumer: "consumer1"},
	}

	cg := NewConsumerGroup(mock, "mystream", "mygroup", "consumer1")
	info, err := cg.Pending(ctx)
	if err != nil {
		t.Fatalf("Pending failed: %v", err)
	}

	if info.Count != 2 {
		t.Errorf("Expected count 2, got %d", info.Count)
	}
}

func TestConsumerGroupManager_CreateGroup(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	manager := NewConsumerGroupManager(mock, "mystream")

	err := manager.CreateGroup(ctx, "group1", "0")
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	if _, ok := mock.groups["mystream"]["group1"]; !ok {
		t.Errorf("Group not created")
	}
}

func TestConsumerGroupManager_ListGroups(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	mock.groups["mystream"] = make(map[string]*mockConsumerGroup)
	mock.groups["mystream"]["group1"] = &mockConsumerGroup{name: "group1"}
	mock.groups["mystream"]["group2"] = &mockConsumerGroup{name: "group2"}

	manager := NewConsumerGroupManager(mock, "mystream")
	groups, err := manager.ListGroups(ctx)
	if err != nil {
		t.Fatalf("ListGroups failed: %v", err)
	}

	if len(groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(groups))
	}
}

func TestConsumerGroupManager_GetConsumer(t *testing.T) {
	mock := newMockStream()
	manager := NewConsumerGroupManager(mock, "mystream")
	consumer := manager.GetConsumer("group1", "consumer1")

	if consumer.streamKey != "mystream" {
		t.Errorf("Expected streamKey 'mystream', got '%s'", consumer.streamKey)
	}
	if consumer.groupName != "group1" {
		t.Errorf("Expected groupName 'group1', got '%s'", consumer.groupName)
	}
	if consumer.consumerName != "consumer1" {
		t.Errorf("Expected consumerName 'consumer1', got '%s'", consumer.consumerName)
	}
}

func TestConsumerGroup_Info(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	mock.groups["mystream"] = make(map[string]*mockConsumerGroup)
	mock.groups["mystream"]["mygroup"] = &mockConsumerGroup{
		name:            "mygroup",
		consumers:       5,
		pending:         10,
		lastDeliveredID: "123-0",
	}

	cg := NewConsumerGroup(mock, "mystream", "mygroup", "consumer1")
	info, err := cg.Info(ctx)
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}

	if info.Name != "mygroup" {
		t.Errorf("Expected name 'mygroup', got '%s'", info.Name)
	}
	if info.Consumers != 5 {
		t.Errorf("Expected 5 consumers, got %d", info.Consumers)
	}
	if info.Pending != 10 {
		t.Errorf("Expected 10 pending, got %d", info.Pending)
	}
}

func TestConsumerGroup_ConsumerInfo(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	mock.consumers["mystream"] = make(map[string]map[string]*mockConsumer)
	mock.consumers["mystream"]["mygroup"] = make(map[string]*mockConsumer)
	mock.consumers["mystream"]["mygroup"]["consumer1"] = &mockConsumer{
		name:    "consumer1",
		pending: 3,
	}

	cg := NewConsumerGroup(mock, "mystream", "mygroup", "consumer1")
	info, err := cg.ConsumerInfo(ctx)
	if err != nil {
		t.Fatalf("ConsumerInfo failed: %v", err)
	}

	if info.Name != "consumer1" {
		t.Errorf("Expected name 'consumer1', got '%s'", info.Name)
	}
	if info.Pending != 3 {
		t.Errorf("Expected 3 pending, got %d", info.Pending)
	}
}

func TestConsumerGroup_ConsumerInfo_NotFound(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	mock.consumers["mystream"] = make(map[string]map[string]*mockConsumer)
	mock.consumers["mystream"]["mygroup"] = make(map[string]*mockConsumer)

	cg := NewConsumerGroup(mock, "mystream", "mygroup", "nonexistent")
	_, err := cg.ConsumerInfo(ctx)
	if err == nil {
		t.Errorf("Expected error for non-existent consumer")
	}
}

func TestDefaultConsumerOptions(t *testing.T) {
	opts := DefaultConsumerOptions()

	if !opts.AutoAck {
		t.Errorf("Expected AutoAck to be true")
	}
	if opts.BatchSize != 10 {
		t.Errorf("Expected BatchSize to be 10, got %d", opts.BatchSize)
	}
	if opts.BlockTimeout != 5*time.Second {
		t.Errorf("Expected BlockTimeout to be 5s, got %v", opts.BlockTimeout)
	}
}

func TestConsumerGroup_StreamInfo(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	mock.streams["mystream"] = []kvx.StreamEntry{
		{ID: "1-0", Values: map[string][]byte{"data": []byte("test")}},
		{ID: "2-0", Values: map[string][]byte{"data": []byte("test")}},
	}
	mock.lastID["mystream"] = "2-0"

	cg := NewConsumerGroup(mock, "mystream", "mygroup", "consumer1")
	info, err := cg.StreamInfo(ctx)
	if err != nil {
		t.Fatalf("StreamInfo failed: %v", err)
	}

	if info.Length != 2 {
		t.Errorf("Expected length 2, got %d", info.Length)
	}
	if info.LastGeneratedID != "2-0" {
		t.Errorf("Expected last ID '2-0', got '%s'", info.LastGeneratedID)
	}
}

func TestConsumerGroup_DeleteConsumer(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	mock.consumers["mystream"] = make(map[string]map[string]*mockConsumer)
	mock.consumers["mystream"]["mygroup"] = make(map[string]*mockConsumer)
	mock.consumers["mystream"]["mygroup"]["consumer1"] = &mockConsumer{name: "consumer1"}

	cg := NewConsumerGroup(mock, "mystream", "mygroup", "consumer1")
	err := cg.DeleteConsumer(ctx)
	if err != nil {
		t.Fatalf("DeleteConsumer failed: %v", err)
	}

	if _, ok := mock.consumers["mystream"]["mygroup"]["consumer1"]; ok {
		t.Errorf("Consumer not deleted")
	}
}

func TestConsumerGroup_Claim(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()

	// Add entries
	mock.streams["mystream"] = []kvx.StreamEntry{
		{ID: "1-0", Values: map[string][]byte{"data": []byte("test1")}},
		{ID: "2-0", Values: map[string][]byte{"data": []byte("test2")}},
	}

	cg := NewConsumerGroup(mock, "mystream", "mygroup", "consumer1")
	entries, err := cg.Claim(ctx, []string{"1-0", "2-0"}, time.Minute)
	if err != nil {
		t.Fatalf("Claim failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}
}

func TestConsumerGroup_AutoClaim(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()

	// Add entries
	mock.streams["mystream"] = []kvx.StreamEntry{
		{ID: "1-0", Values: map[string][]byte{"data": []byte("test1")}},
	}

	cg := NewConsumerGroup(mock, "mystream", "mygroup", "consumer1")
	cursor, entries, err := cg.AutoClaim(ctx, time.Minute, 10)
	if err != nil {
		t.Fatalf("AutoClaim failed: %v", err)
	}

	_ = cursor
	_ = entries
}

func TestConsumerGroup_ReadPending(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()

	// Add entries
	mock.streams["mystream"] = []kvx.StreamEntry{
		{ID: "1-0", Values: map[string][]byte{"data": []byte("test1")}},
	}
	mock.groups["mystream"] = make(map[string]*mockConsumerGroup)
	mock.groups["mystream"]["mygroup"] = &mockConsumerGroup{name: "mygroup"}

	cg := NewConsumerGroup(mock, "mystream", "mygroup", "consumer1")
	entries, err := cg.ReadPending(ctx, 10)
	if err != nil {
		t.Fatalf("ReadPending failed: %v", err)
	}

	// Should return entries starting from "0"
	_ = entries
}

func TestConsumerGroup_PendingRange(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()

	mock.pending["mystream"] = make(map[string][]kvx.PendingEntry)
	mock.pending["mystream"]["mygroup"] = []kvx.PendingEntry{
		{ID: "1-0", Consumer: "consumer1"},
		{ID: "2-0", Consumer: "consumer1"},
	}

	cg := NewConsumerGroup(mock, "mystream", "mygroup", "consumer1")
	entries, err := cg.PendingRange(ctx, "-", "+", 10)
	if err != nil {
		t.Fatalf("PendingRange failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 pending entries, got %d", len(entries))
	}
}

func TestConsumerGroup_AckEntry(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()

	mock.pending["mystream"] = make(map[string][]kvx.PendingEntry)
	mock.pending["mystream"]["mygroup"] = []kvx.PendingEntry{
		{ID: "1-0", Consumer: "consumer1"},
	}

	cg := NewConsumerGroup(mock, "mystream", "mygroup", "consumer1")
	err := cg.AckEntry(ctx, "1-0")
	if err != nil {
		t.Fatalf("AckEntry failed: %v", err)
	}

	if len(mock.pending["mystream"]["mygroup"]) != 0 {
		t.Errorf("Expected pending to be empty after ack")
	}
}

func TestConsumerGroupManager_DestroyGroup(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	mock.groups["mystream"] = make(map[string]*mockConsumerGroup)
	mock.groups["mystream"]["group1"] = &mockConsumerGroup{name: "group1"}

	manager := NewConsumerGroupManager(mock, "mystream")
	err := manager.DestroyGroup(ctx, "group1")
	if err != nil {
		t.Fatalf("DestroyGroup failed: %v", err)
	}

	if _, ok := mock.groups["mystream"]["group1"]; ok {
		t.Errorf("Group not destroyed")
	}
}

func TestConsumerGroupManager_StreamInfo(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	mock.streams["mystream"] = []kvx.StreamEntry{
		{ID: "1-0", Values: map[string][]byte{"data": []byte("test")}},
	}
	mock.lastID["mystream"] = "1-0"

	manager := NewConsumerGroupManager(mock, "mystream")
	info, err := manager.StreamInfo(ctx)
	if err != nil {
		t.Fatalf("StreamInfo failed: %v", err)
	}

	if info.Length != 1 {
		t.Errorf("Expected length 1, got %d", info.Length)
	}
}

func TestNewConsumer(t *testing.T) {
	mock := newMockStream()
	cg := NewConsumerGroup(mock, "mystream", "mygroup", "consumer1")

	handlerCalled := false
	handler := func(ctx context.Context, entry kvx.StreamEntry) error {
		handlerCalled = true
		return nil
	}

	opts := DefaultConsumerOptions()
	consumer := NewConsumer(cg, handler, opts)

	if consumer.autoAck != opts.AutoAck {
		t.Errorf("Expected autoAck %v, got %v", opts.AutoAck, consumer.autoAck)
	}
	if consumer.batchSize != opts.BatchSize {
		t.Errorf("Expected batchSize %d, got %d", opts.BatchSize, consumer.batchSize)
	}
	_ = handlerCalled
}

func TestNewBatchConsumer(t *testing.T) {
	mock := newMockStream()
	cg := NewConsumerGroup(mock, "mystream", "mygroup", "consumer1")

	handler := func(ctx context.Context, entries []kvx.StreamEntry) error {
		return nil
	}

	opts := DefaultConsumerOptions()
	consumer := NewBatchConsumer(cg, handler, opts)

	if consumer.autoAck != opts.AutoAck {
		t.Errorf("Expected autoAck %v, got %v", opts.AutoAck, consumer.autoAck)
	}
}

func TestNewClaimer(t *testing.T) {
	mock := newMockStream()
	cg := NewConsumerGroup(mock, "mystream", "mygroup", "consumer1")

	handler := func(ctx context.Context, entry kvx.StreamEntry) error {
		return nil
	}

	claimer := NewClaimer(cg, handler, time.Minute, 10)

	if claimer.minIdleTime != time.Minute {
		t.Errorf("Expected minIdleTime 1m, got %v", claimer.minIdleTime)
	}
	if claimer.batchSize != 10 {
		t.Errorf("Expected batchSize 10, got %d", claimer.batchSize)
	}
}

func TestConsumerGroup_Info_NotFound(t *testing.T) {
	ctx := context.Background()
	mock := newMockStream()
	mock.groups["mystream"] = make(map[string]*mockConsumerGroup)

	cg := NewConsumerGroup(mock, "mystream", "nonexistent", "consumer1")
	_, err := cg.Info(ctx)
	if err == nil {
		t.Errorf("Expected error for non-existent group")
	}
}
