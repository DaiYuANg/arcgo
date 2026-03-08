package http

import (
	"errors"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DaiYuANg/arcgo/clientx"
	"github.com/samber/lo"
)

func TestExecuteWithNilRequest(t *testing.T) {
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusNoContent)
	}))
	defer srv.Close()

	client := New(Config{
		BaseURL: srv.URL,
		Timeout: time.Second,
	})

	resp, err := client.Execute(nil, stdhttp.MethodGet, "/health")
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if resp.StatusCode() != stdhttp.StatusNoContent {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusNoContent, resp.StatusCode())
	}
}

func TestExecuteWrapsTransportError(t *testing.T) {
	client := New(Config{
		BaseURL: "http://127.0.0.1:1",
		Timeout: 150 * time.Millisecond,
	})

	_, err := client.Execute(client.R(), stdhttp.MethodGet, "")
	if err == nil {
		t.Fatal("expected transport error, got nil")
	}
	var typedErr *clientx.Error
	if !errors.As(err, &typedErr) {
		t.Fatalf("expected *clientx.Error, got %T", err)
	}
	if typedErr.Protocol != clientx.ProtocolHTTP {
		t.Fatalf("expected protocol %q, got %q", clientx.ProtocolHTTP, typedErr.Protocol)
	}
	if typedErr.Op != "get" {
		t.Fatalf("expected op get, got %q", typedErr.Op)
	}
	if !lo.Contains([]clientx.ErrorKind{
		clientx.ErrorKindConnRefused,
		clientx.ErrorKindTimeout,
		clientx.ErrorKindNetwork,
	}, clientx.KindOf(err)) {
		t.Fatalf("unexpected error kind: %q", clientx.KindOf(err))
	}
}

func TestExecuteEmitsHook(t *testing.T) {
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	var got clientx.IOEvent
	client := New(
		Config{
			BaseURL: srv.URL,
			Timeout: time.Second,
		},
		WithHooks(clientx.HookFuncs{
			OnIOFunc: func(event clientx.IOEvent) {
				got = event
			},
		}),
	)

	_, err := client.Execute(nil, stdhttp.MethodGet, "/health")
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if got.Protocol != clientx.ProtocolHTTP {
		t.Fatalf("expected protocol %q, got %q", clientx.ProtocolHTTP, got.Protocol)
	}
	if got.Op != "get" {
		t.Fatalf("expected op get, got %q", got.Op)
	}
	if got.Bytes == 0 {
		t.Fatalf("expected response bytes > 0, got %d", got.Bytes)
	}
}
