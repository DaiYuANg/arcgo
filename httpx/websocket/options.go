package websocket

import (
	"context"
	"net/http"
	"time"

	"github.com/DaiYuANg/arcgo/pkg/option"
)

// Context aliases context.Context for WebSocket handlers and connections.
type Context = context.Context

// Options configures WebSocket upgrade and connection behavior.
type Options struct {
	HandshakeTimeout  time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxMessageSize    int
	EnableCompression bool
	CheckOrigin       func(*http.Request) bool
}

// Option mutates WebSocket options before upgrade.
type Option func(*Options)

const (
	defaultHandshakeTimeout = 5 * time.Second
	defaultMaxMessageSize   = 16 * 1024 * 1024
)

// DefaultOptions returns the default WebSocket server options.
func DefaultOptions() Options {
	return Options{
		HandshakeTimeout: defaultHandshakeTimeout,
		MaxMessageSize:   defaultMaxMessageSize,
	}
}

// WithHandshakeTimeout sets the WebSocket handshake timeout when positive.
func WithHandshakeTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		if timeout > 0 {
			o.HandshakeTimeout = timeout
		}
	}
}

// WithReadTimeout sets the per-read timeout.
func WithReadTimeout(timeout time.Duration) Option {
	return func(o *Options) { o.ReadTimeout = timeout }
}

// WithWriteTimeout sets the per-write timeout.
func WithWriteTimeout(timeout time.Duration) Option {
	return func(o *Options) { o.WriteTimeout = timeout }
}

// WithIdleTimeout sets the idle timeout applied to the connection.
func WithIdleTimeout(timeout time.Duration) Option {
	return func(o *Options) { o.IdleTimeout = timeout }
}

// WithMaxMessageSize sets the maximum accepted message size when positive.
func WithMaxMessageSize(n int) Option {
	return func(o *Options) {
		if n > 0 {
			o.MaxMessageSize = n
		}
	}
}

// WithCompression enables or disables per-message compression.
func WithCompression(enabled bool) Option {
	return func(o *Options) { o.EnableCompression = enabled }
}

// WithCheckOrigin overrides the default origin validator.
func WithCheckOrigin(fn func(*http.Request) bool) Option {
	return func(o *Options) { o.CheckOrigin = fn }
}

func applyOptions(options []Option) Options {
	cfg := DefaultOptions()
	option.Apply(&cfg, options...)
	if cfg.MaxMessageSize <= 0 {
		cfg.MaxMessageSize = defaultMaxMessageSize
	}
	if cfg.HandshakeTimeout <= 0 {
		cfg.HandshakeTimeout = defaultHandshakeTimeout
	}
	return cfg
}
