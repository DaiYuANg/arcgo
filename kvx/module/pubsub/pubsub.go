// Package pubsub provides Pub/Sub functionality.
package pubsub

import (
	"context"
	"sync"

	"github.com/DaiYuANg/archgo/kvx"
)

// PubSub provides high-level pub/sub operations.
type PubSub struct {
	client        kvx.PubSub
	subscriptions map[string]kvx.Subscription
	mu            sync.RWMutex
}

// NewPubSub creates a new PubSub instance.
func NewPubSub(client kvx.PubSub) *PubSub {
	return &PubSub{
		client:        client,
		subscriptions: make(map[string]kvx.Subscription),
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
	p.mu.Lock()
	defer p.mu.Unlock()

	if sub, ok := p.subscriptions[channel]; ok {
		return sub.Channel(), nil
	}

	sub, err := p.client.Subscribe(ctx, channel)
	if err != nil {
		return nil, err
	}

	p.subscriptions[channel] = sub
	return sub.Channel(), nil
}

// Unsubscribe unsubscribes from a channel.
func (p *PubSub) Unsubscribe(ctx context.Context, channel string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if sub, ok := p.subscriptions[channel]; ok {
		if err := sub.Close(); err != nil {
			return err
		}
		delete(p.subscriptions, channel)
	}

	return nil
}

// Close closes all subscriptions.
func (p *PubSub) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var lastErr error
	for channel, sub := range p.subscriptions {
		if err := sub.Close(); err != nil {
			lastErr = err
		}
		delete(p.subscriptions, channel)
	}

	return lastErr
}
