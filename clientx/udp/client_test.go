package udp

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/DaiYuANg/arcgo/clientx"
	clientcodec "github.com/DaiYuANg/arcgo/clientx/codec"
)

func TestDialRoundTrip(t *testing.T) {
	server, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen udp server failed: %v", err)
	}
	defer func() { _ = server.Close() }()

	serverErr := make(chan error, 1)
	go func() {
		buf := make([]byte, 128)
		n, addr, err := server.ReadFrom(buf)
		if err != nil {
			serverErr <- err
			return
		}
		_, err = server.WriteTo(append([]byte("ack:"), buf[:n]...), addr)
		serverErr <- err
	}()

	client := New(Config{
		Address:      server.LocalAddr().String(),
		DialTimeout:  time.Second,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	})

	conn, err := client.Dial(context.Background())
	if err != nil {
		t.Fatalf("dial udp failed: %v", err)
	}
	defer func() { _ = conn.Close() }()

	if _, err := conn.Write([]byte("ping")); err != nil {
		t.Fatalf("write udp failed: %v", err)
	}

	buf := make([]byte, 128)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("read udp failed: %v", err)
	}
	if got := string(buf[:n]); got != "ack:ping" {
		t.Fatalf("unexpected response: %q", got)
	}

	if err := <-serverErr; err != nil {
		t.Fatalf("server round trip failed: %v", err)
	}
}

func TestListenPacketReadTimeout(t *testing.T) {
	client := New(Config{
		Address:     "127.0.0.1:0",
		ReadTimeout: 40 * time.Millisecond,
	})

	conn, err := client.ListenPacket(context.Background())
	if err != nil {
		t.Fatalf("listen packet failed: %v", err)
	}
	defer func() { _ = conn.Close() }()

	buf := make([]byte, 16)
	start := time.Now()
	_, _, err = conn.ReadFrom(buf)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	var typedErr *clientx.Error
	if !errors.As(err, &typedErr) {
		t.Fatalf("expected *clientx.Error, got %T", err)
	}
	if typedErr.Protocol != clientx.ProtocolUDP {
		t.Fatalf("expected protocol %q, got %q", clientx.ProtocolUDP, typedErr.Protocol)
	}
	if !clientx.IsKind(err, clientx.ErrorKindTimeout) {
		t.Fatalf("expected kind %q, got %q", clientx.ErrorKindTimeout, clientx.KindOf(err))
	}

	netErr, ok := err.(net.Error)
	if !ok || !netErr.Timeout() {
		t.Fatalf("expected net timeout error, got: %v", err)
	}

	if elapsed := time.Since(start); elapsed < 30*time.Millisecond {
		t.Fatalf("timeout returned too quickly: %v", elapsed)
	}
}

func TestDialCodecRoundTrip(t *testing.T) {
	type payload struct {
		Message string `json:"message"`
	}

	serverClient := New(Config{
		Address:      "127.0.0.1:0",
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	})

	server, err := serverClient.ListenPacketCodec(context.Background(), clientcodec.JSON)
	if err != nil {
		t.Fatalf("listen packet codec failed: %v", err)
	}
	defer func() { _ = server.Close() }()

	serverErr := make(chan error, 1)
	go func() {
		var req payload
		addr, err := server.ReadValueFrom(&req)
		if err != nil {
			serverErr <- err
			return
		}
		serverErr <- server.WriteValueTo(payload{Message: "ack:" + req.Message}, addr)
	}()

	client := New(Config{
		Address:      server.Raw().LocalAddr().String(),
		DialTimeout:  time.Second,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	})

	codecConn, err := client.DialCodec(context.Background(), clientcodec.JSON)
	if err != nil {
		t.Fatalf("dial codec failed: %v", err)
	}
	defer func() { _ = codecConn.Close() }()

	if err := codecConn.WriteValue(payload{Message: "ping"}); err != nil {
		t.Fatalf("write value failed: %v", err)
	}

	var resp payload
	if err := codecConn.ReadValue(&resp); err != nil {
		t.Fatalf("read value failed: %v", err)
	}
	if resp.Message != "ack:ping" {
		t.Fatalf("unexpected response: %+v", resp)
	}

	if err := <-serverErr; err != nil {
		t.Fatalf("server codec failed: %v", err)
	}
}

func TestDialCodecWithNilCodec(t *testing.T) {
	client := New(Config{Address: "127.0.0.1:9000"})
	_, err := client.DialCodec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !clientx.IsKind(err, clientx.ErrorKindCodec) {
		t.Fatalf("expected kind %q, got %q", clientx.ErrorKindCodec, clientx.KindOf(err))
	}
}

func TestListenPacketEmitsIOHook(t *testing.T) {
	var got clientx.IOEvent
	client := New(
		Config{
			Address:     "127.0.0.1:0",
			ReadTimeout: 40 * time.Millisecond,
		},
		WithHooks(clientx.HookFuncs{
			OnIOFunc: func(event clientx.IOEvent) {
				got = event
			},
		}),
	)

	conn, err := client.ListenPacket(context.Background())
	if err != nil {
		t.Fatalf("listen packet failed: %v", err)
	}
	defer func() { _ = conn.Close() }()

	buf := make([]byte, 8)
	_, _, err = conn.ReadFrom(buf)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if got.Protocol != clientx.ProtocolUDP {
		t.Fatalf("expected protocol %q, got %q", clientx.ProtocolUDP, got.Protocol)
	}
	if got.Op != "read_from" {
		t.Fatalf("expected op read_from, got %q", got.Op)
	}
	if got.Err == nil {
		t.Fatal("expected hook error to be set")
	}
}
