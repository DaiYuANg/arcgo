package http

import (
	"context"
	"errors"
	stdhttp "net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DaiYuANg/archgo/clientx"
	"github.com/samber/lo"
	"resty.dev/v3"
)

func TestExecuteWithNilRequest(t *testing.T) {
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusNoContent)
	}))
	defer srv.Close()

	client, err := New(Config{
		BaseURL: srv.URL,
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	resp, err := client.Execute(context.Background(), nil, stdhttp.MethodGet, "/health")
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if resp.StatusCode() != stdhttp.StatusNoContent {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusNoContent, resp.StatusCode())
	}
}

func TestExecuteWrapsTransportError(t *testing.T) {
	client, err := New(Config{
		BaseURL: "http://127.0.0.1:1",
		Timeout: 150 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	_, err = client.Execute(context.Background(), client.R(), stdhttp.MethodGet, "")
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
	client, err := New(
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
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	_, err = client.Execute(context.Background(), nil, stdhttp.MethodGet, "/health")
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

func TestNewWithInvalidBaseURL(t *testing.T) {
	_, err := New(Config{BaseURL: "://bad"})
	if err == nil {
		t.Fatal("expected config validation error, got nil")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got %v", err)
	}
}

func TestExecuteAppliesPolicies(t *testing.T) {
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusNoContent)
	}))
	defer srv.Close()

	calls := make([]string, 0, 3)
	client, err := New(
		Config{BaseURL: srv.URL, Timeout: time.Second},
		WithPolicies(clientx.PolicyFuncs{
			BeforeFunc: func(ctx context.Context, operation clientx.Operation) (context.Context, error) {
				calls = append(calls, "before")
				if operation.Protocol != clientx.ProtocolHTTP || operation.Kind != clientx.OperationKindRequest {
					t.Fatalf("unexpected operation: %+v", operation)
				}
				return ctx, nil
			},
			AfterFunc: func(ctx context.Context, operation clientx.Operation, err error) error {
				calls = append(calls, "after")
				return nil
			},
		}),
	)
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	_, err = client.Execute(context.Background(), nil, stdhttp.MethodGet, "/health")
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if !reflect.DeepEqual(calls, []string{"before", "after"}) {
		t.Fatalf("unexpected policy calls: %v", calls)
	}
}

func TestExecuteRetriesFromConfig(t *testing.T) {
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusNoContent)
	}))
	defer srv.Close()

	var attempts int32
	client, err := New(
		Config{
			BaseURL: srv.URL,
			Timeout: time.Second,
			Retry: clientx.RetryConfig{
				Enabled:    true,
				MaxRetries: 2,
				WaitMin:    time.Millisecond,
				WaitMax:    2 * time.Millisecond,
			},
		},
		WithRequestMiddleware(func(_ *resty.Client, _ *resty.Request) error {
			if atomic.AddInt32(&attempts, 1) < 3 {
				return context.DeadlineExceeded
			}
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	resp, err := client.Execute(context.Background(), nil, stdhttp.MethodGet, "/health")
	if err != nil {
		t.Fatalf("execute with retry failed: %v", err)
	}
	if resp.StatusCode() != stdhttp.StatusNoContent {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusNoContent, resp.StatusCode())
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
}
