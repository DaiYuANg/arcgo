package echo

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humaecho"
	"github.com/labstack/echo/v4"
)

type lifecycleState struct {
	mu     sync.Mutex
	server *http.Server
}

// Adapter implements the echo runtime bridge for httpx.
type Adapter struct {
	engine    *echo.Echo
	huma      huma.API
	lifecycle *lifecycleState
}

// New constructs an echo adapter backed by an echo server and Huma API.
func New(engine *echo.Echo, opts ...adapter.HumaOptions) *Adapter {
	eng := orDefaultEngine(engine)
	humaOpts := adapter.MergeHumaOptions(opts...)
	cfg := huma.DefaultConfig(humaOpts.Title, humaOpts.Version)
	adapter.ApplyHumaConfig(&cfg, humaOpts)
	api := humaecho.New(eng, cfg)

	return &Adapter{
		engine:    eng,
		huma:      api,
		lifecycle: &lifecycleState{},
	}
}

func orDefaultEngine(engine *echo.Echo) *echo.Echo {
	if engine != nil {
		return engine
	}
	return echo.New()
}

// Name returns the adapter name.
func (a *Adapter) Name() string {
	return "echo"
}

// Router exposes the underlying echo engine.
func (a *Adapter) Router() *echo.Echo {
	return a.engine
}

// Listen starts related services.
func (a *Adapter) Listen(addr string) error {
	server := a.httpServer(addr)
	release := a.trackServer(server)
	defer release()

	if err := a.engine.StartServer(server); err != nil {
		return wrapEchoListenError(addr, err)
	}
	return nil
}

// Shutdown stops the active echo server.
func (a *Adapter) Shutdown() error {
	return a.shutdownContext(context.Background())
}

func (a *Adapter) shutdownContext(ctx context.Context) error {
	server := a.activeServer()
	if server == nil {
		return nil
	}
	if err := server.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("httpx/echo: shutdown: %w", err)
	}
	return nil
}

// ListenContext starts related services.
func (a *Adapter) ListenContext(ctx context.Context, addr string) error {
	if ctx == nil {
		return a.Listen(addr)
	}

	server := a.httpServer(addr)
	release := a.trackServer(server)
	defer release()

	errCh := make(chan error, 1)
	go func() {
		errCh <- a.engine.StartServer(server)
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return wrapEchoListenError(addr, err)
	case <-ctx.Done():
		if err := a.shutdownContext(ctx); err != nil {
			return fmt.Errorf("httpx/echo: shutdown on %q: %w", addr, err)
		}
		err := <-errCh
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return wrapEchoListenError(addr, err)
	}
}

// HumaAPI exposes the underlying Huma API.
func (a *Adapter) HumaAPI() huma.API {
	return a.huma
}

func (a *Adapter) httpServer(addr string) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           a.engine,
		ReadHeaderTimeout: 30 * time.Second,
	}
}

func (a *Adapter) trackServer(server *http.Server) func() {
	if a == nil || a.lifecycle == nil {
		return func() {}
	}

	a.lifecycle.mu.Lock()
	a.lifecycle.server = server
	a.lifecycle.mu.Unlock()

	return func() {
		a.lifecycle.mu.Lock()
		if a.lifecycle.server == server {
			a.lifecycle.server = nil
		}
		a.lifecycle.mu.Unlock()
	}
}

func (a *Adapter) activeServer() *http.Server {
	if a == nil || a.lifecycle == nil {
		return nil
	}

	a.lifecycle.mu.Lock()
	defer a.lifecycle.mu.Unlock()
	return a.lifecycle.server
}

func wrapEchoListenError(addr string, err error) error {
	return fmt.Errorf("httpx/echo: listen on %q: %w", addr, err)
}
