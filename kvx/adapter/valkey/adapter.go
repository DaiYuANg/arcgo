package valkey

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/valkey-io/valkey-go"
)

// Adapter implements kvx.Client using valkey-go.
type Adapter struct {
	client valkey.Client
}

var _ kvx.Client = (*Adapter)(nil)

// New creates a new Valkey adapter.
func New(opts kvx.ClientOptions) (*Adapter, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	kvx.LogDebug(logger, opts.Debug, "kvx valkey adapter create start", "addrs", len(opts.Addrs), "db", opts.DB)
	if len(opts.Addrs) == 0 {
		kvx.LogError(logger, "kvx valkey adapter create failed", "error", kvx.ErrInvalidClientOptions)
		return nil, fmt.Errorf("%w: addrs cannot be empty", kvx.ErrInvalidClientOptions)
	}
	if opts.UseTLS {
		kvx.LogError(logger, "kvx valkey adapter create failed", "error", kvx.ErrUnsupportedOption, "reason", "tls")
		return nil, fmt.Errorf("%w: valkey adapter does not support tls yet", kvx.ErrUnsupportedOption)
	}
	if opts.MasterName != "" {
		kvx.LogError(logger, "kvx valkey adapter create failed", "error", kvx.ErrUnsupportedOption, "reason", "master_name")
		return nil, fmt.Errorf("%w: valkey adapter does not support sentinel master selection", kvx.ErrUnsupportedOption)
	}

	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: opts.Addrs,
		Password:    opts.Password,
		SelectDB:    opts.DB,
		TLSConfig:   nil, // TODO: support TLS
	})
	if err != nil {
		kvx.LogError(logger, "kvx valkey adapter create failed", "stage", "new_client", "error", err)
		return nil, fmt.Errorf("failed to create Valkey client: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Do(ctx, client.B().Ping().Build()).Error(); err != nil {
		kvx.LogError(logger, "kvx valkey adapter ping failed", "addr", opts.Addrs[0], "error", err)
		client.Close()
		return nil, fmt.Errorf("failed to connect to Valkey: %w", err)
	}

	kvx.LogDebug(logger, opts.Debug, "kvx valkey adapter create done", "addr", opts.Addrs[0])
	return &Adapter{client: client}, nil
}

// NewFromClient creates an adapter from an existing valkey.Client.
func NewFromClient(client valkey.Client) *Adapter {
	return &Adapter{client: client}
}

// Close closes the client connection.
func (a *Adapter) Close() error {
	a.client.Close()
	return nil
}
