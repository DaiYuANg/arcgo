package configx

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/go-playground/validator/v10"
	"github.com/samber/lo"
	"github.com/samber/mo"
)

// Source identifies a configuration source.
type Source int

const (
	// SourceDotenv reads values from .env files via godotenv.
	SourceDotenv Source = iota
	// SourceFile reads values from YAML, JSON, or TOML files.
	SourceFile
	// SourceEnv reads values from OS environment variables.
	SourceEnv
)

// String returns a human-readable name for the source.
func (s Source) String() string {
	switch s {
	case SourceDotenv:
		return "dotenv"
	case SourceFile:
		return "file"
	case SourceEnv:
		return "env"
	default:
		return "unknown"
	}
}

// ValidateLevel controls how the loaded config is validated.
type ValidateLevel int

const (
	// ValidateLevelNone skips all validation after loading.
	ValidateLevelNone ValidateLevel = iota
	// ValidateLevelStruct runs go-playground/validator struct validation.
	ValidateLevelStruct
)

// defaultEnvSeparator is the substring in an env key that is replaced with "."
// to form a koanf path. For example, with separator "_", APP_DB_HOST becomes
// db.host. Switch to "__" (WithEnvSeparator) to treat a single underscore as
// part of the key name: APP_DB__HOST → db.host, APP_MAX_RETRY → max_retry.
const defaultEnvSeparator = "_"

// Options holds every knob that controls config loading and watching.
// Build one with NewOptions and then apply functional Option values.
type Options struct {
	// --- loading ---
	dotenvFiles []string
	envPrefix   string
	// envSeparator is the string within an env key that maps to the koanf "."
	// path delimiter. Defaults to "_". Set to "__" for double-underscore
	// nesting convention.
	envSeparator    string
	files           []string
	priority        []Source
	defaults        mo.Option[map[string]any]
	typedDefaults   mo.Option[map[string]any]
	ignoreDotenvErr bool

	// --- validation ---
	validate      *validator.Validate
	validateLevel ValidateLevel

	// --- watching (hot reload) ---
	// watchDebounce is the minimum quiet period after the last file-change
	// event before a reload is triggered. Defaults to 100 ms.
	watchDebounce time.Duration
	// watchErrHandler is called whenever a watch-related error occurs (e.g.
	// a file watcher drops an event or a reload fails). If nil, errors are
	// silently ignored. Use WithWatchErrHandler to supply a logger.
	watchErrHandler func(error)

	// --- observability ---
	observability observabilityx.Observability
}

// Option is a functional option that mutates an *Options.
type Option func(*Options)

// NewOptions returns an *Options pre-filled with sensible defaults.
//
// Default source priority: dotenv → file → env (later sources override earlier ones).
// Default env separator: "_" (APP_DB_HOST → db.host).
// Default watch debounce: 100 ms.
// Dotenv errors are ignored by default (files are optional).
func NewOptions() *Options {
	return &Options{
		dotenvFiles:     []string{".env", ".env.local"},
		priority:        []Source{SourceDotenv, SourceFile, SourceEnv},
		envSeparator:    defaultEnvSeparator,
		validateLevel:   ValidateLevelNone,
		ignoreDotenvErr: true,
		watchDebounce:   100 * time.Millisecond,
		observability:   observabilityx.Nop(),
	}
}

// ── loading options ───────────────────────────────────────────────────────────

// WithDotenv sets the dotenv files to load. When called with no arguments the
// default list (".env", ".env.local") is kept. The files are loaded in order;
// later files override earlier ones.
func WithDotenv(files ...string) Option {
	return func(o *Options) {
		if len(files) > 0 {
			o.dotenvFiles = files
		}
	}
}

// WithEnvPrefix limits which environment variables are considered. Only
// variables whose names start with prefix (case-insensitive, trailing "_" is
// added automatically) are loaded. For example, "APP" matches APP_PORT.
func WithEnvPrefix(prefix string) Option {
	return func(o *Options) { o.envPrefix = prefix }
}

// WithEnvSeparator sets the substring within an env key that is replaced with
// "." to form a nested koanf path. The default is "_", which means
// APP_DB_HOST → db.host. Set to "__" to use the double-underscore convention:
// APP_DB__HOST → db.host while APP_MAX_RETRY stays as max_retry.
func WithEnvSeparator(sep string) Option {
	return func(o *Options) {
		if sep != "" {
			o.envSeparator = sep
		}
	}
}

// WithFiles sets the config files to load. Supported formats: .yaml/.yml,
// .json, .toml. Files are loaded in order; later files override earlier ones.
// Files with unrecognised extensions return ErrUnsupportedFileFormat.
func WithFiles(files ...string) Option {
	return func(o *Options) { o.files = files }
}

// WithPriority overrides the source loading order. Sources listed later
// override sources listed earlier, so [SourceDotenv, SourceFile, SourceEnv]
// means env vars win over file values which win over dotenv values.
func WithPriority(p ...Source) Option {
	return func(o *Options) { o.priority = p }
}

// WithDefaults sets in-memory default values loaded before any file or env
// source. Keys use the koanf "." path delimiter (e.g. "server.port").
func WithDefaults(m map[string]any) Option {
	return func(o *Options) {
		o.defaults = mo.Some(m)
	}
}

// WithDefaultsTyped sets default values from a typed map, converting all
// values to any automatically.
func WithDefaultsTyped[T any](m map[string]T) Option {
	return func(o *Options) {
		o.defaults = mo.Some(lo.MapValues(m, func(value T, _ string) any {
			return value
		}))
	}
}

// WithTypedDefaults sets defaults from a strongly typed config object.
// This option is intended for typed loading flows (LoadT/LoadTErr/LoaderT).
func WithTypedDefaults[T any](cfg T) Option {
	return func(o *Options) {
		m, err := typedDefaultsToMap(cfg)
		if err != nil {
			// Preserve option signature; the load path will report this via ErrDefaults.
			o.typedDefaults = mo.Some(map[string]any{"__configx_invalid_typed_defaults__": err.Error()})
			return
		}
		o.typedDefaults = mo.Some(m)
	}
}

func typedDefaultsToMap(v any) (map[string]any, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("%w: marshal typed defaults: %v", ErrDefaults, err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("%w: unmarshal typed defaults: %v", ErrDefaults, err)
	}
	if out == nil {
		return nil, fmt.Errorf("%w: typed defaults must be an object-like value", ErrDefaults)
	}
	return out, nil
}

// WithIgnoreDotenvError controls whether missing or malformed dotenv files
// are silently skipped (true, the default) or returned as errors (false).
func WithIgnoreDotenvError(ignore bool) Option {
	return func(o *Options) { o.ignoreDotenvErr = ignore }
}

// ── validation options ────────────────────────────────────────────────────────

// WithValidator replaces the default go-playground/validator instance used
// for struct validation after loading.
func WithValidator(v *validator.Validate) Option {
	return func(o *Options) { o.validate = v }
}

// WithValidateLevel sets the validation level applied after a successful load.
// ValidateLevelNone (default) skips validation entirely.
// ValidateLevelStruct runs full struct validation.
func WithValidateLevel(level ValidateLevel) Option {
	return func(o *Options) { o.validateLevel = level }
}

// ── watch / hot-reload options ────────────────────────────────────────────────

// WithWatchDebounce sets how long the Watcher waits after the last file-change
// event before triggering a reload. Rapid successive saves are collapsed into a
// single reload. Defaults to 100 ms. Values ≤ 0 are ignored.
func WithWatchDebounce(d time.Duration) Option {
	return func(o *Options) {
		if d > 0 {
			o.watchDebounce = d
		}
	}
}

// WithWatchErrHandler registers a function that is called whenever the Watcher
// encounters an error (e.g. a file is removed, an fsnotify event is lost, or a
// reload fails). If not set, watch errors are silently discarded.
//
// Example – log watch errors with slog:
//
//	configx.WithWatchErrHandler(func(err error) {
//	    slog.Error("config watch error", "err", err)
//	})
func WithWatchErrHandler(fn func(error)) Option {
	return func(o *Options) { o.watchErrHandler = fn }
}

// ── observability options ─────────────────────────────────────────────────────

// WithObservability attaches an observabilityx.Observability implementation
// that receives traces and metrics for every config load operation.
func WithObservability(obs observabilityx.Observability) Option {
	return func(o *Options) {
		o.observability = obs
	}
}
