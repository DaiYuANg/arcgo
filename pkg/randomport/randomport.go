// Package randomport provides utilities for finding available random ports.
package randomport

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/DaiYuANg/arcgo/collectionx"
)

var (
	// usedPorts tracks ports that have been allocated during the current process.
	usedPorts = collectionx.NewConcurrentSet[int]()
)

const maxFindAttempts = 50

// Find returns a random available port that is not currently in use.
// It checks both TCP port availability and tracks previously allocated ports
// to avoid conflicts when multiple servers are started in the same process.
func Find() (int, error) {
	ctx := context.Background()
	var lastErr error

	// Try up to 50 times to find an available port
	for range maxFindAttempts {
		port, err := findAvailablePort(ctx)
		if err != nil {
			lastErr = err
			continue
		}
		if usedPorts.AddIfAbsent(port) {
			return port, nil
		}
	}

	if lastErr != nil {
		return 0, fmt.Errorf("randomport: failed to find available port after %d attempts: %w", maxFindAttempts, lastErr)
	}

	return 0, errors.New("randomport: failed to find available port after 50 attempts")
}

// findAvailablePort finds a single available port by listening on port 0.
func findAvailablePort(ctx context.Context) (port int, err error) {
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("listen for random port: %w", err)
	}
	defer func() {
		closeErr := listener.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("close random port listener: %w", closeErr)
		}
	}()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("randomport: unexpected listener address type %T", listener.Addr())
	}

	return addr.Port, nil
}

// Release releases a port back to the available pool.
// This is primarily useful for testing scenarios.
func Release(port int) {
	usedPorts.Remove(port)
}

// MustFind returns a random available port or panics if none can be found.
func MustFind() int {
	port, err := Find()
	if err != nil {
		panic(err)
	}

	return port
}
