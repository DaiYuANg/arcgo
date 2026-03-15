//go:build !no_gin

package gin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/DaiYuANg/archgo/httpx/adapter"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
)

// Adapter implements the gin runtime bridge for httpx.
type Adapter struct {
	engine *gin.Engine
	group  *gin.RouterGroup
	logger *slog.Logger
	huma   huma.API
	docs   *adapter.DocsController
	server ServerOptions
}

// New constructs a gin adapter backed by a gin engine and Huma API.
func New(engine *gin.Engine, opts ...adapter.HumaOptions) *Adapter {
	options := DefaultOptions()
	options.Huma = adapter.MergeHumaOptions(opts...)
	return NewWithOptions(engine, options)
}

// ServerOptions configures the gin adapter's underlying http.Server.
type ServerOptions struct {
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	MaxHeaderBytes  int
}

// DefaultServerOptions returns the default gin adapter server config.
func DefaultServerOptions() ServerOptions {
	return ServerOptions{
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    15 * time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownTimeout: 5 * time.Second,
		MaxHeaderBytes:  1 << 20,
	}
}

// Options configures gin adapter construction.
type Options struct {
	Huma   adapter.HumaOptions
	Logger *slog.Logger
	Server ServerOptions
}

// DefaultOptions returns the default gin adapter config.
func DefaultOptions() Options {
	return Options{
		Huma:   adapter.DefaultHumaOptions(),
		Logger: slog.Default(),
		Server: DefaultServerOptions(),
	}
}

// NewWithOptions constructs a gin adapter from explicit construction-time options.
func NewWithOptions(engine *gin.Engine, opts Options) *Adapter {
	var eng *gin.Engine
	if engine != nil {
		eng = engine
	} else {
		eng = gin.New()
	}

	humaOpts := adapter.MergeHumaOptions(opts.Huma)
	cfg := huma.DefaultConfig(humaOpts.Title, humaOpts.Version)
	adapter.ApplyHumaConfig(&cfg, humaOpts)

	docsCfg := cfg
	docsCfg.DocsPath = ""
	docsCfg.OpenAPIPath = ""
	docsCfg.SchemasPath = ""
	api := humagin.New(eng, docsCfg)
	docs := adapter.NewDocsController(api, humaOpts)
	eng.Use(func(c *gin.Context) {
		if docs.ServeHTTP(c.Writer, c.Request) {
			c.Abort()
			return
		}
		c.Next()
	})

	return &Adapter{
		engine: eng,
		group:  &eng.RouterGroup,
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
	return "gin"
}

// Handle registers a native handler on the gin router group.
func (a *Adapter) Handle(method, path string, handler adapter.HandlerFunc) {
	a.group.Handle(method, path, a.wrapHandler(handler))
}

// Group returns a child adapter scoped to a gin router group.
func (a *Adapter) Group(prefix string) adapter.Adapter {
	return &Adapter{
		engine: a.engine,
		group:  a.group.Group(prefix),
		logger: a.logger,
		huma:   a.huma,
		docs:   a.docs,
		server: a.server,
	}
}

// ServeHTTP delegates request handling to the gin engine.
func (a *Adapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.engine.ServeHTTP(w, r)
}

// Router exposes the underlying gin engine.
func (a *Adapter) Router() *gin.Engine {
	return a.engine
}

// Listen starts related services.
func (a *Adapter) Listen(addr string) error {
	if err := a.httpServer(addr).ListenAndServe(); err != nil {
		return fmt.Errorf("httpx/gin: listen on %q: %w", addr, err)
	}
	return nil
}

// ListenContext starts related services.
func (a *Adapter) ListenContext(ctx context.Context, addr string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	server := a.httpServer(addr)

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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.server.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("httpx/gin: shutdown on %q: %w", addr, err)
		}
		err := <-errCh
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("httpx/gin: listen on %q: %w", addr, err)
	}
}

// wrapHandler wraps related logic.
func (a *Adapter) wrapHandler(handler adapter.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		params := make(map[string]string, len(c.Params))
		for _, p := range c.Params {
			params[p.Key] = p.Value
		}

		req := c.Request.WithContext(adapter.WithRouteParams(c.Request.Context(), params))

		if err := handler(req.Context(), c.Writer, req); err != nil {
			a.logger.Error("Handler error",
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.String("error", err.Error()),
			)
			_ = c.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
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
		Handler:        a.engine.Handler(),
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
