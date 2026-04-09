package dix

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/DaiYuANg/arcgo/collectionx"
	collectionlist "github.com/DaiYuANg/arcgo/collectionx/list"
	"github.com/DaiYuANg/arcgo/pkg/option"
	"github.com/samber/oops"
)

// AppOption configures an App specification during construction.
type AppOption func(*appSpec)

// DefaultAppName is the fallback name used by NewDefault.
const DefaultAppName = "dix application"

// NewDefault creates an application with the default framework name.
func NewDefault(opts ...AppOption) *App {
	return New(DefaultAppName, opts...)
}

// New creates an immutable application specification.
func New(name string, opts ...AppOption) *App {
	spec := &appSpec{
		meta:    AppMeta{Name: name},
		profile: ProfileDefault,
		logger:  defaultLogger(),
	}

	option.Apply(spec, opts...)

	return &App{spec: spec}
}

// NewApp keeps backward compatibility with the v0.3 style constructor surface.
func NewApp(name string, modules ...Module) *App {
	return New(name, WithModules(modules...))
}

// NewAppWithOptions keeps backward compatibility with the v0.3 style.
//
// Deprecated: prefer New(name, WithModules(...), WithProfile(...), ...).
func NewAppWithOptions(name string, opts collectionx.List[AppOption], modules ...Module) *App {
	merged := collectionlist.NewListWithCapacity[AppOption](opts.Len()+1, WithModules(modules...))
	merged.Merge(opts)
	return New(name, merged.Values()...)
}

// WithProfile selects the runtime profile for the application.
func WithProfile(profile Profile) AppOption {
	return func(spec *appSpec) { spec.profile = profile }
}

// UseProfile selects the runtime profile for the application.
func UseProfile(profile Profile) AppOption {
	return WithProfile(profile)
}

// WithVersion sets application version metadata.
func WithVersion(version string) AppOption {
	return func(spec *appSpec) { spec.meta.Version = version }
}

// Version sets application version metadata.
func Version(version string) AppOption {
	return WithVersion(version)
}

// WithAppDescription sets application description metadata.
func WithAppDescription(description string) AppOption {
	return func(spec *appSpec) { spec.meta.Description = description }
}

// AppDescription sets application description metadata.
func AppDescription(description string) AppOption {
	return WithAppDescription(description)
}

// WithLogger sets the framework logger.
func WithLogger(logger *slog.Logger) AppOption {
	return func(spec *appSpec) {
		if logger != nil {
			spec.logger = logger
		}
	}
}

// UseLogger sets the framework logger.
func UseLogger(logger *slog.Logger) AppOption {
	return WithLogger(logger)
}

// UseLogger0 resolves the framework logger from a zero-dependency callback.
func UseLogger0(fn func() *slog.Logger) AppOption {
	return WithLoggerFrom0(fn)
}

// UseLoggerErr0 resolves the framework logger from a zero-dependency callback that can fail.
func UseLoggerErr0(fn func() (*slog.Logger, error)) AppOption {
	if fn == nil {
		return func(*appSpec) {}
	}
	return WithLoggerFrom(func(*Container) (*slog.Logger, error) {
		return fn()
	})
}

// UseLogger1 resolves the framework logger from a one-dependency callback.
func UseLogger1[D1 any](fn func(D1) *slog.Logger) AppOption {
	return WithLoggerFrom1(fn)
}

// UseLoggerErr1 resolves the framework logger from a one-dependency callback that can fail.
func UseLoggerErr1[D1 any](fn func(D1) (*slog.Logger, error)) AppOption {
	if fn == nil {
		return func(*appSpec) {}
	}
	return WithLoggerFrom(func(c *Container) (*slog.Logger, error) {
		d1, err := ResolveAs[D1](c)
		if err != nil {
			return nil, err
		}
		return fn(d1)
	})
}

// UseEventLogger sets the framework event logger. When configured, dix internal logging routes through it.
func UseEventLogger(logger EventLogger) AppOption {
	return func(spec *appSpec) {
		if logger != nil {
			spec.eventLogger = logger
		}
	}
}

// UseEventLogger0 resolves the framework event logger from a zero-dependency callback.
func UseEventLogger0(fn func() EventLogger) AppOption {
	if fn == nil {
		return func(*appSpec) {}
	}
	return UseEventLoggerErr0(func() (EventLogger, error) {
		return fn(), nil
	})
}

// UseEventLoggerErr0 resolves the framework event logger from a zero-dependency callback that can fail.
func UseEventLoggerErr0(fn func() (EventLogger, error)) AppOption {
	if fn == nil {
		return func(*appSpec) {}
	}
	return func(spec *appSpec) {
		spec.eventLoggerFromContainer = func(*Container) (EventLogger, error) {
			return fn()
		}
	}
}

// UseEventLogger1 resolves the framework event logger from a one-dependency callback.
func UseEventLogger1[D1 any](fn func(D1) EventLogger) AppOption {
	if fn == nil {
		return func(*appSpec) {}
	}
	return UseEventLoggerErr1(func(d1 D1) (EventLogger, error) {
		return fn(d1), nil
	})
}

// UseEventLoggerErr1 resolves the framework event logger from a one-dependency callback that can fail.
func UseEventLoggerErr1[D1 any](fn func(D1) (EventLogger, error)) AppOption {
	if fn == nil {
		return func(*appSpec) {}
	}
	return func(spec *appSpec) {
		spec.eventLoggerFromContainer = func(c *Container) (EventLogger, error) {
			d1, err := ResolveAs[D1](c)
			if err != nil {
				return nil, err
			}
			return fn(d1)
		}
	}
}

// WithLoggerFrom resolves the framework logger from the built DI container.
// The resolved logger overrides the default logger and updates runtime internals.
func WithLoggerFrom(fn func(*Container) (*slog.Logger, error)) AppOption {
	return func(spec *appSpec) {
		if fn != nil {
			spec.loggerFromContainer = fn
		}
	}
}

// LoggerFrom resolves the framework logger from the built DI container.
func LoggerFrom(fn func(*Container) (*slog.Logger, error)) AppOption {
	return WithLoggerFrom(fn)
}

// WithLoggerFrom0 resolves the framework logger from a zero-dependency callback.
func WithLoggerFrom0(fn func() *slog.Logger) AppOption {
	if fn == nil {
		return func(*appSpec) {}
	}
	return WithLoggerFrom(func(*Container) (*slog.Logger, error) {
		return fn(), nil
	})
}

// LoggerFrom0 resolves the framework logger from a zero-dependency callback.
func LoggerFrom0(fn func() *slog.Logger) AppOption {
	return WithLoggerFrom0(fn)
}

// WithLoggerFrom1 resolves the framework logger from a one-dependency callback.
func WithLoggerFrom1[D1 any](fn func(D1) *slog.Logger) AppOption {
	if fn == nil {
		return func(*appSpec) {}
	}
	return WithLoggerFrom(func(c *Container) (*slog.Logger, error) {
		d1, err := ResolveAs[D1](c)
		if err != nil {
			return nil, err
		}
		return fn(d1), nil
	})
}

// LoggerFrom1 resolves the framework logger from a one-dependency callback.
func LoggerFrom1[D1 any](fn func(D1) *slog.Logger) AppOption {
	return WithLoggerFrom1(fn)
}

// WithModules appends application modules.
func WithModules(modules ...Module) AppOption {
	return func(spec *appSpec) {
		spec.modules.Add(modules...)
	}
}

// Modules appends application modules.
func Modules(modules ...Module) AppOption {
	return WithModules(modules...)
}

// WithModule appends a single application module.
func WithModule(module Module) AppOption {
	return WithModules(module)
}

// WithObservers appends runtime observers that receive internal dix events.
func WithObservers(observers ...Observer) AppOption {
	return func(spec *appSpec) {
		for _, observer := range observers {
			if observer != nil {
				spec.observers = append(spec.observers, observer)
			}
		}
	}
}

// Observers appends runtime observers that receive internal dix events.
func Observers(observers ...Observer) AppOption {
	return WithObservers(observers...)
}

// WithObserver appends a single runtime observer.
func WithObserver(observer Observer) AppOption {
	return WithObservers(observer)
}

// WithDebugScopeTree logs do's scope tree after build.
func WithDebugScopeTree(enabled bool) AppOption {
	return func(spec *appSpec) { spec.debug.scopeTree = enabled }
}

// DebugScopeTree logs do's scope tree after build.
func DebugScopeTree(enabled bool) AppOption {
	return WithDebugScopeTree(enabled)
}

// WithDebugNamedServiceDependencies logs dependency trees for named services after build.
func WithDebugNamedServiceDependencies(names ...string) AppOption {
	return func(spec *appSpec) {
		spec.debug.namedServiceDependencies.Add(names...)
	}
}

// DebugNamedServiceDependencies logs dependency trees for named services after build.
func DebugNamedServiceDependencies(names ...string) AppOption {
	return WithDebugNamedServiceDependencies(names...)
}

func defaultLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))
}

// Name returns the configured application name.
func (a *App) Name() string {
	if a == nil || a.spec == nil {
		return ""
	}
	return a.spec.meta.Name
}

// Profile returns the configured application profile.
func (a *App) Profile() Profile {
	if a == nil || a.spec == nil {
		return ""
	}
	return a.spec.profile
}

// Logger returns the application logger.
func (a *App) Logger() *slog.Logger {
	if a == nil || a.spec == nil {
		return nil
	}
	return a.spec.logger
}

// EventLogger returns the configured application event logger when one is explicitly configured.
func (a *App) EventLogger() EventLogger {
	if a == nil || a.spec == nil {
		return nil
	}
	return a.spec.resolvedEventLogger()
}

// Meta returns the application metadata.
func (a *App) Meta() AppMeta {
	if a == nil || a.spec == nil {
		return AppMeta{}
	}
	return a.spec.meta
}

// Modules returns the configured application modules.
func (a *App) Modules() collectionx.List[Module] {
	if a == nil || a.spec == nil {
		return collectionx.NewList[Module]()
	}
	return a.spec.modules.Clone()
}

// Build compiles the immutable App spec into a Runtime.
func (a *App) Build() (*Runtime, error) {
	plan, _, err := a.cachedBuildPlan()
	if err != nil {
		return nil, err
	}
	return plan.Build()
}

// Start builds a Runtime and starts it with the provided context.
func (a *App) Start(ctx context.Context) (*Runtime, error) {
	rt, err := a.Build()
	if err != nil {
		return nil, oops.In("dix").
			With("op", "start", "app", a.Name()).
			Wrapf(err, "build failed")
	}
	if err := rt.Start(ctx); err != nil {
		return nil, oops.In("dix").
			With("op", "start", "app", a.Name()).
			Wrapf(err, "start failed")
	}
	return rt, nil
}

// RunContext builds a Runtime, starts it, waits for the context to finish, and stops it.
func (a *App) RunContext(ctx context.Context) error {
	rt, err := a.Start(ctx)
	if err != nil {
		return err
	}

	<-ctx.Done()

	stopCtx := context.WithoutCancel(ctx)
	if err := rt.Stop(stopCtx); err != nil {
		return oops.In("dix").
			With("op", "run_context_stop", "app", a.Name()).
			Wrapf(err, "stop failed")
	}

	return nil
}

// Run builds a Runtime, starts it, waits for shutdown signals, and stops it.
func (a *App) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return a.RunContext(ctx)
}
