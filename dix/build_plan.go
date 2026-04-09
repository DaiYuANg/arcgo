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

func newBuildPlan(app *App) (*buildPlan, error) {
	plan, _, err := computeBuildPlan(app)
	return plan, err
}

func newUnvalidatedBuildPlan(app *App) (*buildPlan, error) {
	if app == nil || app.spec == nil {
		return nil, oops.In("dix").
			With("op", "new_unvalidated_build_plan").
			New("app is nil")
	}

	modules, err := flattenModuleList(&app.spec.modules, app.spec.profile)
	if err != nil {
		logMessageEvent(context.Background(), app.spec.resolvedEventLogger(), EventLevelError, "module flatten failed", "app", app.Name(), "error", err)
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

func (p *buildPlan) Build() (_ *Runtime, err error) {
	startedAt := time.Now()
	var rt *Runtime
	defer func() {
		if p != nil && p.spec != nil {
			if rt != nil {
				emitEventLogger(context.Background(), rt.eventLogger, p.buildEvent(time.Since(startedAt), err))
				emitObservers(context.Background(), rt.logger, p.spec.observers, func(observer Observer) {
					observer.OnBuild(context.Background(), p.buildEvent(time.Since(startedAt), err))
				})
			} else {
				p.spec.emitBuild(context.Background(), p.buildEvent(time.Since(startedAt), err))
			}
		}
	}()

	if p == nil || p.spec == nil {
		err = oops.In("dix").
			With("op", "build_runtime").
			New("build plan is nil")
		return nil, err
	}

	rt = newRuntime(p.spec, p)
	registerRuntimeCoreServices(rt)

	providersRegistered, err := p.prepareBuildLogging(rt)
	if err != nil {
		err = cleanupBuildFailure(rt, err)
		return nil, err
	}

	debugEnabled := eventLoggerEnabled(rt.eventLogger, context.Background(), EventLevelDebug)
	infoEnabled := eventLoggerEnabled(rt.eventLogger, context.Background(), EventLevelInfo)
	p.logBuildStart(rt, infoEnabled, debugEnabled)

	if providersRegistered {
		p.logProviderRegistrations(rt, debugEnabled)
	} else {
		p.registerProviders(rt, debugEnabled)
	}

	if err := p.bindHooksAndRunSetups(rt, debugEnabled); err != nil {
		err = cleanupBuildFailure(rt, err)
		return nil, err
	}

	if err := p.runInvokes(rt, debugEnabled); err != nil {
		err = cleanupBuildFailure(rt, err)
		return nil, err
	}

	rt.transitionState(context.Background(), AppStateBuilt, "build completed")
	rt.logDebugInformation()
	return rt, nil
}

func (p *buildPlan) prepareBuildLogging(rt *Runtime) (bool, error) {
	if p == nil || p.spec == nil || rt == nil {
		return false, nil
	}
	if p.spec.eventLoggerFromContainer == nil && p.spec.loggerFromContainer == nil {
		return false, nil
	}

	p.registerProviders(rt, false)

	if p.spec.eventLoggerFromContainer != nil {
		resolvedEventLogger, err := p.resolveFrameworkEventLogger(rt)
		if err != nil {
			return true, err
		}
		rt.eventLogger = resolvedEventLogger
		rt.container.eventLogger = resolvedEventLogger
		rt.lifecycle.eventLogger = resolvedEventLogger
		return true, nil
	}

	if p.spec.loggerFromContainer != nil {
		resolvedLogger, err := p.resolveFrameworkLogger(rt)
		if err != nil {
			return true, err
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
	}

	return true, nil
}

func cleanupBuildFailure(rt *Runtime, buildErr error) error {
	if rt == nil || rt.container == nil {
		return buildErr
	}

	report := rt.container.ShutdownReport(context.Background())
	if report == nil || len(report.Errors) == 0 {
		return buildErr
	}
	rt.logMessage(context.Background(), EventLevelError, "build cleanup failed", "app", rt.Name(), "error", report)
	return errors.Join(buildErr, report)
}

func (p *buildPlan) logBuildStart(rt *Runtime, infoEnabled, debugEnabled bool) {
	if infoEnabled {
		rt.logMessage(context.Background(), EventLevelInfo, "building app", "app", p.spec.meta.Name, "profile", p.spec.profile)
	}
	if debugEnabled {
		rt.logMessage(context.Background(), EventLevelDebug, "build plan ready",
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

func (p *buildPlan) registerProviders(rt *Runtime, debugEnabled bool) {
	p.modules.Range(func(_ int, mod *moduleSpec) bool {
		if debugEnabled {
			rt.logMessage(context.Background(), EventLevelDebug, "registering module",
				"module", mod.name,
				"providers", mod.providers.Len(),
				"hooks", mod.hooks.Len(),
				"setups", mod.setups.Len(),
				"invokes", mod.invokes.Len(),
			)
		}
		mod.providers.Range(func(_ int, provider ProviderFunc) bool {
			if debugEnabled {
				rt.logMessage(context.Background(), EventLevelDebug, "registering provider",
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

func (p *buildPlan) logProviderRegistrations(rt *Runtime, debugEnabled bool) {
	if !debugEnabled {
		return
	}
	p.modules.Range(func(_ int, mod *moduleSpec) bool {
		rt.logMessage(context.Background(), EventLevelDebug, "registering module",
			"module", mod.name,
			"providers", mod.providers.Len(),
			"hooks", mod.hooks.Len(),
			"setups", mod.setups.Len(),
			"invokes", mod.invokes.Len(),
		)
		mod.providers.Range(func(_ int, provider ProviderFunc) bool {
			rt.logMessage(context.Background(), EventLevelDebug, "registering provider",
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

func (p *buildPlan) bindHooksAndRunSetups(rt *Runtime, debugEnabled bool) error {
	var setupErr error
	p.modules.Range(func(_ int, mod *moduleSpec) bool {
		bindModuleHooks(mod, rt, debugEnabled)
		setupErr = runModuleSetups(mod, rt, debugEnabled)
		return setupErr == nil
	})
	return setupErr
}

func bindModuleHooks(mod *moduleSpec, rt *Runtime, debugEnabled bool) {
	mod.hooks.Range(func(_ int, hook HookFunc) bool {
		if debugEnabled {
			rt.logMessage(context.Background(), EventLevelDebug, "binding lifecycle hook",
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

func runModuleSetups(mod *moduleSpec, rt *Runtime, debugEnabled bool) error {
	var setupErr error
	mod.setups.Range(func(_ int, setup SetupFunc) bool {
		if debugEnabled {
			rt.logMessage(context.Background(), EventLevelDebug, "running module setup",
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
			rt.logMessage(context.Background(), EventLevelError, "module setup failed", "module", mod.name, "label", setup.meta.Label, "error", err)
			setupErr = oops.In("dix").
				With("op", "module_setup", "module", mod.name, "label", setup.meta.Label).
				Wrapf(err, "setup failed for module %s via %s", mod.name, setup.meta.Label)
			return false
		}
		if debugEnabled {
			rt.logMessage(context.Background(), EventLevelDebug, "module setup completed", "module", mod.name, "label", setup.meta.Label)
		}
		return true
	})
	return setupErr
}

func (p *buildPlan) runInvokes(rt *Runtime, debugEnabled bool) error {
	var buildErr error
	p.modules.Range(func(_ int, mod *moduleSpec) bool {
		buildErr = runModuleInvokes(mod, rt, debugEnabled)
		return buildErr == nil
	})
	return buildErr
}

func runModuleInvokes(mod *moduleSpec, rt *Runtime, debugEnabled bool) error {
	var invokeErr error
	mod.invokes.Range(func(_ int, invoke InvokeFunc) bool {
		if debugEnabled {
			rt.logMessage(context.Background(), EventLevelDebug, "running invoke",
				"module", mod.name,
				"label", invoke.meta.Label,
				"dependencies", serviceRefNames(invoke.meta.Dependencies),
				"raw", invoke.meta.Raw,
			)
		}
		invokeErr = invoke.apply(rt.container)
		if invokeErr == nil && debugEnabled {
			rt.logMessage(context.Background(), EventLevelDebug, "invoke completed", "module", mod.name, "label", invoke.meta.Label)
		}
		return invokeErr == nil
	})
	if invokeErr != nil {
		rt.logMessage(context.Background(), EventLevelError, "invoke failed", "module", mod.name, "error", invokeErr)
		return oops.In("dix").
			With("op", "module_invoke", "module", mod.name).
			Wrapf(invokeErr, "invoke failed in module %s", mod.name)
	}
	return nil
}
