package stream

import (
	"context"
	"fmt"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/kvx"
)

// ConsumerGroup provides high-level consumer group operations.
type ConsumerGroup struct {
	client       kvx.Stream
	streamKey    string
	groupName    string
	consumerName string
}

// NewConsumerGroup creates a new ConsumerGroup.
func NewConsumerGroup(client kvx.Stream, streamKey, groupName, consumerName string) *ConsumerGroup {
	return &ConsumerGroup{
		client:       client,
		streamKey:    streamKey,
		groupName:    groupName,
		consumerName: consumerName,
	}
}

// Create creates the consumer group.
func (cg *ConsumerGroup) Create(ctx context.Context, startID string) error {
	return wrapError(
		cg.client.XGroupCreate(ctx, cg.streamKey, cg.groupName, startID),
		cg.groupAction("create consumer group"),
	)
}

// CreateFromBeginning creates the consumer group reading from the beginning.
func (cg *ConsumerGroup) CreateFromBeginning(ctx context.Context) error {
	return wrapError(
		cg.client.XGroupCreate(ctx, cg.streamKey, cg.groupName, "0"),
		cg.groupAction("create consumer group from beginning"),
	)
}

// CreateFromLatest creates the consumer group reading from new messages only.
func (cg *ConsumerGroup) CreateFromLatest(ctx context.Context) error {
	return wrapError(
		cg.client.XGroupCreate(ctx, cg.streamKey, cg.groupName, "$"),
		cg.groupAction("create consumer group from latest"),
	)
}

// Destroy destroys the consumer group.
func (cg *ConsumerGroup) Destroy(ctx context.Context) error {
	return wrapError(
		cg.client.XGroupDestroy(ctx, cg.streamKey, cg.groupName),
		cg.groupAction("destroy consumer group"),
	)
}

// Read reads messages from the consumer group.
func (cg *ConsumerGroup) Read(ctx context.Context, count int64, block time.Duration) (collectionx.List[kvx.StreamEntry], error) {
	streams := map[string]string{
		cg.streamKey: ">",
	}

	results, readErr := cg.client.XReadGroup(ctx, cg.groupName, cg.consumerName, streams, count, block)
	results, err := wrapResult(results, readErr, cg.consumerAction("read consumer group entries"))
	if err != nil {
		return nil, err
	}

	return streamEntriesFromMultiMap(results, cg.streamKey), nil
}

// ReadPending reads pending messages (previously delivered but not acknowledged).
func (cg *ConsumerGroup) ReadPending(ctx context.Context, count int64) (collectionx.List[kvx.StreamEntry], error) {
	streams := map[string]string{
		cg.streamKey: "0",
	}

	results, readErr := cg.client.XReadGroup(ctx, cg.groupName, cg.consumerName, streams, count, 0)
	results, err := wrapResult(results, readErr, cg.consumerAction("read pending consumer group entries"))
	if err != nil {
		return nil, err
	}

	return streamEntriesFromMultiMap(results, cg.streamKey), nil
}

// Ack acknowledges processing of messages.
func (cg *ConsumerGroup) Ack(ctx context.Context, ids []string) error {
	return wrapError(
		cg.client.XAck(ctx, cg.streamKey, cg.groupName, ids),
		cg.groupAction("ack consumer group entries"),
	)
}

// AckEntry acknowledges a single entry.
func (cg *ConsumerGroup) AckEntry(ctx context.Context, id string) error {
	return wrapError(
		cg.client.XAck(ctx, cg.streamKey, cg.groupName, []string{id}),
		cg.groupAction("ack consumer group entry"),
	)
}

// Pending gets pending entries information.
func (cg *ConsumerGroup) Pending(ctx context.Context) (*kvx.PendingInfo, error) {
	info, err := cg.client.XPending(ctx, cg.streamKey, cg.groupName)
	return wrapResult(info, err, cg.groupAction("read pending info"))
}

// PendingRange gets pending entries in a range.
func (cg *ConsumerGroup) PendingRange(ctx context.Context, start, stop string, count int64) (collectionx.List[kvx.PendingEntry], error) {
	entries, err := cg.client.XPendingRange(ctx, cg.streamKey, cg.groupName, start, stop, count)
	return wrapResult(entries, err, cg.groupAction("read pending entry range"))
}

// Claim claims pending entries from other consumers.
func (cg *ConsumerGroup) Claim(ctx context.Context, ids []string, minIdleTime time.Duration) (collectionx.List[kvx.StreamEntry], error) {
	entries, err := cg.client.XClaim(ctx, cg.streamKey, cg.groupName, cg.consumerName, minIdleTime, ids)
	return wrapResult(entries, err, cg.consumerAction("claim pending entries"))
}

// AutoClaim auto-claims pending entries that have been idle for minIdleTime.
func (cg *ConsumerGroup) AutoClaim(ctx context.Context, minIdleTime time.Duration, count int64) (string, collectionx.List[kvx.StreamEntry], error) {
	nextID, entries, err := cg.client.XAutoClaim(
		ctx,
		cg.streamKey,
		cg.groupName,
		cg.consumerName,
		minIdleTime,
		"0",
		count,
	)
	if err != nil {
		return "", nil, fmt.Errorf("%s: %w", cg.consumerAction("auto claim pending entries"), err)
	}

	return nextID, entries, nil
}

// Info gets information about the consumer group.
func (cg *ConsumerGroup) Info(ctx context.Context) (*kvx.GroupInfo, error) {
	groups, groupsErr := cg.client.XInfoGroups(ctx, cg.streamKey)
	groups, err := wrapResult(groups, groupsErr, cg.groupAction("read consumer group info"))
	if err != nil {
		return nil, err
	}

	group, ok := groups.FirstWhere(func(_ int, group kvx.GroupInfo) bool {
		return group.Name == cg.groupName
	}).Get()
	if !ok {
		return nil, fmt.Errorf("consumer group %s not found", cg.groupName)
	}
	return &group, nil
}

// ConsumerInfo gets information about this consumer.
func (cg *ConsumerGroup) ConsumerInfo(ctx context.Context) (*kvx.ConsumerInfo, error) {
	consumers, consumersErr := cg.client.XInfoConsumers(ctx, cg.streamKey, cg.groupName)
	consumers, err := wrapResult(consumers, consumersErr, cg.consumerAction("read consumer info"))
	if err != nil {
		return nil, err
	}

	consumer, ok := consumers.FirstWhere(func(_ int, consumer kvx.ConsumerInfo) bool {
		return consumer.Name == cg.consumerName
	}).Get()
	if !ok {
		return nil, fmt.Errorf("consumer %s not found in group %s", cg.consumerName, cg.groupName)
	}
	return &consumer, nil
}

// DeleteConsumer deletes this consumer from the group.
func (cg *ConsumerGroup) DeleteConsumer(ctx context.Context) error {
	return wrapError(
		cg.client.XGroupDelConsumer(ctx, cg.streamKey, cg.groupName, cg.consumerName),
		cg.consumerAction("delete consumer"),
	)
}

// StreamInfo gets information about the stream.
func (cg *ConsumerGroup) StreamInfo(ctx context.Context) (*kvx.StreamInfo, error) {
	info, err := cg.client.XInfoStream(ctx, cg.streamKey)
	return wrapResult(info, err, cg.groupAction("read stream info"))
}

// ConsumerGroupManager manages multiple consumer groups for a stream.
type ConsumerGroupManager struct {
	client    kvx.Stream
	streamKey string
}

// NewConsumerGroupManager creates a new ConsumerGroupManager.
func NewConsumerGroupManager(client kvx.Stream, streamKey string) *ConsumerGroupManager {
	return &ConsumerGroupManager{
		client:    client,
		streamKey: streamKey,
	}
}

// CreateGroup creates a new consumer group.
func (m *ConsumerGroupManager) CreateGroup(ctx context.Context, groupName, startID string) error {
	return wrapError(
		m.client.XGroupCreate(ctx, m.streamKey, groupName, startID),
		m.groupAction("create consumer group", groupName),
	)
}

// CreateGroupFromBeginning creates a new consumer group reading from the beginning.
func (m *ConsumerGroupManager) CreateGroupFromBeginning(ctx context.Context, groupName string) error {
	return wrapError(
		m.client.XGroupCreate(ctx, m.streamKey, groupName, "0"),
		m.groupAction("create consumer group from beginning", groupName),
	)
}

// CreateGroupFromLatest creates a new consumer group reading from new messages.
func (m *ConsumerGroupManager) CreateGroupFromLatest(ctx context.Context, groupName string) error {
	return wrapError(
		m.client.XGroupCreate(ctx, m.streamKey, groupName, "$"),
		m.groupAction("create consumer group from latest", groupName),
	)
}

// DestroyGroup destroys a consumer group.
func (m *ConsumerGroupManager) DestroyGroup(ctx context.Context, groupName string) error {
	return wrapError(
		m.client.XGroupDestroy(ctx, m.streamKey, groupName),
		m.groupAction("destroy consumer group", groupName),
	)
}

// ListGroups lists all consumer groups for the stream.
func (m *ConsumerGroupManager) ListGroups(ctx context.Context) (collectionx.List[kvx.GroupInfo], error) {
	groups, err := m.client.XInfoGroups(ctx, m.streamKey)
	return wrapResult(groups, err, m.streamAction("list consumer groups"))
}

// GetConsumer creates a ConsumerGroup instance for a specific consumer.
func (m *ConsumerGroupManager) GetConsumer(groupName, consumerName string) *ConsumerGroup {
	return NewConsumerGroup(m.client, m.streamKey, groupName, consumerName)
}

// StreamInfo gets information about the stream.
func (m *ConsumerGroupManager) StreamInfo(ctx context.Context) (*kvx.StreamInfo, error) {
	info, err := m.client.XInfoStream(ctx, m.streamKey)
	return wrapResult(info, err, m.streamAction("read stream info"))
}

func (cg *ConsumerGroup) groupAction(action string) string {
	return action + " for group " + cg.groupName + " on stream " + cg.streamKey
}

func (cg *ConsumerGroup) consumerAction(action string) string {
	return action + " for consumer " + cg.consumerName + " in group " + cg.groupName + " on stream " + cg.streamKey
}

func (m *ConsumerGroupManager) groupAction(action, groupName string) string {
	return action + " for group " + groupName + " on stream " + m.streamKey
}

func (m *ConsumerGroupManager) streamAction(action string) string {
	return action + " on stream " + m.streamKey
}
