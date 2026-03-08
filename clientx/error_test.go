package clientx

import (
	"context"
	"errors"
	"net"
	"os"
	"syscall"
	"testing"
)

type netErrMock struct {
	timeout   bool
	temporary bool
}

func (e netErrMock) Error() string {
	return "net error mock"
}

func (e netErrMock) Timeout() bool {
	return e.timeout
}

func (e netErrMock) Temporary() bool {
	return e.temporary
}

func TestWrapErrorNil(t *testing.T) {
	if got := WrapError(ProtocolTCP, "dial", "127.0.0.1:8080", nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestWrapErrorCanceled(t *testing.T) {
	err := WrapError(ProtocolTCP, "dial", "127.0.0.1:8080", context.Canceled)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsKind(err, ErrorKindCanceled) {
		t.Fatalf("expected kind %q, got %q", ErrorKindCanceled, KindOf(err))
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected wrapped canceled error, got %v", err)
	}
}

func TestWrapErrorConnRefused(t *testing.T) {
	baseErr := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: &os.SyscallError{Err: syscall.ECONNREFUSED},
	}
	err := WrapError(ProtocolTCP, "dial", "127.0.0.1:1", baseErr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsKind(err, ErrorKindConnRefused) {
		t.Fatalf("expected kind %q, got %q", ErrorKindConnRefused, KindOf(err))
	}
}

func TestWrapErrorStillImplementsNetError(t *testing.T) {
	err := WrapError(ProtocolUDP, "read", "127.0.0.1:0", netErrMock{timeout: true})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	netErr, ok := err.(net.Error)
	if !ok {
		t.Fatalf("expected net.Error, got %T", err)
	}
	if !netErr.Timeout() {
		t.Fatal("expected timeout=true")
	}
	if !IsKind(err, ErrorKindTimeout) {
		t.Fatalf("expected kind %q, got %q", ErrorKindTimeout, KindOf(err))
	}
}

func TestWrapErrorIdempotent(t *testing.T) {
	once := WrapError(ProtocolHTTP, "get", "http://127.0.0.1:1", context.DeadlineExceeded)
	twice := WrapError(ProtocolHTTP, "get", "http://127.0.0.1:1", once)
	if once != twice {
		t.Fatal("expected wrapped error to stay unchanged on second wrap")
	}
}
