package stream

import (
	"context"
	"time"

	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/samber/lo"
)

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
		if err := waitForShutdown(ctx, "run consumer"); err != nil {
			return err
		}

		entries, err := c.group.Read(ctx, c.batchSize, c.blockTimeout)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			continue
		}

		if err := c.processEntries(ctx, entries); err != nil {
			return err
		}
	}
}

func (c *Consumer) processEntries(ctx context.Context, entries []kvx.StreamEntry) error {
	idsToAck := make([]string, 0, len(entries))
	for _, entry := range entries {
		if err := c.handler(ctx, entry); err != nil {
			continue
		}
		if c.autoAck {
			idsToAck = append(idsToAck, entry.ID)
		}
	}

	if !c.autoAck || len(idsToAck) == 0 {
		return nil
	}

	return wrapError(c.group.Ack(ctx, idsToAck), "ack processed consumer entries")
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
	defer stopTimer(timer)

	for {
		flushed, err := c.flushOnTimer(ctx, timer, &buffer)
		if err != nil {
			return err
		}
		if flushed {
			continue
		}

		entries, err := c.group.Read(ctx, c.remainingCapacity(buffer), c.blockTimeout)
		if err != nil {
			return err
		}

		buffer = append(buffer, entries...)
		if err := c.flushIfReady(ctx, timer, &buffer); err != nil {
			return err
		}
	}
}

func (c *BatchConsumer) flushOnTimer(ctx context.Context, timer *time.Timer, buffer *[]kvx.StreamEntry) (bool, error) {
	select {
	case <-ctx.Done():
		return false, wrapContextError(ctx, "run batch consumer")
	case <-timer.C:
		if err := c.flushBuffer(ctx, buffer); err != nil {
			return false, err
		}
		resetTimer(timer, c.maxWaitTime)
		return true, nil
	default:
		return false, nil
	}
}

func (c *BatchConsumer) remainingCapacity(buffer []kvx.StreamEntry) int64 {
	remaining := c.batchSize - int64(len(buffer))
	if remaining > 0 {
		return remaining
	}

	return c.batchSize
}

func (c *BatchConsumer) flushIfReady(ctx context.Context, timer *time.Timer, buffer *[]kvx.StreamEntry) error {
	if int64(len(*buffer)) < c.batchSize {
		return nil
	}
	if err := c.flushBuffer(ctx, buffer); err != nil {
		return err
	}

	resetTimer(timer, c.maxWaitTime)
	return nil
}

func (c *BatchConsumer) flushBuffer(ctx context.Context, buffer *[]kvx.StreamEntry) error {
	if len(*buffer) == 0 {
		return nil
	}
	if err := c.processBatch(ctx, *buffer); err != nil {
		return err
	}

	*buffer = (*buffer)[:0]
	return nil
}

func (c *BatchConsumer) processBatch(ctx context.Context, entries []kvx.StreamEntry) error {
	if err := c.handler(ctx, entries); err != nil {
		return err
	}
	if !c.autoAck {
		return nil
	}

	ids := lo.Map(entries, func(entry kvx.StreamEntry, _ int) string {
		return entry.ID
	})
	return wrapError(c.group.Ack(ctx, ids), "ack processed batch entries")
}

func waitForShutdown(ctx context.Context, action string) error {
	select {
	case <-ctx.Done():
		return wrapContextError(ctx, action)
	default:
		return nil
	}
}

func stopTimer(timer *time.Timer) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}

func resetTimer(timer *time.Timer, wait time.Duration) {
	stopTimer(timer)
	timer.Reset(wait)
}
