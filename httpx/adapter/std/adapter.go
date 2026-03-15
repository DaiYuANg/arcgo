package std

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/DaiYuANg/archgo/httpx/adapter"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

// Adapter implements the std/chi runtime bridge for httpx.
type Adapter struct {
	router *chi.Mux
	prefix string
	logger *slog.Logger
	huma   huma.API
	docs   *adapter.DocsController
	server ServerOptions
}

// New constructs a std adapter backed by a chi router and Huma API.
func New(opts ...adapter.HumaOptions) *Adapter {
	options := DefaultOptions()
	options.Huma = adapter.MergeHumaOptions(opts...)
	return NewWithOptions(options)
}

// ServerOptions configures the std adapter's underlying http.Server.
type ServerOptions struct {
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	MaxHeaderBytes  int
}

// DefaultServerOptions returns the default std adapter server config.
func DefaultServerOptions() ServerOptions {
	return ServerOptions{
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    15 * time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownTimeout: 5 * time.Second,
		MaxHeaderBytes:  1 << 20,
	}
}

// Options configures std adapter construction.
type Options struct {
	Huma   adapter.HumaOptions
	Logger *slog.Logger
	Server ServerOptions
}

// DefaultOptions returns the default std adapter config.
func DefaultOptions() Options {
	return Options{
		Huma:   adapter.DefaultHumaOptions(),
		Logger: slog.Default(),
		Server: DefaultServerOptions(),
	}
}

// NewWithOptions constructs a std adapter from explicit construction-time options.
func NewWithOptions(opts Options) *Adapter {
	router := chi.NewMux()

	humaOpts := adapter.MergeHumaOptions(opts.Huma)
	cfg := huma.DefaultConfig(humaOpts.Title, humaOpts.Version)
	adapter.ApplyHumaConfig(&cfg, humaOpts)

	docsCfg := cfg
	docsCfg.DocsPath = ""
	docsCfg.OpenAPIPath = ""
	docsCfg.SchemasPath = ""
	api := humachi.New(router, docsCfg)
	docs := adapter.NewDocsController(api, humaOpts)

	return &Adapter{
		router: router,
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
	return "std"
}

// Handle registers a native handler on the chi router.
func (a *Adapter) Handle(method, path string, handler adapter.HandlerFunc) {
	fullPath := joinPath(a.prefix, path)
	a.router.Method(method, fullPath, a.wrapHandler(handler))
}

// Group returns a prefixed child adapter that shares the same router and Huma API.
func (a *Adapter) Group(prefix string) adapter.Adapter {
	nextPrefix := a.prefix
	if prefix != "" && prefix != "/" {
		nextPrefix = joinPath(a.prefix, prefix)
	}
	return &Adapter{
		router: a.router,
		prefix: nextPrefix,
		logger: a.logger,
		huma:   a.huma,
		docs:   a.docs,
		server: a.server,
	}
}

// ServeHTTP serves docs routes first, then falls through to the chi router.
func (a *Adapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if a.docs != nil && a.docs.ServeHTTP(w, r) {
		return
	}
	a.router.ServeHTTP(w, r)
}

// Router exposes the underlying chi router.
func (a *Adapter) Router() *chi.Mux {
	return a.router
}

// Listen starts related services.
func (a *Adapter) Listen(addr string) error {
	if err := a.httpServer(addr).ListenAndServe(); err != nil {
		return fmt.Errorf("httpx/std: listen on %q: %w", addr, err)
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
		return fmt.Errorf("httpx/std: listen on %q: %w", addr, err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.server.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("httpx/std: shutdown on %q: %w", addr, err)
		}
		err := <-errCh
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("httpx/std: listen on %q: %w", addr, err)
	}
}

// wrapHandler converts adapter-native handlers into `http.HandlerFunc`.
func (a *Adapter) wrapHandler(handler adapter.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := handler(r.Context(), w, r); err != nil {
			a.logger.Error("Handler error",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("error", err.Error()),
			)
			http.Error(w, err.Error(), http.StatusInternalServerError)
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
		Handler:        a,
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

func joinPath(prefix, path string) string {
	cleanPrefix := strings.TrimRight(prefix, "/")
	if cleanPrefix == "" {
		if path == "" {
			return "/"
		}
		if strings.HasPrefix(path, "/") {
			return path
		}
		return "/" + path
	}

	if path == "" || path == "/" {
		return cleanPrefix
	}
	if strings.HasPrefix(path, "/") {
		return cleanPrefix + path
	}
	return cleanPrefix + "/" + path
}
