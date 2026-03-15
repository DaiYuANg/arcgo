//go:build !no_echo

package echo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/DaiYuANg/archgo/httpx/adapter"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humaecho"
	"github.com/labstack/echo/v4"
)

// Adapter implements the echo runtime bridge for httpx.
type Adapter struct {
	engine *echo.Echo
	group  *echo.Group
	logger *slog.Logger
	huma   huma.API
	docs   *adapter.DocsController
	server ServerOptions
}

// New constructs an echo adapter backed by an echo server and Huma API.
func New(engine *echo.Echo, opts ...adapter.HumaOptions) *Adapter {
	options := DefaultOptions()
	options.Huma = adapter.MergeHumaOptions(opts...)
	return NewWithOptions(engine, options)
}

// ServerOptions configures the echo adapter's underlying http.Server.
type ServerOptions struct {
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	MaxHeaderBytes  int
}

// DefaultServerOptions returns the default echo adapter server config.
func DefaultServerOptions() ServerOptions {
	return ServerOptions{
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    15 * time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownTimeout: 5 * time.Second,
		MaxHeaderBytes:  1 << 20,
	}
}

// Options configures echo adapter construction.
type Options struct {
	Huma   adapter.HumaOptions
	Logger *slog.Logger
	Server ServerOptions
}

// DefaultOptions returns the default echo adapter config.
func DefaultOptions() Options {
	return Options{
		Huma:   adapter.DefaultHumaOptions(),
		Logger: slog.Default(),
		Server: DefaultServerOptions(),
	}
}

// NewWithOptions constructs an echo adapter from explicit construction-time options.
func NewWithOptions(engine *echo.Echo, opts Options) *Adapter {
	var eng *echo.Echo
	if engine != nil {
		eng = engine
	} else {
		eng = echo.New()
	}

	humaOpts := adapter.MergeHumaOptions(opts.Huma)
	cfg := huma.DefaultConfig(humaOpts.Title, humaOpts.Version)
	adapter.ApplyHumaConfig(&cfg, humaOpts)

	docsCfg := cfg
	docsCfg.DocsPath = ""
	docsCfg.OpenAPIPath = ""
	docsCfg.SchemasPath = ""
	api := humaecho.New(eng, docsCfg)
	docs := adapter.NewDocsController(api, humaOpts)
	eng.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if docs.ServeHTTP(c.Response(), c.Request()) {
				return nil
			}
			return next(c)
		}
	})

	return &Adapter{
		engine: eng,
		group:  nil,
		logger: defaultLogger(opts.Logger),
		huma:   api,
		docs:   docs,
		server: mergeServerOptions(opts.Server),
	}
}

// WithLogger replaces the adapter logger.
func (a *Adapter) WithLogger(logger *slog.Logger) *Adapter {
	a.SetLogger(logger)
	return a
}

// SetLogger replaces the adapter logger.
func (a *Adapter) SetLogger(logger *slog.Logger) {
	if a == nil || logger == nil {
		return
	}
	a.logger = logger
}

// Name returns the adapter name.
func (a *Adapter) Name() string {
	return "echo"
}

// Handle registers a native handler on the echo app or group.
func (a *Adapter) Handle(method, path string, handler adapter.HandlerFunc) {
	if a.group != nil {
		a.group.Add(method, path, a.echoHandler(handler))
	} else {
		a.engine.Add(method, path, a.echoHandler(handler))
	}
}

// Group returns a child adapter scoped to an echo group.
func (a *Adapter) Group(prefix string) adapter.Adapter {
	var g *echo.Group
	if a.group != nil {
		g = a.group.Group(prefix)
	} else {
		g = a.engine.Group(prefix)
	}

	return &Adapter{
		engine: a.engine,
		group:  g,
		logger: a.logger,
		huma:   a.huma,
		docs:   a.docs,
		server: a.server,
	}
}

// ServeHTTP delegates request handling to the echo engine.
func (a *Adapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.engine.ServeHTTP(w, r)
}

// Router exposes the underlying echo engine.
func (a *Adapter) Router() *echo.Echo {
	return a.engine
}

// Listen starts related services.
func (a *Adapter) Listen(addr string) error {
	if err := a.engine.StartServer(a.httpServer(addr)); err != nil {
		return fmt.Errorf("httpx/echo: listen on %q: %w", addr, err)
	}
	return nil
}

// ListenContext starts related services.
func (a *Adapter) ListenContext(ctx context.Context, addr string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- a.engine.StartServer(a.httpServer(addr))
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("httpx/echo: listen on %q: %w", addr, err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.server.ShutdownTimeout)
		defer cancel()
		if err := a.engine.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("httpx/echo: shutdown on %q: %w", addr, err)
		}
		err := <-errCh
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("httpx/echo: listen on %q: %w", addr, err)
	}
}

// echoHandler wraps related logic.
func (a *Adapter) echoHandler(handler adapter.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		params := make(map[string]string, len(c.ParamNames()))
		for _, name := range c.ParamNames() {
			params[name] = c.Param(name)
		}

		req := c.Request().WithContext(adapter.WithRouteParams(c.Request().Context(), params))
		if err := handler(req.Context(), c.Response(), req); err != nil {
			a.logger.Error("Handler error",
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.String("error", err.Error()),
			)
			return fmt.Errorf("httpx/echo: handler failed: %w", err)
		}
		return nil
	}
}

// HumaAPI exposes the underlying Huma API.
func (a *Adapter) HumaAPI() huma.API {
	return a.huma
}

// ConfigureHumaOptions updates adapter-managed docs/openapi routing.
func (a *Adapter) ConfigureHumaOptions(opts adapter.HumaOptions) {
	if a == nil || a.docs == nil {
		return
	}
	a.docs.Configure(opts)
}

func (a *Adapter) httpServer(addr string) *http.Server {
	return &http.Server{
		Addr:           addr,
		Handler:        a.engine,
		ReadTimeout:    a.server.ReadTimeout,
		WriteTimeout:   a.server.WriteTimeout,
		IdleTimeout:    a.server.IdleTimeout,
		MaxHeaderBytes: a.server.MaxHeaderBytes,
	}
}

func defaultLogger(logger *slog.Logger) *slog.Logger {
	if logger != nil {
		return logger
	}
	return slog.Default()
}

func mergeServerOptions(opts ServerOptions) ServerOptions {
	defaults := DefaultServerOptions()
	if opts.ReadTimeout > 0 {
		defaults.ReadTimeout = opts.ReadTimeout
	}
	if opts.WriteTimeout > 0 {
		defaults.WriteTimeout = opts.WriteTimeout
	}
	if opts.IdleTimeout > 0 {
		defaults.IdleTimeout = opts.IdleTimeout
	}
	if opts.ShutdownTimeout > 0 {
		defaults.ShutdownTimeout = opts.ShutdownTimeout
	}
	if opts.MaxHeaderBytes > 0 {
		defaults.MaxHeaderBytes = opts.MaxHeaderBytes
	}
	return defaults
}
