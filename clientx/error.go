package clientx

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"

	"github.com/samber/lo"
)

type Protocol string

const (
	ProtocolUnknown Protocol = "unknown"
	ProtocolHTTP    Protocol = "http"
	ProtocolTCP     Protocol = "tcp"
	ProtocolUDP     Protocol = "udp"
)

type ErrorKind string

const (
	ErrorKindUnknown     ErrorKind = "unknown"
	ErrorKindCanceled    ErrorKind = "canceled"
	ErrorKindTimeout     ErrorKind = "timeout"
	ErrorKindTemporary   ErrorKind = "temporary"
	ErrorKindConnRefused ErrorKind = "conn_refused"
	ErrorKindDNS         ErrorKind = "dns"
	ErrorKindTLS         ErrorKind = "tls"
	ErrorKindClosed      ErrorKind = "closed"
	ErrorKindNetwork     ErrorKind = "network"
	ErrorKindCodec       ErrorKind = "codec"
)

type Error struct {
	Protocol Protocol
	Op       string
	Addr     string
	Kind     ErrorKind
	Err      error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return fmt.Sprintf("%s %s %s (%s)", e.Protocol, e.Op, e.Addr, e.Kind)
	}
	if e.Addr != "" {
		return fmt.Sprintf("%s %s %s (%s): %v", e.Protocol, e.Op, e.Addr, e.Kind, e.Err)
	}
	return fmt.Sprintf("%s %s (%s): %v", e.Protocol, e.Op, e.Kind, e.Err)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *Error) Timeout() bool {
	if e == nil {
		return false
	}
	if e.Kind == ErrorKindTimeout {
		return true
	}
	var netErr net.Error
	return errors.As(e.Err, &netErr) && netErr.Timeout()
}

func (e *Error) Temporary() bool {
	if e == nil {
		return false
	}
	if e.Kind == ErrorKindTemporary {
		return true
	}
	// net.Error.Temporary() 已弃用，这里仅检查 Kind 标记
	return false
}

func WrapError(protocol Protocol, op, addr string, err error) error {
	return WrapErrorWithKind(protocol, op, addr, "", err)
}

func WrapErrorWithKind(protocol Protocol, op, addr string, kind ErrorKind, err error) error {
	if err == nil {
		return nil
	}
	if _, ok := errors.AsType[*Error](err); ok {
		return err
	}
	if protocol == "" {
		protocol = ProtocolUnknown
	}
	return &Error{
		Protocol: protocol,
		Op:       op,
		Addr:     addr,
		Kind:     lo.Ternary(kind != "", kind, classifyErrorKind(err)),
		Err:      err,
	}
}

func IsKind(err error, kind ErrorKind) bool {
	var e *Error
	if !errors.As(err, &e) {
		return false
	}
	return e.Kind == kind
}

func KindOf(err error) ErrorKind {
	var e *Error
	if !errors.As(err, &e) {
		return ErrorKindUnknown
	}
	return e.Kind
}

func classifyErrorKind(err error) ErrorKind {
	if err == nil {
		return ErrorKindUnknown
	}
	if errors.Is(err, context.Canceled) {
		return ErrorKindCanceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrorKindTimeout
	}
	if errors.Is(err, net.ErrClosed) || errors.Is(err, os.ErrClosed) {
		return ErrorKindClosed
	}

	if _, ok := errors.AsType[*net.DNSError](err); ok {
		return ErrorKindDNS
	}

	if netErr, ok := errors.AsType[net.Error](err); ok {
		if netErr.Timeout() {
			return ErrorKindTimeout
		}
		// netErr.Temporary() 已弃用，不再使用
	}

	if opErr, ok := errors.AsType[*net.OpError](err); ok && opErr.Err != nil {
		if isConnRefused(opErr.Err) {
			return ErrorKindConnRefused
		}
	}
	if isConnRefused(err) {
		return ErrorKindConnRefused
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "tls"), strings.Contains(msg, "x509"), strings.Contains(msg, "certificate"):
		return ErrorKindTLS
	case strings.Contains(msg, "use of closed network connection"), strings.Contains(msg, "file already closed"):
		return ErrorKindClosed
	case strings.Contains(msg, "network"):
		return ErrorKindNetwork
	default:
		return ErrorKindUnknown
	}
}

func isConnRefused(err error) bool {
	if errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}
	if errno, ok := errors.AsType[syscall.Errno](err); ok {
		return lo.Contains([]syscall.Errno{syscall.ECONNREFUSED, syscall.Errno(10061)}, errno)
	}
	if sysErr, ok := errors.AsType[*os.SyscallError](err); ok {
		return isConnRefused(sysErr.Err)
	}
	return false
}
