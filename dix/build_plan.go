//revive:disable:file-length-limit Build planning is kept together because the steps share one orchestration flow.

package dix

import (
	"context"
	"errors"
	"log/slog"
	"time"

	collectionlist "github.com/DaiYuANg/arcgo/collectionx/list"
	"github.com/samber/do/v2"
	"github.com/samber/oops"
)

type buildPlan struct {
	spec    *appSpec
	modules *collectionlist.List[*moduleSpec]
}

func newUnvalidatedBuildPlan(ctx context.Context, app *App) (*buildPlan, error) {
	if app == nil || app.spec == nil {
		return nil, oops.In("dix").
			With("op", "new_unvalidated_build_plan").
			New("app is nil")
	}

	modules, err := flattenModuleList(&app.spec.modules, app.spec.profile)
	if err != nil {
		logMessageEvent(ctx, app.spec.resolvedEventLogger(), EventLevelError, "module flatten failed", "app", app.Name(), "error", err)
		return nil, oops.In("dix").
			With("op", "flatten_modules", "app", app.Name()).
			Wrapf(err, "module flatten failed")
	}

	plan := &buildPlan{
		spec:    app.spec,
		modules: modules,
	}

	return plan, nil
}

func (p *buildPlan) Build(ctx context.Context) (_ *Runtime, err error) {
	startedAt := time.Now()
	var rt *Runtime
	defer func() {
		p.emitBuildResult(ctx, rt, time.Since(startedAt), err)
	}()

	if p == nil || p.spec == nil {
		err = oops.In("dix").
			With("op", "build_runtime").
			New("build plan is nil")
		return nil, err
	}

	rt = newRuntime(p.spec, p)
	registerRuntimeCoreServices(rt)

	providersRegistered, err := p.prepareBuildLogging(ctx, rt)
	if err != nil {
		err = cleanupBuildFailure(ctx, rt, err)
		return nil, err
	}

	debugEnabled := eventLoggerEnabled(ctx, rt.eventLogger, EventLevelDebug)
	infoEnabled := eventLoggerEnabled(ctx, rt.eventLogger, EventLevelInfo)
	p.logBuildStart(ctx, rt, infoEnabled, debugEnabled)

	if providersRegistered {
		p.logProviderRegistrations(ctx, rt, debugEnabled)
	} else {
		p.registerProviders(ctx, rt, debugEnabled)
	}

	if err := p.bindHooksAndRunSetups(ctx, rt, debugEnabled); err != nil {
		err = cleanupBuildFailure(ctx, rt, err)
		return nil, err
	}

	if err := p.runInvokes(ctx, rt, debugEnabled); err != nil {
		err = cleanupBuildFailure(ctx, rt, err)
		return nil, err
	}

	rt.transitionState(ctx, AppStateBuilt, "build completed")
	rt.logDebugInformation(ctx)
	return rt, nil
}

func (p *buildPlan) emitBuildResult(ctx context.Context, rt *Runtime, duration time.Duration, err error) {
	if p == nil || p.spec == nil {
		return
	}
	event := p.buildEvent(duration, err)
	if rt == nil {
		p.spec.emitBuild(ctx, event)
		return
	}
	emitEventLogger(ctx, rt.eventLogger, event)
	emitObservers(ctx, rt.logger, p.spec.observers, func(ctx context.Context, observer Observer) {
		observer.OnBuild(ctx, event)
	})
}

func (p *buildPlan) prepareBuildLogging(ctx context.Context, rt *Runtime) (bool, error) {
	if !p.needsBuildLogging(rt) {
		return false, nil
	}

	p.registerProviders(ctx, rt, false)

	if p.spec.eventLoggerFromContainer != nil {
		return true, p.applyResolvedEventLogger(rt)
	}

	if p.spec.loggerFromContainer != nil {
		return true, p.applyResolvedLogger(rt)
	}

	return true, nil
}

func (p *buildPlan) needsBuildLogging(rt *Runtime) bool {
	return p != nil &&
		p.spec != nil &&
		rt != nil &&
		(p.spec.eventLoggerFromContainer != nil || p.spec.loggerFromContainer != nil)
}

func (p *buildPlan) applyResolvedEventLogger(rt *Runtime) error {
	resolvedEventLogger, err := p.resolveFrameworkEventLogger(rt)
	if err != nil {
		return err
	}
	rt.eventLogger = resolvedEventLogger
	rt.container.eventLogger = resolvedEventLogger
	rt.lifecycle.eventLogger = resolvedEventLogger
	return nil
}

func (p *buildPlan) applyResolvedLogger(rt *Runtime) error {
	resolvedLogger, err := p.resolveFrameworkLogger(rt)
	if err != nil {
		return err
	}
	rt.logger = resolvedLogger
	rt.container.logger = resolvedLogger
	rt.lifecycle.logger = resolvedLogger
	if p.spec.eventLogger == nil {
		resolvedEventLogger := NewSlogEventLogger(resolvedLogger)
		rt.eventLogger = resolvedEventLogger
		rt.container.eventLogger = resolvedEventLogger
		rt.lifecycle.eventLogger = resolvedEventLogger
	}
	do.OverrideNamedValue(rt.container.Raw(), serviceNameOf[*slog.Logger](), resolvedLogger)
	return nil
}

func cleanupBuildFailure(ctx context.Context, rt *Runtime, buildErr error) error {
	if rt == nil || rt.container == nil {
		return buildErr
	}

	report := rt.container.ShutdownReport(ctx)
	if report == nil || len(report.Errors) == 0 {
		return buildErr
	}
	rt.logMessage(ctx, EventLevelError, "build cleanup failed", "app", rt.Name(), "error", report)
	return errors.Join(buildErr, report)
}

func (p *buildPlan) logBuildStart(ctx context.Context, rt *Runtime, infoEnabled, debugEnabled bool) {
	if infoEnabled {
		rt.logMessage(ctx, EventLevelInfo, "building app", "app", p.spec.meta.Name, "profile", p.spec.profile)
	}
	if debugEnabled {
		rt.logMessage(ctx, EventLevelDebug, "build plan ready",
			"app", p.spec.meta.Name,
			"modules", p.modules.Len(),
			"providers", countModuleProviders(p.modules),
			"hooks", countModuleHooks(p.modules),
			"setups", countModuleSetups(p.modules),
			"invokes", countModuleInvokes(p.modules),
		)
	}
}

func registerRuntimeCoreServices(rt *Runtime) {
	ProvideValueT[*slog.Logger](rt.container, rt.logger)
	ProvideValueT[AppMeta](rt.container, rt.spec.meta)
	ProvideValueT[Profile](rt.container, rt.spec.profile)
}

func (p *buildPlan) resolveFrameworkLogger(rt *Runtime) (*slog.Logger, error) {
	if p == nil || p.spec == nil || rt == nil || p.spec.loggerFromContainer == nil {
		return nil, oops.In("dix").
			With("op", "resolve_framework_logger").
			New("resolve framework logger failed: resolver is not configured")
	}

	logger, err := p.spec.loggerFromContainer(rt.container)
	if err != nil {
		return nil, oops.In("dix").
			With("op", "resolve_framework_logger", "app", rt.Name()).
			Wrapf(err, "resolve framework logger failed")
	}
	if logger == nil {
		return nil, oops.In("dix").
			With("op", "resolve_framework_logger", "app", rt.Name()).
			New("resolve framework logger failed: resolver returned nil logger")
	}

	return logger, nil
}

func (p *buildPlan) resolveFrameworkEventLogger(rt *Runtime) (EventLogger, error) {
	if p == nil || p.spec == nil || rt == nil || p.spec.eventLoggerFromContainer == nil {
		return nil, oops.In("dix").
			With("op", "resolve_framework_event_logger").
			New("resolve framework event logger failed: resolver is not configured")
	}

	logger, err := p.spec.eventLoggerFromContainer(rt.container)
	if err != nil {
		return nil, oops.In("dix").
			With("op", "resolve_framework_event_logger", "app", rt.Name()).
			Wrapf(err, "resolve framework event logger failed")
	}
	if logger == nil {
		return nil, oops.In("dix").
			With("op", "resolve_framework_event_logger", "app", rt.Name()).
			New("resolve framework event logger failed: resolver returned nil event logger")
	}

	return logger, nil
}

func (p *buildPlan) registerProviders(ctx context.Context, rt *Runtime, debugEnabled bool) {
	p.modules.Range(func(_ int, mod *moduleSpec) bool {
		if debugEnabled {
			rt.logMessage(ctx, EventLevelDebug, "registering module",
				"module", mod.name,
				"providers", mod.providers.Len(),
				"hooks", mod.hooks.Len(),
				"setups", mod.setups.Len(),
				"invokes", mod.invokes.Len(),
			)
		}
		mod.providers.Range(func(_ int, provider ProviderFunc) bool {
			if debugEnabled {
				rt.logMessage(ctx, EventLevelDebug, "registering provider",
					"module", mod.name,
					"label", provider.meta.Label,
					"output", provider.meta.Output.Name,
					"dependencies", serviceRefNames(provider.meta.Dependencies),
					"raw", provider.meta.Raw,
				)
			}
			provider.apply(rt.container)
			return true
		})
		return true
	})
}

func (p *buildPlan) logProviderRegistrations(ctx context.Context, rt *Runtime, debugEnabled bool) {
	if !debugEnabled {
		return
	}
	p.modules.Range(func(_ int, mod *moduleSpec) bool {
		rt.logMessage(ctx, EventLevelDebug, "registering module",
			"module", mod.name,
			"providers", mod.providers.Len(),
			"hooks", mod.hooks.Len(),
			"setups", mod.setups.Len(),
			"invokes", mod.invokes.Len(),
		)
		mod.providers.Range(func(_ int, provider ProviderFunc) bool {
			rt.logMessage(ctx, EventLevelDebug, "registering provider",
				"module", mod.name,
				"label", provider.meta.Label,
				"output", provider.meta.Output.Name,
				"dependencies", serviceRefNames(provider.meta.Dependencies),
				"raw", provider.meta.Raw,
			)
			return true
		})
		return true
	})
}

func (p *buildPlan) bindHooksAndRunSetups(ctx context.Context, rt *Runtime, debugEnabled bool) error {
	var setupErr error
	p.modules.Range(func(_ int, mod *moduleSpec) bool {
		bindModuleHooks(ctx, mod, rt, debugEnabled)
		setupErr = runModuleSetups(ctx, mod, rt, debugEnabled)
		return setupErr == nil
	})
	return setupErr
}

func bindModuleHooks(ctx context.Context, mod *moduleSpec, rt *Runtime, debugEnabled bool) {
	mod.hooks.Range(func(_ int, hook HookFunc) bool {
		if debugEnabled {
			rt.logMessage(ctx, EventLevelDebug, "binding lifecycle hook",
				"module", mod.name,
				"label", hook.meta.Label,
				"kind", hook.meta.Kind,
				"dependencies", serviceRefNames(hook.meta.Dependencies),
				"raw", hook.meta.Raw,
			)
		}
		hook.bind(rt.container, rt.lifecycle)
		return true
	})
}

func runModuleSetups(ctx context.Context, mod *moduleSpec, rt *Runtime, debugEnabled bool) error {
	var setupErr error
	mod.setups.Range(func(_ int, setup SetupFunc) bool {
		if debugEnabled {
			rt.logMessage(ctx, EventLevelDebug, "running module setup",
				"module", mod.name,
				"label", setup.meta.Label,
				"dependencies", serviceRefNames(setup.meta.Dependencies),
				"provides", serviceRefNames(setup.meta.Provides),
				"overrides", serviceRefNames(setup.meta.Overrides),
				"graph_mutation", setup.meta.GraphMutation,
				"raw", setup.meta.Raw,
			)
		}
		if err := setup.apply(rt.container, rt.lifecycle); err != nil {
			rt.logMessage(ctx, EventLevelError, "module setup failed", "module", mod.name, "label", setup.meta.Label, "error", err)
			setupErr = oops.In("dix").
				With("op", "module_setup", "module", mod.name, "label", setup.meta.Label).
				Wrapf(err, "setup failed for module %s via %s", mod.name, setup.meta.Label)
			return false
		}
		if debugEnabled {
			rt.logMessage(ctx, EventLevelDebug, "module setup completed", "module", mod.name, "label", setup.meta.Label)
		}
		return true
	})
	return setupErr
}

func (p *buildPlan) runInvokes(ctx context.Context, rt *Runtime, debugEnabled bool) error {
	var buildErr error
	p.modules.Range(func(_ int, mod *moduleSpec) bool {
		buildErr = runModuleInvokes(ctx, mod, rt, debugEnabled)
		return buildErr == nil
	})
	return buildErr
}

func runModuleInvokes(ctx context.Context, mod *moduleSpec, rt *Runtime, debugEnabled bool) error {
	var invokeErr error
	mod.invokes.Range(func(_ int, invoke InvokeFunc) bool {
		if debugEnabled {
			rt.logMessage(ctx, EventLevelDebug, "running invoke",
				"module", mod.name,
				"label", invoke.meta.Label,
				"dependencies", serviceRefNames(invoke.meta.Dependencies),
				"raw", invoke.meta.Raw,
			)
		}
		invokeErr = invoke.apply(rt.container)
		if invokeErr == nil && debugEnabled {
			rt.logMessage(ctx, EventLevelDebug, "invoke completed", "module", mod.name, "label", invoke.meta.Label)
		}
		return invokeErr == nil
	})
	if invokeErr != nil {
		rt.logMessage(ctx, EventLevelError, "invoke failed", "module", mod.name, "error", invokeErr)
		return oops.In("dix").
			With("op", "module_invoke", "module", mod.name).
			Wrapf(invokeErr, "invoke failed in module %s", mod.name)
	}
	return nil
}
