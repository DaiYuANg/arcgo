package randomport_test

import (
	"context"
	"net"
	"testing"

	"github.com/DaiYuANg/arcgo/pkg/randomport"
)

func TestFind(t *testing.T) {
	skipIfLoopbackListenUnavailable(t)

	port, err := randomport.Find()
	if err != nil {
		t.Fatalf("Find() returned error: %v", err)
	}
	if port <= 0 {
		t.Fatalf("Find() returned invalid port: %d", port)
	}

	// The found port should be in valid range
	if port < 1 || port > 65535 {
		t.Fatalf("Find() returned port out of range: %d", port)
	}
}

func TestMustFind(t *testing.T) {
	skipIfLoopbackListenUnavailable(t)

	port := randomport.MustFind()
	if port <= 0 {
		t.Fatalf("MustFind() returned invalid port: %d", port)
	}
}

func TestFindMultiple(t *testing.T) {
	skipIfLoopbackListenUnavailable(t)

	ports := make(map[int]bool)
	for i := range 10 {
		port, err := randomport.Find()
		if err != nil {
			t.Fatalf("Find() iteration %d returned error: %v", i, err)
		}
		if ports[port] {
			t.Fatalf("Find() returned duplicate port: %d", port)
		}
		ports[port] = true
	}
}

func TestRelease(t *testing.T) {
	skipIfLoopbackListenUnavailable(t)

	port, err := randomport.Find()
	if err != nil {
		t.Fatalf("Find() returned error: %v", err)
	}

	randomport.Release(port)

	// After release, we should be able to find ports again
	newPort, err := randomport.Find()
	if err != nil {
		t.Fatalf("Find() after release returned error: %v", err)
	}
	if newPort <= 0 {
		t.Fatalf("Find() after release returned invalid port: %d", newPort)
	}
}

func skipIfLoopbackListenUnavailable(t *testing.T) {
	t.Helper()

	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("loopback listen unavailable in this environment: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := listener.Close(); closeErr != nil {
			t.Errorf("close loopback test listener: %v", closeErr)
		}
	})
}
