package tcp

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/DaiYuANg/archgo/clientx"
	clientcodec "github.com/DaiYuANg/archgo/clientx/codec"
	"github.com/samber/lo"
)

func TestDialErrorIsTyped(t *testing.T) {
	client, err := New(Config{
		Address:     "127.0.0.1:1",
		DialTimeout: 150 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	_, err = client.Dial(context.Background())
	if err == nil {
		t.Fatal("expected dial error, got nil")
	}

	var typedErr *clientx.Error
	if !errors.As(err, &typedErr) {
		t.Fatalf("expected *clientx.Error, got %T", err)
	}
	if typedErr.Protocol != clientx.ProtocolTCP {
		t.Fatalf("expected protocol %q, got %q", clientx.ProtocolTCP, typedErr.Protocol)
	}
	if !lo.Contains([]clientx.ErrorKind{
		clientx.ErrorKindConnRefused,
		clientx.ErrorKindTimeout,
		clientx.ErrorKindNetwork,
	}, typedErr.Kind) {
		t.Fatalf("unexpected error kind: %q", typedErr.Kind)
	}
}

func TestReadTimeoutIsTypedAndStillNetError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp failed: %v", err)
	}
	defer func() { _ = ln.Close() }()

	done := make(chan struct{})
	var doneOnce sync.Once
	closeDone := func() {
		doneOnce.Do(func() { close(done) })
	}
	go func() {
		conn, acceptErr := ln.Accept()
		if acceptErr != nil {
			closeDone()
			return
		}
		defer func() { _ = conn.Close() }()
		<-done
	}()

	client, err := New(Config{
		Address:     ln.Addr().String(),
		DialTimeout: time.Second,
		ReadTimeout: 40 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	conn, err := client.Dial(context.Background())
	if err != nil {
		closeDone()
		t.Fatalf("dial tcp failed: %v", err)
	}
	defer func() { _ = conn.Close() }()
	defer closeDone()

	buf := make([]byte, 8)
	_, err = conn.Read(buf)
	if err == nil {
		t.Fatal("expected read timeout error, got nil")
	}
	if !clientx.IsKind(err, clientx.ErrorKindTimeout) {
		t.Fatalf("expected kind %q, got %q", clientx.ErrorKindTimeout, clientx.KindOf(err))
	}

	netErr, ok := err.(net.Error)
	if !ok || !netErr.Timeout() {
		t.Fatalf("expected timeout net.Error, got %v", err)
	}
}

func TestDialCodecRoundTrip(t *testing.T) {
	type payload struct {
		Message string `json:"message"`
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp failed: %v", err)
	}
	defer func() { _ = ln.Close() }()

	serverErr := make(chan error, 1)
	go func() {
		conn, acceptErr := ln.Accept()
		if acceptErr != nil {
			serverErr <- acceptErr
			return
		}
		defer func() { _ = conn.Close() }()

		cc := NewCodecConn(conn, clientcodec.JSON, clientcodec.NewLengthPrefixed(1024), ln.Addr().String())
		var req payload
		if err := cc.ReadValue(&req); err != nil {
			serverErr <- err
			return
		}
		serverErr <- cc.WriteValue(payload{Message: "ack:" + req.Message})
	}()

	client, err := New(Config{
		Address:      ln.Addr().String(),
		DialTimeout:  time.Second,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	cc, err := client.DialCodec(context.Background(), clientcodec.JSON, clientcodec.NewLengthPrefixed(1024))
	if err != nil {
		t.Fatalf("dial codec failed: %v", err)
	}
	defer func() { _ = cc.Close() }()

	if err := cc.WriteValue(payload{Message: "ping"}); err != nil {
		t.Fatalf("client write value failed: %v", err)
	}

	var resp payload
	if err := cc.ReadValue(&resp); err != nil {
		t.Fatalf("client read value failed: %v", err)
	}
	if resp.Message != "ack:ping" {
		t.Fatalf("unexpected response: %+v", resp)
	}

	if err := <-serverErr; err != nil {
		t.Fatalf("server failed: %v", err)
	}
}

func TestDialCodecWithNilCodec(t *testing.T) {
	client, err := New(Config{Address: "127.0.0.1:9000"})
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	_, err = client.DialCodec(context.Background(), nil, clientcodec.NewLengthPrefixed(1024))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !clientx.IsKind(err, clientx.ErrorKindCodec) {
		t.Fatalf("expected kind %q, got %q", clientx.ErrorKindCodec, clientx.KindOf(err))
	}
}

func TestDialEmitsHookOnError(t *testing.T) {
	var got clientx.DialEvent
	client, err := New(
		Config{
			Address:     "127.0.0.1:1",
			DialTimeout: 100 * time.Millisecond,
		},
		WithHooks(clientx.HookFuncs{
			OnDialFunc: func(event clientx.DialEvent) {
				got = event
			},
		}),
	)
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	_, err = client.Dial(context.Background())
	if err == nil {
		t.Fatal("expected dial error, got nil")
	}
	if got.Protocol != clientx.ProtocolTCP {
		t.Fatalf("expected protocol %q, got %q", clientx.ProtocolTCP, got.Protocol)
	}
	if got.Op != "dial" {
		t.Fatalf("expected op dial, got %q", got.Op)
	}
	if got.Err == nil {
		t.Fatal("expected hook error to be set")
	}
}

func TestNewWithInvalidConfig(t *testing.T) {
	_, err := New(Config{})
	if err == nil {
		t.Fatal("expected config validation error, got nil")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got %v", err)
	}
}

func TestDialPolicyBeforeError(t *testing.T) {
	denyErr := errors.New("deny dial")
	client, err := New(
		Config{Address: "127.0.0.1:1"},
		WithPolicies(clientx.PolicyFuncs{
			BeforeFunc: func(ctx context.Context, operation clientx.Operation) (context.Context, error) {
				if operation.Protocol != clientx.ProtocolTCP || operation.Kind != clientx.OperationKindDial {
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

	_, err = client.Dial(context.Background())
	if !errors.Is(err, denyErr) {
		t.Fatalf("expected policy error, got %v", err)
	}
}
