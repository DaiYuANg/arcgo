package redis

import (
	"context"
	"fmt"
	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/redis/go-redis/v9"
	"time"
)

// Adapter implements kvx.Client using go-redis.
type Adapter struct {
	client *redis.Client
}

var _ kvx.Client = (*Adapter)(nil)

// New creates a new Redis adapter.
func New(opts kvx.ClientOptions) (*Adapter, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:            opts.Addrs[0],
		Password:        opts.Password,
		DB:              opts.DB,
		TLSConfig:       nil, // TODO: support TLS
		PoolSize:        opts.PoolSize,
		MinIdleConns:    opts.MinIdleConns,
		ConnMaxLifetime: opts.ConnMaxLifetime,
		ConnMaxIdleTime: opts.ConnMaxIdleTime,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Adapter{client: rdb}, nil
}

// NewFromClient creates an adapter from an existing redis.Client.
func NewFromClient(client *redis.Client) *Adapter {
	return &Adapter{client: client}
}

// Close closes the client connection.
func (a *Adapter) Close() error {
	return a.client.Close()
}
