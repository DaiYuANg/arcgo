package redis

import (
	"context"
	"errors"
	"sync"

	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/redis/go-redis/v9"
)

// Publish publishes a message to a channel.
func (a *Adapter) Publish(ctx context.Context, channel string, message []byte) error {
	return wrapRedisError("publish message", a.client.Publish(ctx, channel, message).Err())
}

// Subscribe subscribes to a channel.
func (a *Adapter) Subscribe(ctx context.Context, channel string) (kvx.Subscription, error) {
	pubsub := a.client.Subscribe(ctx, channel)
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return nil, errors.Join(
			wrapRedisError("subscribe to channel", err),
			wrapRedisError("close failed subscription", pubsub.Close()),
		)
	}

	return &redisSubscription{pubsub: pubsub}, nil
}

// PSubscribe subscribes to channels matching a pattern.
func (a *Adapter) PSubscribe(ctx context.Context, pattern string) (kvx.Subscription, error) {
	pubsub := a.client.PSubscribe(ctx, pattern)
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return nil, errors.Join(
			wrapRedisError("psubscribe to pattern", err),
			wrapRedisError("close failed pattern subscription", pubsub.Close()),
		)
	}

	return &redisSubscription{pubsub: pubsub}, nil
}

type redisSubscription struct {
	pubsub *redis.PubSub
	once   sync.Once
	ch     chan []byte
}

func (s *redisSubscription) Channel() <-chan []byte {
	s.once.Do(func() {
		s.ch = make(chan []byte, 100)
		go func() {
			defer close(s.ch)
			ch := s.pubsub.Channel()
			for msg := range ch {
				s.ch <- []byte(msg.Payload)
			}
		}()
	})
	return s.ch
}

func (s *redisSubscription) Close() error {
	return wrapRedisError("close subscription", s.pubsub.Close())
}
