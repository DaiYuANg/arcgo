package fiber

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Listen starts the fiber server.
func (a *Adapter) Listen(addr string) error {
	if err := a.app.Listen(addr); err != nil {
		return fmt.Errorf("httpx/fiber: listen on %q: %w", addr, err)
	}
	return nil
}

// Shutdown stops the fiber server.
func (a *Adapter) Shutdown() error {
	if err := a.app.Shutdown(); err != nil {
		return fmt.Errorf("httpx/fiber: shutdown: %w", err)
	}
	return nil
}

// ListenContext starts related services.
func (a *Adapter) ListenContext(ctx context.Context, addr string) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- a.Listen(addr)
	}()

	select {
	case err := <-errCh:
		if isExpectedFiberClose(err) {
			return nil
		}
		return fmt.Errorf("httpx/fiber: listen on %q: %w", addr, err)
	case <-ctx.Done():
		shutdownErr := a.shutdown()
		listenErr := <-errCh
		if shutdownErr != nil {
			return fmt.Errorf("httpx/fiber: shutdown on %q: %w", addr, shutdownErr)
		}
		if isExpectedFiberClose(listenErr) {
			return nil
		}
		return fmt.Errorf("httpx/fiber: listen on %q: %w", addr, listenErr)
	}
}

func (a *Adapter) shutdown() error {
	if a.opts.ShutdownTimeout > 0 {
		if err := a.app.ShutdownWithTimeout(a.opts.ShutdownTimeout); err != nil {
			return fmt.Errorf("httpx/fiber: shutdown: %w", err)
		}
		return nil
	}
	return a.Shutdown()
}

func isExpectedFiberClose(err error) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, http.ErrServerClosed) {
		return true
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "server is not running") ||
		strings.Contains(lower, "use of closed network connection")
}
