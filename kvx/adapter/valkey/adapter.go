package valkey

import (
	"context"
	"fmt"
	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/valkey-io/valkey-go"
	"time"
)

// Adapter implements kvx.Client using valkey-go.
type Adapter struct {
	client valkey.Client
}

var _ kvx.Client = (*Adapter)(nil)

// New creates a new Valkey adapter.
func New(opts kvx.ClientOptions) (*Adapter, error) {
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: opts.Addrs,
		Password:    opts.Password,
		SelectDB:    opts.DB,
		TLSConfig:   nil, // TODO: support TLS
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Valkey client: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Do(ctx, client.B().Ping().Build()).Error(); err != nil {
		return nil, fmt.Errorf("failed to connect to Valkey: %w", err)
	}

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
