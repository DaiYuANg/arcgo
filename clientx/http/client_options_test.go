package http

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
)

func TestExecuteWithConcurrencyLimitOption(t *testing.T) {
	var active int32
	var maxActive int32

	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		current := atomic.AddInt32(&active, 1)
		for {
			seen := atomic.LoadInt32(&maxActive)
			if current <= seen || atomic.CompareAndSwapInt32(&maxActive, seen, current) {
				break
			}
		}
		time.Sleep(40 * time.Millisecond)
		atomic.AddInt32(&active, -1)
		w.WriteHeader(stdhttp.StatusNoContent)
	}))
	defer srv.Close()

	client, err := New(
		Config{BaseURL: srv.URL, Timeout: 2 * time.Second},
		WithConcurrencyLimit(1),
	)
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	for range 2 {
		go func() {
			defer wg.Done()
			_, execErr := client.Execute(context.Background(), nil, stdhttp.MethodGet, "/health")
			errCh <- execErr
		}()
	}
	wg.Wait()
	close(errCh)

	for execErr := range errCh {
		if execErr != nil {
			t.Fatalf("execute failed: %v", execErr)
		}
	}

	if got := atomic.LoadInt32(&maxActive); got != 1 {
		t.Fatalf("expected max in-flight 1, got %d", got)
	}
}

func TestExecuteWithTimeoutGuardOption(t *testing.T) {
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		time.Sleep(120 * time.Millisecond)
		w.WriteHeader(stdhttp.StatusNoContent)
	}))
	defer srv.Close()

	client, err := New(
		Config{
			BaseURL: srv.URL,
			Timeout: time.Second,
		},
		WithTimeoutGuard(30*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	_, err = client.Execute(context.Background(), nil, stdhttp.MethodGet, "/slow")
	if err == nil {
		t.Fatal("expected timeout guard error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
	if !clientx.IsKind(err, clientx.ErrorKindTimeout) {
		t.Fatalf("expected timeout kind, got %q", clientx.KindOf(err))
	}
}
