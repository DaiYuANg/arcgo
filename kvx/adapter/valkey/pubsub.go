package valkey

import (
	"context"
	"errors"

	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/valkey-io/valkey-go"
)

// Publish publishes a message to a channel.
func (a *Adapter) Publish(ctx context.Context, channel string, message []byte) error {
	return wrapValkeyError("publish message", a.client.Do(ctx, a.client.B().Publish().Channel(channel).Message(valkey.BinaryString(message)).Build()).Error())
}

// Subscribe subscribes to a channel.
func (a *Adapter) Subscribe(ctx context.Context, channel string) (kvx.Subscription, error) {
	return a.newSubscription(ctx, a.client.B().Subscribe().Channel(channel).Build())
}

// PSubscribe subscribes to channels matching a pattern.
func (a *Adapter) PSubscribe(ctx context.Context, pattern string) (kvx.Subscription, error) {
	return a.newSubscription(ctx, a.client.B().Psubscribe().Pattern(pattern).Build())
}

type valkeySubscription struct {
	ch chan []byte
}

func (s *valkeySubscription) Channel() <-chan []byte {
	return s.ch
}

func (s *valkeySubscription) Close() error {
	return nil
}

func (a *Adapter) newSubscription(ctx context.Context, command valkey.Completed) (kvx.Subscription, error) {
	sub := &valkeySubscription{
		ch: make(chan []byte, 100),
	}

	go func() {
		defer close(sub.ch)
		err := a.client.Receive(ctx, command, func(msg valkey.PubSubMessage) {
			sub.ch <- []byte(msg.Message)
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			return
		}
	}()

	return sub, nil
}
