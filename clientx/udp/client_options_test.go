package udp

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DaiYuANg/archgo/clientx"
)

func TestNewWithInvalidConfig(t *testing.T) {
	_, err := New(Config{})
	if err == nil {
		t.Fatal("expected config validation error, got nil")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got %v", err)
	}
}

func TestListenPolicyBeforeError(t *testing.T) {
	denyErr := errors.New("deny listen")
	client, err := New(
		Config{Address: "127.0.0.1:0"},
		WithPolicies(clientx.PolicyFuncs{
			BeforeFunc: func(ctx context.Context, operation clientx.Operation) (context.Context, error) {
				if operation.Protocol != clientx.ProtocolUDP || operation.Kind != clientx.OperationKindListen {
					t.Fatalf("unexpected operation: %+v", operation)
				}
				return ctx, denyErr
			},
		}),
	)
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	_, err = client.ListenPacket(context.Background())
	if !errors.Is(err, denyErr) {
		t.Fatalf("expected policy error, got %v", err)
	}
}

func TestDialWithConcurrencyLimitOption(t *testing.T) {
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

	client, err := New(
		Config{Address: "127.0.0.1:1", DialTimeout: 150 * time.Millisecond},
		WithConcurrencyLimit(1),
		WithPolicies(trackingPolicy),
	)
	if err != nil {
		t.Fatalf("new client failed: %v", err)
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

func TestDialWithTimeoutGuardOption(t *testing.T) {
	blockingPolicy := clientx.PolicyFuncs{
		BeforeFunc: func(ctx context.Context, operation clientx.Operation) (context.Context, error) {
			<-ctx.Done()
			return ctx, ctx.Err()
		},
	}

	client, err := New(
		Config{Address: "127.0.0.1:1", DialTimeout: time.Second},
		WithTimeoutGuard(25*time.Millisecond),
		WithPolicies(blockingPolicy),
	)
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	_, err = client.Dial(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}
