package configx

import (
	"context"
	"fmt"
	"time"

	"github.com/DaiYuANg/archgo/observabilityx"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
	"github.com/samber/lo"
	"github.com/samber/mo"
)

const (
	metricConfigLoadTotal            = "configx_load_total"
	metricConfigLoadDurationMS       = "configx_load_duration_ms"
	metricConfigSourceLoadTotal      = "configx_source_load_total"
	metricConfigSourceLoadDurationMS = "configx_source_load_duration_ms"
)

// ─── Loader ───────────────────────────────────────────────────────────────────

// Loader loads configuration from the sources defined in its Options and can
// optionally watch those sources for live changes.
//
// Build one with [New] and then call [Loader.Load], [Loader.LoadConfig], or
// [Loader.Watch] / [Loader.NewWatcher] for hot-reload support.
type Loader struct {
	opts *Options
}

// New creates a Loader from the supplied functional options.
//
//	loader := configx.New(
//	    configx.WithFiles("config.yaml"),
//	    configx.WithEnvPrefix("APP"),
//	)
func New(opts ...Option) *Loader {
	options := NewOptions()
	lo.ForEach(opts, func(opt Option, _ int) {
		if opt != nil {
			opt(options)
		}
	})
	return &Loader{opts: options}
}

// Load reads all configured sources, unmarshals the result into out, and runs
// struct validation according to the configured ValidateLevel.
func (l *Loader) Load(out any) error {
	cfg, err := l.loadInternal()
	if err != nil {
		return fmt.Errorf("configx: load config: %w", err)
	}
	if err := cfg.k.Unmarshal("", out); err != nil {
		return fmt.Errorf("configx: unmarshal config into output: %w", err)
	}
	if err := cfg.validateStruct(out); err != nil {
		return fmt.Errorf("configx: validate output: %w", err)
	}
	return nil
}

// LoadConfig reads all configured sources and returns a *Config for ad-hoc
// path-based access (GetString, GetInt, Unmarshal, …).
func (l *Loader) LoadConfig() (*Config, error) {
	return l.loadInternal()
}

// NewWatcher performs the initial load and returns a *Watcher that will
// re-read all sources whenever a watched config file changes.
//
// Call [Watcher.Start] (typically in a goroutine) to begin watching.
func (l *Loader) NewWatcher() (*Watcher, error) {
	return newWatcherFromOptions(l.opts)
}

// Watch is a convenience wrapper around [Loader.NewWatcher] + [Watcher.Start].
// It registers onChange as a [ChangeHandler] and then blocks until ctx is
// cancelled.  onChange may be nil if the caller only needs the side-effect of
// keeping w.Config() up-to-date.
func (l *Loader) Watch(ctx context.Context, onChange ChangeHandler) error {
	w, err := newWatcherFromOptions(l.opts)
	if err != nil {
		return err
	}
	if onChange != nil {
		w.OnChange(onChange)
	}
	return w.Start(ctx)
}

func (l *Loader) loadInternal() (*Config, error) {
	return loadConfigFromOptions(l.opts)
}

// ─── LoaderT ──────────────────────────────────────────────────────────────────

// LoaderT is the generic, type-safe counterpart of [Loader]. It unmarshals the
// full config into T and returns the result wrapped in a [mo.Result].
//
// Build one with [NewT] and then call [LoaderT.Load], [LoaderT.LoadConfig], or
// [LoaderT.Watch] / [LoaderT.NewWatcher] for hot-reload support.
type LoaderT[T any] struct {
	opts *Options
}

// NewT creates a LoaderT[T] from the supplied functional options.
//
//	loader := configx.NewT[AppConfig](
//	    configx.WithFiles("config.yaml"),
//	    configx.WithEnvPrefix("APP"),
//	    configx.WithValidateLevel(configx.ValidateLevelRequired),
//	)
func NewT[T any](opts ...Option) *LoaderT[T] {
	options := NewOptions()
	lo.ForEach(opts, func(opt Option, _ int) {
		if opt != nil {
			opt(options)
		}
	})
	return &LoaderT[T]{opts: options}
}

// Load reads all configured sources, unmarshals the result into a new T, runs
// struct validation, and returns the value wrapped in a [mo.Result].
func (l *LoaderT[T]) Load() mo.Result[T] {
	cfg, err := l.loadInternal()
	if err != nil {
		return mo.Err[T](err)
	}

	var out T
	if err := cfg.k.Unmarshal("", &out); err != nil {
		return mo.Err[T](fmt.Errorf("configx: unmarshal config into typed output: %w", err))
	}
	if err := cfg.validateStruct(out); err != nil {
		return mo.Err[T](fmt.Errorf("configx: validate typed output: %w", err))
	}
	return mo.Ok(out)
}

// LoadConfig reads all configured sources and returns a raw *Config for
// path-based access.
func (l *LoaderT[T]) LoadConfig() (*Config, error) {
	return l.loadInternal()
}

// NewWatcher performs the initial load and returns a *Watcher that will
// re-read all sources whenever a watched config file changes.
//
// Call [Watcher.Start] (typically in a goroutine) to begin watching.
func (l *LoaderT[T]) NewWatcher() (*Watcher, error) {
	return newWatcherFromOptions(l.opts)
}

// Watch is a convenience wrapper around [LoaderT.NewWatcher] + [Watcher.Start].
// It registers onChange as a [ChangeHandler] and then blocks until ctx is
// cancelled.
func (l *LoaderT[T]) Watch(ctx context.Context, onChange ChangeHandler) error {
	w, err := newWatcherFromOptions(l.opts)
	if err != nil {
		return err
	}
	if onChange != nil {
		w.OnChange(onChange)
	}
	return w.Start(ctx)
}

func (l *LoaderT[T]) loadInternal() (*Config, error) {
	return loadConfigFromOptions(l.opts)
}

// ─── package-level helpers ────────────────────────────────────────────────────

// Load is a one-shot helper: it creates a temporary Loader, loads all sources,
// and unmarshals the result into out.
//
//	var cfg AppConfig
//	if err := configx.Load(&cfg,
//	    configx.WithFiles("config.yaml"),
//	    configx.WithEnvPrefix("APP"),
//	); err != nil { … }
func Load(out any, opts ...Option) error {
	return New(opts...).Load(out)
}

// LoadT is a one-shot helper that returns the typed config wrapped in a
// [mo.Result].
func LoadT[T any](opts ...Option) mo.Result[T] {
	return NewT[T](opts...).Load()
}

// LoadTErr is a one-shot helper that returns the typed config as a plain
// (value, error) pair.
func LoadTErr[T any](opts ...Option) (T, error) {
	result := LoadT[T](opts...)
	if result.IsError() {
		var zero T
		return zero, result.Error()
	}
	return result.Get()
}

// LoadConfig is a one-shot helper that returns a raw *Config.
func LoadConfig(opts ...Option) (*Config, error) {
	return New(opts...).LoadConfig()
}

// LoadConfigT is a one-shot helper that returns a raw *Config (the type
// parameter T is used only for option inference; it is not unmarshalled here).
func LoadConfigT[T any](opts ...Option) (*Config, error) {
	return NewT[T](opts...).LoadConfig()
}

// ─── core load logic ──────────────────────────────────────────────────────────

// loadConfigFromOptions is the single authoritative code path that builds a
// koanf instance from an *Options and wraps it in a *Config.  All exported
// load functions ultimately call this.
func loadConfigFromOptions(opts *Options) (*Config, error) {
	if opts == nil {
		opts = NewOptions()
	}

	obs := observabilityx.Normalize(opts.observability, nil)
	ctx, span := obs.StartSpan(context.Background(), "configx.load")
	defer span.End()

	start := time.Now()
	result := "success"
	defer func() {
		obs.AddCounter(ctx, metricConfigLoadTotal, 1,
			observabilityx.String("result", result),
		)
		obs.RecordHistogram(ctx, metricConfigLoadDurationMS, float64(time.Since(start).Milliseconds()),
			observabilityx.String("result", result),
		)
	}()

	k := koanf.New(".")

	// 1. In-memory defaults (map form) – loaded first so every other source
	//    can override them.
	if opts.defaults.IsPresent() {
		defaults, _ := opts.defaults.Get()
		if err := k.Load(confmap.Provider(defaults, "."), nil); err != nil {
			result = "error"
			span.RecordError(err)
			return nil, fmt.Errorf("configx: load defaults map: %w", err)
		}
	}

	// 2. In-memory defaults (struct form).
	if opts.defaultsStruct != nil {
		if err := loadDefaultsStruct(k, opts.defaultsStruct); err != nil {
			result = "error"
			span.RecordError(err)
			return nil, fmt.Errorf("configx: load defaults struct: %w", err)
		}
	}

	// 3. External sources in priority order (later = higher precedence).
	for _, src := range opts.priority {
		switch src {
		case SourceDotenv:
			if err := loadSourceWithObservability(ctx, obs, src, func() error {
				return loadDotenv(opts.dotenvFiles, opts.ignoreDotenvErr)
			}); err != nil {
				result = "error"
				span.RecordError(err)
				return nil, fmt.Errorf("configx: load dotenv source: %w", err)
			}

		case SourceFile:
			if err := loadSourceWithObservability(ctx, obs, src, func() error {
				return loadFiles(k, opts.files)
			}); err != nil {
				result = "error"
				span.RecordError(err)
				return nil, fmt.Errorf("configx: load file source: %w", err)
			}

		case SourceEnv:
			if err := loadSourceWithObservability(ctx, obs, src, func() error {
				// envSeparator is resolved inside loadEnv; passing it here
				// makes the behaviour explicit and testable.
				return loadEnv(k, opts.envPrefix, opts.envSeparator)
			}); err != nil {
				result = "error"
				span.RecordError(err)
				return nil, fmt.Errorf("configx: load env source: %w", err)
			}
		}
	}

	return newConfig(k, opts), nil
}

// loadSourceWithObservability wraps fn with a child span and per-source
// metrics so that every load operation is independently observable.
func loadSourceWithObservability(
	ctx context.Context,
	obs observabilityx.Observability,
	source Source,
	fn func() error,
) error {
	if fn == nil {
		return nil
	}

	sourceName := source.String()
	sourceCtx, sourceSpan := obs.StartSpan(ctx, "configx.load."+sourceName,
		observabilityx.String("source", sourceName),
	)
	defer sourceSpan.End()

	start := time.Now()
	result := "success"
	defer func() {
		obs.AddCounter(sourceCtx, metricConfigSourceLoadTotal, 1,
			observabilityx.String("source", sourceName),
			observabilityx.String("result", result),
		)
		obs.RecordHistogram(sourceCtx, metricConfigSourceLoadDurationMS, float64(time.Since(start).Milliseconds()),
			observabilityx.String("source", sourceName),
			observabilityx.String("result", result),
		)
	}()

	if err := fn(); err != nil {
		result = "error"
		sourceSpan.RecordError(err)
		return err
	}

	return nil
}
