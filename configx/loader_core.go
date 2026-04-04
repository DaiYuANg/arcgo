package configx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
	"github.com/samber/lo"
)

// loadConfigFromOptions is the single authoritative code path that builds a
// koanf instance from an *Options and wraps it in a *Config. All exported load
// functions ultimately call this.
func loadConfigFromOptions(ctx context.Context, opts *Options) (_ *Config, err error) {
	opts = normalizeLoadOptions(opts)
	logDebug(opts,
		"configx load started",
		"files", len(opts.files),
		"dotenv_files", len(opts.dotenvFiles),
		"priority", len(opts.priority),
		"env_prefix", opts.envPrefix,
		"raw_args", len(opts.args),
		"args_flags", changedFlagCount(opts.argsFlagSet),
	)

	ctx, obs, finish := beginConfigLoad(ctx, opts)
	defer func() {
		finish(err)
	}()

	k := koanf.New(".")
	if err := loadConfiguredDefaults(k, opts); err != nil {
		return nil, err
	}
	if err := loadConfiguredSources(ctx, obs, k, opts); err != nil {
		return nil, err
	}

	return newConfig(k, opts), nil
}

func normalizeLoadOptions(opts *Options) *Options {
	if opts == nil {
		return NewOptions()
	}
	return opts
}

func beginConfigLoad(
	ctx context.Context,
	opts *Options,
) (context.Context, observabilityx.Observability, func(error)) {
	if ctx == nil {
		ctx = context.Background()
	}

	obs := observabilityx.Normalize(opts.observability, nil)
	ctx, span := obs.StartSpan(ctx, "configx.load")
	start := time.Now()

	return ctx, obs, func(err error) {
		result := "success"
		if err != nil {
			result = "error"
			span.RecordError(err)
			logError(opts, "configx load failed", "error", err)
		} else {
			logDebug(opts, "configx load completed", "result", result)
		}

		obs.AddCounter(ctx, metricConfigLoadTotal, 1,
			observabilityx.String("result", result),
		)
		obs.RecordHistogram(ctx, metricConfigLoadDurationMS, float64(time.Since(start).Milliseconds()),
			observabilityx.String("result", result),
		)
		span.End()
	}
}

func loadConfiguredDefaults(k *koanf.Koanf, opts *Options) error {
	if err := loadTypedDefaults(k, opts); err != nil {
		return err
	}

	if !opts.defaults.IsPresent() {
		return nil
	}

	defaults, _ := opts.defaults.Get()
	if err := k.Load(confmap.Provider(defaults, "."), nil); err != nil {
		return fmt.Errorf("defaults map: %w", errors.Join(ErrDefaults, err))
	}

	logDebug(opts, "configx defaults loaded")
	return nil
}

func loadTypedDefaults(k *koanf.Koanf, opts *Options) error {
	if !opts.typedDefaults.IsPresent() {
		return nil
	}

	defaults, _ := opts.typedDefaults.Get()
	if errMsg, bad := defaults["__configx_invalid_typed_defaults__"].(string); bad {
		return fmt.Errorf("typed defaults: %w", errors.Join(ErrDefaults, errors.New(errMsg)))
	}

	if err := k.Load(confmap.Provider(defaults, "."), nil); err != nil {
		return fmt.Errorf("typed defaults map: %w", errors.Join(ErrDefaults, err))
	}

	logDebug(opts, "configx typed defaults loaded")
	return nil
}

func loadConfiguredSources(
	ctx context.Context,
	obs observabilityx.Observability,
	k *koanf.Koanf,
	opts *Options,
) error {
	if err := lo.Reduce(opts.priority, func(result error, src Source, _ int) error {
		if result != nil {
			return result
		}
		return loadConfiguredSource(ctx, obs, k, opts, src)
	}, error(nil)); err != nil {
		return fmt.Errorf("configx: load configured sources: %w", err)
	}
	return nil
}

func loadConfiguredSource(
	ctx context.Context,
	obs observabilityx.Observability,
	k *koanf.Koanf,
	opts *Options,
	src Source,
) error {
	switch src {
	case SourceDotenv:
		logDebug(opts, "configx source loading", "source", src.String())
		if err := loadSourceWithObservability(ctx, obs, src, func() error {
			return loadDotenv(opts.dotenvFiles, opts.ignoreDotenvErr)
		}); err != nil {
			return fmt.Errorf("dotenv source: %w", errors.Join(ErrLoad, err))
		}
		logDebug(opts, "configx source loaded", "source", src.String())

	case SourceFile:
		logDebug(opts, "configx source loading", "source", src.String(), "files", len(opts.files))
		if err := loadSourceWithObservability(ctx, obs, src, func() error {
			return loadFiles(k, opts.files)
		}); err != nil {
			return fmt.Errorf("file source: %w", errors.Join(ErrLoad, err))
		}
		logDebug(opts, "configx source loaded", "source", src.String())

	case SourceEnv:
		logDebug(opts, "configx source loading", "source", src.String(), "env_prefix", opts.envPrefix)
		if err := loadSourceWithObservability(ctx, obs, src, func() error {
			return loadEnv(k, opts.envPrefix, opts.envSeparator)
		}); err != nil {
			return fmt.Errorf("env source: %w", errors.Join(ErrLoad, err))
		}
		logDebug(opts, "configx source loaded", "source", src.String())

	case SourceArgs:
		logDebug(opts,
			"configx source loading",
			"source", src.String(),
			"raw_args", len(opts.args),
			"changed_flags", changedFlagCount(opts.argsFlagSet),
		)
		if err := loadSourceWithObservability(ctx, obs, src, func() error {
			return loadArgs(k, opts.args, opts.argsFlagSet, opts.argsNameFunc)
		}); err != nil {
			return fmt.Errorf("args source: %w", errors.Join(ErrLoad, err))
		}
		logDebug(opts, "configx source loaded", "source", src.String())
	}

	return nil
}

// loadSourceWithObservability wraps fn with a child span and per-source metrics
// so that every load operation is independently observable.
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
