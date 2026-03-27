//go:build !no_gin

package gin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
)

type lifecycleState struct {
	mu     sync.Mutex
	server *http.Server
}

// Adapter implements the gin runtime bridge for httpx.
type Adapter struct {
	engine    *gin.Engine
	huma      huma.API
	lifecycle *lifecycleState
}

// New constructs a gin adapter backed by a gin engine and Huma API.
func New(engine *gin.Engine, opts ...adapter.HumaOptions) *Adapter {
	var eng *gin.Engine
	if engine != nil {
		eng = engine
	} else {
		eng = gin.New()
	}

	humaOpts := adapter.MergeHumaOptions(opts...)
	cfg := huma.DefaultConfig(humaOpts.Title, humaOpts.Version)
	adapter.ApplyHumaConfig(&cfg, humaOpts)
	api := humagin.New(eng, cfg)

	return &Adapter{
		engine:    eng,
		huma:      api,
		lifecycle: &lifecycleState{},
	}
}

// Name returns the adapter name.
func (a *Adapter) Name() string {
	return "gin"
}

// Router exposes the underlying gin engine.
func (a *Adapter) Router() *gin.Engine {
	return a.engine
}

// Listen starts related services.
func (a *Adapter) Listen(addr string) error {
	server := a.httpServer(addr)
	release := a.trackServer(server)
	defer release()

	if err := server.ListenAndServe(); err != nil {
		return fmt.Errorf("httpx/gin: listen on %q: %w", addr, err)
	}
	return nil
}

// Shutdown stops the active gin server.
func (a *Adapter) Shutdown() error {
	return a.shutdownContext(context.Background())
}

func (a *Adapter) shutdownContext(ctx context.Context) error {
	server := a.activeServer()
	if server == nil {
		return nil
	}
	if err := server.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("httpx/gin: shutdown: %w", err)
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
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("httpx/gin: listen on %q: %w", addr, err)
	case <-ctx.Done():
		if err := a.shutdownContext(ctx); err != nil {
			return fmt.Errorf("httpx/gin: shutdown on %q: %w", addr, err)
		}
		err := <-errCh
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("httpx/gin: listen on %q: %w", addr, err)
	}
}

// HumaAPI exposes the underlying Huma API.
func (a *Adapter) HumaAPI() huma.API {
	return a.huma
}

func (a *Adapter) httpServer(addr string) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           a.engine.Handler(),
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
