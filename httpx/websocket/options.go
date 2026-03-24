package websocket

import (
	"context"
	"net/http"
	"time"
)

type Context = context.Context

type Options struct {
	HandshakeTimeout  time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxMessageSize    int
	EnableCompression bool
	CheckOrigin       func(*http.Request) bool
}

type Option func(*Options)

func DefaultOptions() Options {
	return Options{
		HandshakeTimeout: 5 * time.Second,
		MaxMessageSize:   16 * 1024 * 1024,
	}
}

func WithHandshakeTimeout(timeout time.Duration) Option {
	return func(o *Options) { o.HandshakeTimeout = timeout }
}

func WithReadTimeout(timeout time.Duration) Option {
	return func(o *Options) { o.ReadTimeout = timeout }
}

func WithWriteTimeout(timeout time.Duration) Option {
	return func(o *Options) { o.WriteTimeout = timeout }
}

func WithIdleTimeout(timeout time.Duration) Option {
	return func(o *Options) { o.IdleTimeout = timeout }
}

func WithMaxMessageSize(n int) Option {
	return func(o *Options) { o.MaxMessageSize = n }
}

func WithCompression(enabled bool) Option {
	return func(o *Options) { o.EnableCompression = enabled }
}

func WithCheckOrigin(fn func(*http.Request) bool) Option {
	return func(o *Options) { o.CheckOrigin = fn }
}

func applyOptions(options []Option) Options {
	cfg := DefaultOptions()
	for _, opt := range options {
		if opt != nil {
			opt(&cfg)
		}
	}
	if cfg.MaxMessageSize <= 0 {
		cfg.MaxMessageSize = 16 * 1024 * 1024
	}
	if cfg.HandshakeTimeout <= 0 {
		cfg.HandshakeTimeout = 5 * time.Second
	}
	return cfg
}
