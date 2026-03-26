package stream

import (
	"context"
	"fmt"
	"time"

	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/samber/lo"
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
	return cg.client.XGroupCreate(ctx, cg.streamKey, cg.groupName, startID)
}

// CreateFromBeginning creates the consumer group reading from the beginning.
func (cg *ConsumerGroup) CreateFromBeginning(ctx context.Context) error {
	return cg.client.XGroupCreate(ctx, cg.streamKey, cg.groupName, "0")
}

// CreateFromLatest creates the consumer group reading from new messages only.
func (cg *ConsumerGroup) CreateFromLatest(ctx context.Context) error {
	return cg.client.XGroupCreate(ctx, cg.streamKey, cg.groupName, "$")
}

// Destroy destroys the consumer group.
func (cg *ConsumerGroup) Destroy(ctx context.Context) error {
	return cg.client.XGroupDestroy(ctx, cg.streamKey, cg.groupName)
}

// Read reads messages from the consumer group.
func (cg *ConsumerGroup) Read(ctx context.Context, count int64, block time.Duration) ([]kvx.StreamEntry, error) {
	streams := map[string]string{
		cg.streamKey: ">", // Read only new messages for this consumer
	}

	results, err := cg.client.XReadGroup(ctx, cg.groupName, cg.consumerName, streams, count, block)
	if err != nil {
		return nil, err
	}

	return results[cg.streamKey], nil
}

// ReadPending reads pending messages (previously delivered but not acknowledged).
func (cg *ConsumerGroup) ReadPending(ctx context.Context, count int64) ([]kvx.StreamEntry, error) {
	streams := map[string]string{
		cg.streamKey: "0", // Read pending messages
	}

	results, err := cg.client.XReadGroup(ctx, cg.groupName, cg.consumerName, streams, count, 0)
	if err != nil {
		return nil, err
	}

	return results[cg.streamKey], nil
}

// Ack acknowledges processing of messages.
func (cg *ConsumerGroup) Ack(ctx context.Context, ids []string) error {
	return cg.client.XAck(ctx, cg.streamKey, cg.groupName, ids)
}

// AckEntry acknowledges a single entry.
func (cg *ConsumerGroup) AckEntry(ctx context.Context, id string) error {
	return cg.client.XAck(ctx, cg.streamKey, cg.groupName, []string{id})
}

// Pending gets pending entries information.
func (cg *ConsumerGroup) Pending(ctx context.Context) (*kvx.PendingInfo, error) {
	return cg.client.XPending(ctx, cg.streamKey, cg.groupName)
}

// PendingRange gets pending entries in a range.
func (cg *ConsumerGroup) PendingRange(ctx context.Context, start string, stop string, count int64) ([]kvx.PendingEntry, error) {
	return cg.client.XPendingRange(ctx, cg.streamKey, cg.groupName, start, stop, count)
}

// Claim claims pending entries from other consumers.
func (cg *ConsumerGroup) Claim(ctx context.Context, ids []string, minIdleTime time.Duration) ([]kvx.StreamEntry, error) {
	return cg.client.XClaim(ctx, cg.streamKey, cg.groupName, cg.consumerName, minIdleTime, ids)
}

// AutoClaim auto-claims pending entries that have been idle for minIdleTime.
func (cg *ConsumerGroup) AutoClaim(ctx context.Context, minIdleTime time.Duration, count int64) (string, []kvx.StreamEntry, error) {
	return cg.client.XAutoClaim(ctx, cg.streamKey, cg.groupName, cg.consumerName, minIdleTime, "0", count)
}

// Info gets information about the consumer group.
func (cg *ConsumerGroup) Info(ctx context.Context) (*kvx.GroupInfo, error) {
	groups, err := cg.client.XInfoGroups(ctx, cg.streamKey)
	if err != nil {
		return nil, err
	}

	for _, group := range groups {
		if group.Name == cg.groupName {
			return &group, nil
		}
	}

	return nil, fmt.Errorf("consumer group %s not found", cg.groupName)
}

// ConsumerInfo gets information about this consumer.
func (cg *ConsumerGroup) ConsumerInfo(ctx context.Context) (*kvx.ConsumerInfo, error) {
	consumers, err := cg.client.XInfoConsumers(ctx, cg.streamKey, cg.groupName)
	if err != nil {
		return nil, err
	}

	for _, consumer := range consumers {
		if consumer.Name == cg.consumerName {
			return &consumer, nil
		}
	}

	return nil, fmt.Errorf("consumer %s not found in group %s", cg.consumerName, cg.groupName)
}

// DeleteConsumer deletes this consumer from the group.
func (cg *ConsumerGroup) DeleteConsumer(ctx context.Context) error {
	return cg.client.XGroupDelConsumer(ctx, cg.streamKey, cg.groupName, cg.consumerName)
}

// StreamInfo gets information about the stream.
func (cg *ConsumerGroup) StreamInfo(ctx context.Context) (*kvx.StreamInfo, error) {
	return cg.client.XInfoStream(ctx, cg.streamKey)
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
	return m.client.XGroupCreate(ctx, m.streamKey, groupName, startID)
}

// CreateGroupFromBeginning creates a new consumer group reading from the beginning.
func (m *ConsumerGroupManager) CreateGroupFromBeginning(ctx context.Context, groupName string) error {
	return m.client.XGroupCreate(ctx, m.streamKey, groupName, "0")
}

// CreateGroupFromLatest creates a new consumer group reading from new messages.
func (m *ConsumerGroupManager) CreateGroupFromLatest(ctx context.Context, groupName string) error {
	return m.client.XGroupCreate(ctx, m.streamKey, groupName, "$")
}

// DestroyGroup destroys a consumer group.
func (m *ConsumerGroupManager) DestroyGroup(ctx context.Context, groupName string) error {
	return m.client.XGroupDestroy(ctx, m.streamKey, groupName)
}

// ListGroups lists all consumer groups for the stream.
func (m *ConsumerGroupManager) ListGroups(ctx context.Context) ([]kvx.GroupInfo, error) {
	return m.client.XInfoGroups(ctx, m.streamKey)
}

// GetConsumer creates a ConsumerGroup instance for a specific consumer.
func (m *ConsumerGroupManager) GetConsumer(groupName, consumerName string) *ConsumerGroup {
	return NewConsumerGroup(m.client, m.streamKey, groupName, consumerName)
}

// StreamInfo gets information about the stream.
func (m *ConsumerGroupManager) StreamInfo(ctx context.Context) (*kvx.StreamInfo, error) {
	return m.client.XInfoStream(ctx, m.streamKey)
}

// Consumer handles message processing with automatic acknowledgment.
type Consumer struct {
	group        *ConsumerGroup
	handler      MessageHandler
	autoAck      bool
	batchSize    int64
	blockTimeout time.Duration
}

// MessageHandler is the callback function for processing messages.
type MessageHandler func(ctx context.Context, entry kvx.StreamEntry) error

// BatchMessageHandler is the callback function for processing messages in batch.
type BatchMessageHandler func(ctx context.Context, entries []kvx.StreamEntry) error

// ConsumerOptions contains options for creating a Consumer.
type ConsumerOptions struct {
	AutoAck      bool
	BatchSize    int64
	BlockTimeout time.Duration
}

// DefaultConsumerOptions returns default consumer options.
func DefaultConsumerOptions() ConsumerOptions {
	return ConsumerOptions{
		AutoAck:      true,
		BatchSize:    10,
		BlockTimeout: 5 * time.Second,
	}
}

// NewConsumer creates a new Consumer.
func NewConsumer(group *ConsumerGroup, handler MessageHandler, opts ConsumerOptions) *Consumer {
	return &Consumer{
		group:        group,
		handler:      handler,
		autoAck:      opts.AutoAck,
		batchSize:    opts.BatchSize,
		blockTimeout: opts.BlockTimeout,
	}
}

// Run starts the consumer loop.
func (c *Consumer) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		entries, err := c.group.Read(ctx, c.batchSize, c.blockTimeout)
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			continue
		}

		idsToAck := make([]string, 0, len(entries))

		for _, entry := range entries {
			if err := c.handler(ctx, entry); err != nil {
				// Handle error - could log, retry, or continue based on strategy
				continue
			}

			if c.autoAck {
				idsToAck = append(idsToAck, entry.ID)
			}
		}

		if c.autoAck && len(idsToAck) > 0 {
			_ = c.group.Ack(ctx, idsToAck)
		}
	}
}

// BatchConsumer handles message processing in batches.
type BatchConsumer struct {
	group        *ConsumerGroup
	handler      BatchMessageHandler
	autoAck      bool
	batchSize    int64
	blockTimeout time.Duration
	maxWaitTime  time.Duration
}

// NewBatchConsumer creates a new BatchConsumer.
func NewBatchConsumer(group *ConsumerGroup, handler BatchMessageHandler, opts ConsumerOptions) *BatchConsumer {
	return &BatchConsumer{
		group:        group,
		handler:      handler,
		autoAck:      opts.AutoAck,
		batchSize:    opts.BatchSize,
		blockTimeout: opts.BlockTimeout,
		maxWaitTime:  time.Second,
	}
}

// Run starts the batch consumer loop.
func (c *BatchConsumer) Run(ctx context.Context) error {
	buffer := make([]kvx.StreamEntry, 0, c.batchSize)
	timer := time.NewTimer(c.maxWaitTime)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			if len(buffer) > 0 {
				if err := c.processBatch(ctx, buffer); err != nil {
					return err
				}
				buffer = buffer[:0]
			}
			timer.Reset(c.maxWaitTime)
		default:
		}

		entries, err := c.group.Read(ctx, c.batchSize-int64(len(buffer)), c.blockTimeout)
		if err != nil {
			return err
		}

		buffer = append(buffer, entries...)

		if int64(len(buffer)) >= c.batchSize {
			if err := c.processBatch(ctx, buffer); err != nil {
				return err
			}
			buffer = buffer[:0]
			timer.Reset(c.maxWaitTime)
		}
	}
}

func (c *BatchConsumer) processBatch(ctx context.Context, entries []kvx.StreamEntry) error {
	if err := c.handler(ctx, entries); err != nil {
		return err
	}

	if c.autoAck {
		ids := lo.Map(entries, func(entry kvx.StreamEntry, _ int) string {
			return entry.ID
		})
		return c.group.Ack(ctx, ids)
	}

	return nil
}

// Claimer handles claiming stale messages from other consumers.
type Claimer struct {
	group       *ConsumerGroup
	handler     MessageHandler
	minIdleTime time.Duration
	batchSize   int64
	interval    time.Duration
}

// NewClaimer creates a new Claimer.
func NewClaimer(group *ConsumerGroup, handler MessageHandler, minIdleTime time.Duration, batchSize int64) *Claimer {
	return &Claimer{
		group:       group,
		handler:     handler,
		minIdleTime: minIdleTime,
		batchSize:   batchSize,
		interval:    time.Minute,
	}
}

// Run starts the claimer loop.
func (c *Claimer) Run(ctx context.Context) error {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	// Run immediately on start
	if err := c.claimAndProcess(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := c.claimAndProcess(ctx); err != nil {
				return err
			}
		}
	}
}

func (c *Claimer) claimAndProcess(ctx context.Context) error {
	for {
		_, entries, err := c.group.AutoClaim(ctx, c.minIdleTime, c.batchSize)
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			break
		}

		idsToAck := make([]string, 0, len(entries))
		for _, entry := range entries {
			if err := c.handler(ctx, entry); err != nil {
				continue
			}
			idsToAck = append(idsToAck, entry.ID)
		}

		if len(idsToAck) > 0 {
			_ = c.group.Ack(ctx, idsToAck)
		}
	}

	return nil
}
