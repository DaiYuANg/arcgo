// Package pubsub provides Pub/Sub functionality.
package pubsub

import (
	"context"
	"errors"

	collectionmapping "github.com/DaiYuANg/arcgo/collectionx/mapping"
	"github.com/DaiYuANg/arcgo/kvx"
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
	return p.client.Publish(ctx, channel, message)
}

// PublishString publishes a string message to a channel.
func (p *PubSub) PublishString(ctx context.Context, channel string, message string) error {
	return p.Publish(ctx, channel, []byte(message))
}

// Subscribe subscribes to a channel.
func (p *PubSub) Subscribe(ctx context.Context, channel string) (<-chan []byte, error) {
	if sub, ok := p.subscriptions.Get(channel); ok {
		return sub.Channel(), nil
	}

	sub, err := p.client.Subscribe(ctx, channel)
	if err != nil {
		return nil, err
	}

	actual, loaded := p.subscriptions.GetOrStore(channel, sub)
	if loaded {
		_ = sub.Close()
		return actual.Channel(), nil
	}

	return sub.Channel(), nil
}

// Unsubscribe unsubscribes from a channel.
func (p *PubSub) Unsubscribe(ctx context.Context, channel string) error {
	if sub, ok := p.subscriptions.LoadAndDelete(channel); ok {
		if err := sub.Close(); err != nil {
			return err
		}
	}

	return nil
}

// Close closes all subscriptions.
func (p *PubSub) Close() error {
	errs := make([]error, 0)
	for _, channel := range p.subscriptions.Keys() {
		sub, ok := p.subscriptions.LoadAndDelete(channel)
		if !ok {
			continue
		}
		if err := sub.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
