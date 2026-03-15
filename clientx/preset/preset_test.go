package preset

import (
	"context"
	"errors"
	stdhttp "net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DaiYuANg/archgo/clientx"
	clienthttp "github.com/DaiYuANg/archgo/clientx/http"
	clienttcp "github.com/DaiYuANg/archgo/clientx/tcp"
	clientudp "github.com/DaiYuANg/archgo/clientx/udp"
	"resty.dev/v3"
)

func TestNewEdgeHTTPRetryPreset(t *testing.T) {
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusNoContent)
	}))
	defer srv.Close()

	var attempts int32
	client, err := NewEdgeHTTP(
		clienthttp.Config{BaseURL: srv.URL},
		WithEdgeHTTPRetry(clientx.RetryConfig{
			Enabled:    true,
			MaxRetries: 2,
			WaitMin:    time.Millisecond,
			WaitMax:    2 * time.Millisecond,
		}),
		WithEdgeHTTPOption(clienthttp.WithRequestMiddleware(func(_ *resty.Client, _ *resty.Request) error {
			if atomic.AddInt32(&attempts, 1) < 3 {
				return context.DeadlineExceeded
			}
			return nil
		})),
	)
	if err != nil {
		t.Fatalf("new edge http client failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	resp, err := client.Execute(context.Background(), nil, stdhttp.MethodGet, "/health")
	if err != nil {
		t.Fatalf("execute failed: %v (attempts=%d)", err, atomic.LoadInt32(&attempts))
	}
	if resp.StatusCode() != stdhttp.StatusNoContent {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusNoContent, resp.StatusCode())
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
}

func TestNewEdgeHTTPTimeoutGuard(t *testing.T) {
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		time.Sleep(120 * time.Millisecond)
		w.WriteHeader(stdhttp.StatusNoContent)
	}))
	defer srv.Close()

	client, err := NewEdgeHTTP(
		clienthttp.Config{BaseURL: srv.URL},
		WithEdgeHTTPTimeout(2*time.Second),
		WithEdgeHTTPTimeoutGuard(25*time.Millisecond),
		WithEdgeHTTPDisableRetry(),
	)
	if err != nil {
		t.Fatalf("new edge http client failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	_, err = client.Execute(context.Background(), nil, stdhttp.MethodGet, "/slow")
	if err == nil {
		t.Fatal("expected timeout guard error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}

func TestNewInternalRPCTimeoutGuard(t *testing.T) {
	blockingPolicy := clientx.PolicyFuncs{
		BeforeFunc: func(ctx context.Context, operation clientx.Operation) (context.Context, error) {
			<-ctx.Done()
			return ctx, ctx.Err()
		},
	}

	client, err := NewInternalRPC(
		clienttcp.Config{Address: "127.0.0.1:1"},
		WithInternalRPCDisableRetry(),
		WithInternalRPCTimeoutGuard(25*time.Millisecond),
		WithInternalRPCOption(clienttcp.WithPolicies(blockingPolicy)),
	)
	if err != nil {
		t.Fatalf("new internal rpc client failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	_, err = client.Dial(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}

func TestNewLowLatencyUDPConcurrencyLimit(t *testing.T) {
	var active int32
	var maxActive int32
	trackingPolicy := clientx.PolicyFuncs{
		BeforeFunc: func(ctx context.Context, operation clientx.Operation) (context.Context, error) {
			current := atomic.AddInt32(&active, 1)
			for {
				seen := atomic.LoadInt32(&maxActive)
				if current <= seen || atomic.CompareAndSwapInt32(&maxActive, seen, current) {
					break
				}
			}
			time.Sleep(30 * time.Millisecond)
			return ctx, nil
		},
		AfterFunc: func(ctx context.Context, operation clientx.Operation, err error) error {
			atomic.AddInt32(&active, -1)
			return nil
		},
	}

	client, err := NewLowLatencyUDP(
		clientudp.Config{Address: "127.0.0.1:1"},
		WithLowLatencyUDPConcurrencyLimit(1),
		WithLowLatencyUDPOption(clientudp.WithPolicies(trackingPolicy)),
	)
	if err != nil {
		t.Fatalf("new low latency udp client failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	var wg sync.WaitGroup
	wg.Add(2)
	for range 2 {
		go func() {
			defer wg.Done()
			_, _ = client.Dial(context.Background())
		}()
	}
	wg.Wait()

	if got := atomic.LoadInt32(&maxActive); got != 1 {
		t.Fatalf("expected max in-flight 1, got %d", got)
	}
}
