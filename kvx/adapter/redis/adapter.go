package redis

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/redis/go-redis/v9"
)

// Adapter implements kvx.Client using go-redis.
type Adapter struct {
	client *redis.Client
}

var _ kvx.Client = (*Adapter)(nil)

// New creates a new Redis adapter.
func New(opts kvx.ClientOptions) (*Adapter, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	kvx.LogDebug(logger, opts.Debug, "kvx redis adapter create start", "addrs", len(opts.Addrs), "db", opts.DB)
	if len(opts.Addrs) == 0 {
		kvx.LogError(logger, "kvx redis adapter create failed", "error", kvx.ErrInvalidClientOptions)
		return nil, fmt.Errorf("%w: addrs cannot be empty", kvx.ErrInvalidClientOptions)
	}
	if opts.UseTLS {
		kvx.LogError(logger, "kvx redis adapter create failed", "error", kvx.ErrUnsupportedOption, "reason", "tls")
		return nil, fmt.Errorf("%w: redis adapter does not support tls yet", kvx.ErrUnsupportedOption)
	}
	if opts.MasterName != "" {
		kvx.LogError(logger, "kvx redis adapter create failed", "error", kvx.ErrUnsupportedOption, "reason", "master_name")
		return nil, fmt.Errorf("%w: redis adapter does not support sentinel master selection", kvx.ErrUnsupportedOption)
	}

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
		kvx.LogError(logger, "kvx redis adapter ping failed", "addr", opts.Addrs[0], "error", err)
		return nil, errors.Join(
			fmt.Errorf("failed to connect to Redis: %w", err),
			wrapRedisError("close client after failed ping", rdb.Close()),
		)
	}

	kvx.LogDebug(logger, opts.Debug, "kvx redis adapter create done", "addr", opts.Addrs[0])
	return &Adapter{client: rdb}, nil
}

// NewFromClient creates an adapter from an existing redis.Client.
func NewFromClient(client *redis.Client) *Adapter {
	return &Adapter{client: client}
}

// Close closes the client connection.
func (a *Adapter) Close() error {
	return wrapRedisError("close client", a.client.Close())
}
