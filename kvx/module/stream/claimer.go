package stream

import (
	"context"
	"time"

	"github.com/DaiYuANg/arcgo/kvx"
)

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

	if err := c.claimAndProcess(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return wrapContextError(ctx, "run stream claimer")
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
			return nil
		}
		if err := c.processClaimedEntries(ctx, entries); err != nil {
			return err
		}
	}
}

func (c *Claimer) processClaimedEntries(ctx context.Context, entries []kvx.StreamEntry) error {
	idsToAck := make([]string, 0, len(entries))
	for _, entry := range entries {
		if err := c.handler(ctx, entry); err != nil {
			continue
		}
		idsToAck = append(idsToAck, entry.ID)
	}

	if len(idsToAck) == 0 {
		return nil
	}

	return wrapError(c.group.Ack(ctx, idsToAck), "ack claimed entries")
}
