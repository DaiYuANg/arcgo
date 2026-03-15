//go:build !no_fiber

package fiber

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/DaiYuANg/archgo/httpx/adapter"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humafiber"
	"github.com/gofiber/fiber/v2"
)

// Adapter implements the fiber runtime bridge for httpx.
type Adapter struct {
	app    *fiber.App
	group  fiber.Router
	logger *slog.Logger
	huma   huma.API
	docs   *adapter.DocsController
	opts   AppOptions
}

// AppOptions configures the fiber app created by the adapter when no app is supplied.
type AppOptions struct {
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// Options configures fiber adapter construction.
type Options struct {
	Huma   adapter.HumaOptions
	Logger *slog.Logger
	App    AppOptions
}

// New constructs a fiber adapter backed by a fiber app and Huma API.
func New(app *fiber.App, opts ...adapter.HumaOptions) *Adapter {
	options := DefaultOptions()
	options.Huma = adapter.MergeHumaOptions(opts...)
	return NewWithOptions(app, options)
}

// DefaultAppOptions returns the default fiber adapter app config.
func DefaultAppOptions() AppOptions {
	return AppOptions{
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    15 * time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownTimeout: 5 * time.Second,
	}
}

// DefaultOptions returns the default fiber adapter config.
func DefaultOptions() Options {
	return Options{
		Huma:   adapter.DefaultHumaOptions(),
		Logger: slog.Default(),
		App:    DefaultAppOptions(),
	}
}

// NewWithOptions constructs a fiber adapter from explicit construction-time options.
// App timeout settings only apply when the adapter creates the fiber app itself.
func NewWithOptions(app *fiber.App, opts Options) *Adapter {
	var a *fiber.App
	if app != nil {
		a = app
	} else {
		cfg := mergeAppOptions(opts.App)
		a = fiber.New(fiber.Config{
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		})
	}

	humaOpts := adapter.MergeHumaOptions(opts.Huma)
	cfg := huma.DefaultConfig(humaOpts.Title, humaOpts.Version)
	adapter.ApplyHumaConfig(&cfg, humaOpts)

	docsCfg := cfg
	docsCfg.DocsPath = ""
	docsCfg.OpenAPIPath = ""
	docsCfg.SchemasPath = ""

	api := humafiber.New(a, docsCfg)
	docs := adapter.NewDocsController(api, humaOpts)
	a.Use(func(c *fiber.Ctx) error {
		if docs.ServeHTTP(&responseWriter{ctx: c}, convertRequest(c)) {
			return nil
		}
		return c.Next()
	})

	return &Adapter{
		app:    a,
		group:  a,
		logger: defaultLogger(opts.Logger),
		huma:   api,
		docs:   docs,
		opts:   mergeAppOptions(opts.App),
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
	return "fiber"
}

// Handle registers a native handler on the current fiber router.
func (a *Adapter) Handle(method, path string, handler adapter.HandlerFunc) {
	a.group.Add(method, path, a.wrapHandler(handler))
}

// Group returns a child adapter scoped to a fiber group.
func (a *Adapter) Group(prefix string) adapter.Adapter {
	return &Adapter{
		app:    a.app,
		group:  a.group.Group(prefix),
		logger: a.logger,
		huma:   a.huma,
		docs:   a.docs,
		opts:   a.opts,
	}
}

// ServeHTTP reports that the fiber adapter is not exposed as a net/http handler.
func (a *Adapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "fiber adapter does not support net/http ServeHTTP; use ListenAndServe", http.StatusNotImplemented)
}

// Router exposes the underlying fiber app.
func (a *Adapter) Router() *fiber.App {
	return a.app
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

func defaultLogger(logger *slog.Logger) *slog.Logger {
	if logger != nil {
		return logger
	}
	return slog.Default()
}

func mergeAppOptions(opts AppOptions) AppOptions {
	defaults := DefaultAppOptions()
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
	return defaults
}
