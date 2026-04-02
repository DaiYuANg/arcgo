package stream

import (
	"context"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
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
		if entries.IsEmpty() {
			return nil
		}
		if err := c.processClaimedEntries(ctx, entries); err != nil {
			return err
		}
	}
}

func (c *Claimer) processClaimedEntries(ctx context.Context, entries collectionx.List[kvx.StreamEntry]) error {
	idsToAck := collectionx.FilterMapList(entries, func(_ int, entry kvx.StreamEntry) (string, bool) {
		if err := c.handler(ctx, entry); err != nil {
			return "", false
		}
		return entry.ID, true
	})
	if idsToAck.IsEmpty() {
		return nil
	}

	return wrapError(c.group.Ack(ctx, idsToAck.Values()), "ack claimed entries")
}
