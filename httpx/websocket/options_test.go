package websocket

import (
	"net/http"
	"testing"
	"time"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	if opts.HandshakeTimeout <= 0 {
		t.Fatalf("expected handshake timeout > 0, got %v", opts.HandshakeTimeout)
	}
	if opts.MaxMessageSize <= 0 {
		t.Fatalf("expected max message size > 0, got %d", opts.MaxMessageSize)
	}
}

func TestApplyOptions(t *testing.T) {
	checkOrigin := func(*http.Request) bool { return true }
	opts := applyOptions([]Option{
		WithHandshakeTimeout(2 * time.Second),
		WithReadTimeout(3 * time.Second),
		WithWriteTimeout(4 * time.Second),
		WithIdleTimeout(5 * time.Second),
		WithMaxMessageSize(128),
		WithCompression(true),
		WithCheckOrigin(checkOrigin),
	})

	if opts.HandshakeTimeout != 2*time.Second {
		t.Fatalf("unexpected handshake timeout: %v", opts.HandshakeTimeout)
	}
	if opts.ReadTimeout != 3*time.Second {
		t.Fatalf("unexpected read timeout: %v", opts.ReadTimeout)
	}
	if opts.WriteTimeout != 4*time.Second {
		t.Fatalf("unexpected write timeout: %v", opts.WriteTimeout)
	}
	if opts.IdleTimeout != 5*time.Second {
		t.Fatalf("unexpected idle timeout: %v", opts.IdleTimeout)
	}
	if opts.MaxMessageSize != 128 {
		t.Fatalf("unexpected max message size: %d", opts.MaxMessageSize)
	}
	if !opts.EnableCompression {
		t.Fatal("expected compression to be enabled")
	}
	if opts.CheckOrigin == nil || !opts.CheckOrigin(nil) {
		t.Fatal("expected check origin function to be set")
	}
}

func TestApplyOptionsFallback(t *testing.T) {
	opts := applyOptions([]Option{
		WithHandshakeTimeout(0),
		WithMaxMessageSize(0),
	})
	if opts.HandshakeTimeout <= 0 {
		t.Fatalf("expected fallback handshake timeout, got %v", opts.HandshakeTimeout)
	}
	if opts.MaxMessageSize <= 0 {
		t.Fatalf("expected fallback max message size, got %d", opts.MaxMessageSize)
	}
}
