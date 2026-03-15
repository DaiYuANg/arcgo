// Package randomport provides utilities for finding available random ports.
package randomport

import (
	"fmt"
	"net"

	"github.com/DaiYuANg/archgo/collectionx"
)

var (
	// usedPorts tracks ports that have been allocated during the current process.
	usedPorts = collectionx.NewConcurrentSet[int]()
)

// Find returns a random available port that is not currently in use.
// It checks both TCP port availability and tracks previously allocated ports
// to avoid conflicts when multiple servers are started in the same process.
func Find() (int, error) {
	// Try up to 50 times to find an available port
	for i := 0; i < 50; i++ {
		port, err := findAvailablePort()
		if err != nil {
			continue
		}
		if usedPorts.AddIfAbsent(port) {
			return port, nil
		}
	}

	return 0, fmt.Errorf("randomport: failed to find available port after 50 attempts")
}

// findAvailablePort finds a single available port by listening on port 0.
func findAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = listener.Close() }()

	addr := listener.Addr().(*net.TCPAddr)
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
