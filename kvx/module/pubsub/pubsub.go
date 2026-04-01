// Package pubsub provides Pub/Sub functionality.
package pubsub

import (
	"context"
	"errors"
	"fmt"

	collectionmapping "github.com/DaiYuANg/arcgo/collectionx/mapping"
	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/samber/lo"
)

// PubSub provides high-level pub/sub operations.
type PubSub struct {
	client        kvx.PubSub
	subscriptions *collectionmapping.ConcurrentMap[string, kvx.Subscription]
}

// NewPubSub creates a new PubSub instance.
func NewPubSub(client kvx.PubSub) *PubSub {
	return &PubSub{
		client:        client,
		subscriptions: collectionmapping.NewConcurrentMap[string, kvx.Subscription](),
	}
}

// Publish publishes a message to a channel.
func (p *PubSub) Publish(ctx context.Context, channel string, message []byte) error {
	if err := p.client.Publish(ctx, channel, message); err != nil {
		return fmt.Errorf("publish to channel %q: %w", channel, err)
	}
	return nil
}

// PublishString publishes a string message to a channel.
func (p *PubSub) PublishString(ctx context.Context, channel, message string) error {
	return p.Publish(ctx, channel, []byte(message))
}

// Subscribe subscribes to a channel.
func (p *PubSub) Subscribe(ctx context.Context, channel string) (<-chan []byte, error) {
	if sub, ok := p.subscriptions.Get(channel); ok {
		return sub.Channel(), nil
	}

	sub, err := p.client.Subscribe(ctx, channel)
	if err != nil {
		return nil, fmt.Errorf("subscribe to channel %q: %w", channel, err)
	}

	actual, loaded := p.subscriptions.GetOrStore(channel, sub)
	if loaded {
		if err := sub.Close(); err != nil {
			return nil, fmt.Errorf("close duplicate subscription for channel %q: %w", channel, err)
		}
		return actual.Channel(), nil
	}

	return sub.Channel(), nil
}

// Unsubscribe unsubscribes from a channel.
func (p *PubSub) Unsubscribe(_ context.Context, channel string) error {
	if sub, ok := p.subscriptions.LoadAndDelete(channel); ok {
		if err := sub.Close(); err != nil {
			return fmt.Errorf("unsubscribe from channel %q: %w", channel, err)
		}
	}

	return nil
}

// Close closes all subscriptions.
func (p *PubSub) Close() error {
	errs := lo.FilterMap(p.subscriptions.Keys(), func(channel string, _ int) (error, bool) {
		sub, ok := p.subscriptions.LoadAndDelete(channel)
		if !ok {
			return nil, false
		}
		err := sub.Close()
		if err == nil {
			return nil, false
		}
		return fmt.Errorf("close subscription for channel %q: %w", channel, err), true
	})

	return errors.Join(errs...)
}
