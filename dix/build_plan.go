package dix

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	collectionlist "github.com/DaiYuANg/arcgo/collectionx/list"
	"github.com/samber/do/v2"
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
		return nil, errors.New("app is nil")
	}

	modules, err := flattenModuleList(&app.spec.modules, app.spec.profile)
	if err != nil {
		logger := app.spec.logger
		if logger != nil {
			logger.Error("module flatten failed", "app", app.Name(), "error", err)
		}
		return nil, fmt.Errorf("module flatten failed: %w", err)
	}

	plan := &buildPlan{
		spec:    app.spec,
		modules: modules,
	}

	return plan, nil
}

func (p *buildPlan) Build() (*Runtime, error) {
	if p == nil || p.spec == nil {
		return nil, errors.New("build plan is nil")
	}

	logger := p.spec.logger
	if logger == nil {
		logger = slog.Default()
	}

	rt := newRuntime(p.spec, p)
	registerRuntimeCoreServices(rt)

	logger, providersRegistered, err := p.prepareBuildLogger(rt, logger)
	if err != nil {
		return nil, cleanupBuildFailure(rt, logger, err)
	}

	debugEnabled := logger.Enabled(context.Background(), slog.LevelDebug)
	infoEnabled := logger.Enabled(context.Background(), slog.LevelInfo)
	p.logBuildStart(logger, infoEnabled, debugEnabled)

	if providersRegistered {
		p.logProviderRegistrations(logger, debugEnabled)
	} else {
		p.registerProviders(rt, logger, debugEnabled)
	}

	if err := p.bindHooksAndRunSetups(rt, logger, debugEnabled); err != nil {
		return nil, cleanupBuildFailure(rt, logger, err)
	}

	if err := p.runInvokes(rt, logger, debugEnabled); err != nil {
		return nil, cleanupBuildFailure(rt, logger, err)
	}

	rt.state = AppStateBuilt
	if infoEnabled {
		logger.Info("app built", "app", rt.Name(), "modules", p.modules.Len())
	}
	rt.logDebugInformation()
	return rt, nil
}

func (p *buildPlan) prepareBuildLogger(rt *Runtime, logger *slog.Logger) (*slog.Logger, bool, error) {
	if p == nil || p.spec == nil || p.spec.loggerFromContainer == nil {
		return logger, false, nil
	}

	p.registerProviders(rt, nil, false)
	resolvedLogger, err := p.resolveFrameworkLogger(rt)
	if err != nil {
		return logger, true, err
	}

	return resolvedLogger, true, nil
}

func cleanupBuildFailure(rt *Runtime, logger *slog.Logger, buildErr error) error {
	if rt == nil || rt.container == nil {
		return buildErr
	}

	report := rt.container.ShutdownReport(context.Background())
	if report == nil || len(report.Errors) == 0 {
		return buildErr
	}
	if logger != nil {
		logger.Error("build cleanup failed", "app", rt.Name(), "error", report)
	}
	return errors.Join(buildErr, report)
}

func (p *buildPlan) logBuildStart(logger *slog.Logger, infoEnabled, debugEnabled bool) {
	if infoEnabled {
		logger.Info("building app", "app", p.spec.meta.Name, "profile", p.spec.profile)
	}
	if debugEnabled {
		logger.Debug("build plan ready",
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
		return nil, errors.New("resolve framework logger failed: resolver is not configured")
	}

	logger, err := p.spec.loggerFromContainer(rt.container)
	if err != nil {
		return nil, fmt.Errorf("resolve framework logger failed: %w", err)
	}
	if logger == nil {
		return nil, errors.New("resolve framework logger failed: resolver returned nil logger")
	}

	rt.logger = logger
	rt.container.logger = logger
	rt.lifecycle.logger = logger
	do.OverrideNamedValue(rt.container.Raw(), serviceNameOf[*slog.Logger](), logger)
	return logger, nil
}

func (p *buildPlan) registerProviders(rt *Runtime, logger *slog.Logger, debugEnabled bool) {
	p.modules.Range(func(_ int, mod *moduleSpec) bool {
		if debugEnabled {
			logger.Debug("registering module",
				"module", mod.name,
				"providers", mod.providers.Len(),
				"hooks", mod.hooks.Len(),
				"setups", mod.setups.Len(),
				"invokes", mod.invokes.Len(),
			)
		}
		mod.providers.Range(func(_ int, provider ProviderFunc) bool {
			if debugEnabled {
				logger.Debug("registering provider",
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

func (p *buildPlan) logProviderRegistrations(logger *slog.Logger, debugEnabled bool) {
	if !debugEnabled {
		return
	}
	p.modules.Range(func(_ int, mod *moduleSpec) bool {
		logger.Debug("registering module",
			"module", mod.name,
			"providers", mod.providers.Len(),
			"hooks", mod.hooks.Len(),
			"setups", mod.setups.Len(),
			"invokes", mod.invokes.Len(),
		)
		mod.providers.Range(func(_ int, provider ProviderFunc) bool {
			logger.Debug("registering provider",
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

func (p *buildPlan) bindHooksAndRunSetups(rt *Runtime, logger *slog.Logger, debugEnabled bool) error {
	var setupErr error
	p.modules.Range(func(_ int, mod *moduleSpec) bool {
		bindModuleHooks(mod, rt, logger, debugEnabled)
		setupErr = runModuleSetups(mod, rt, logger, debugEnabled)
		return setupErr == nil
	})
	return setupErr
}

func bindModuleHooks(mod *moduleSpec, rt *Runtime, logger *slog.Logger, debugEnabled bool) {
	mod.hooks.Range(func(_ int, hook HookFunc) bool {
		if debugEnabled {
			logger.Debug("binding lifecycle hook",
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

func runModuleSetups(mod *moduleSpec, rt *Runtime, logger *slog.Logger, debugEnabled bool) error {
	var setupErr error
	mod.setups.Range(func(_ int, setup SetupFunc) bool {
		if debugEnabled {
			logger.Debug("running module setup",
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
			logger.Error("module setup failed", "module", mod.name, "label", setup.meta.Label, "error", err)
			setupErr = fmt.Errorf("setup failed for module %s via %s: %w", mod.name, setup.meta.Label, err)
			return false
		}
		if debugEnabled {
			logger.Debug("module setup completed", "module", mod.name, "label", setup.meta.Label)
		}
		return true
	})
	return setupErr
}

func (p *buildPlan) runInvokes(rt *Runtime, logger *slog.Logger, debugEnabled bool) error {
	var buildErr error
	p.modules.Range(func(_ int, mod *moduleSpec) bool {
		buildErr = runModuleInvokes(mod, rt, logger, debugEnabled)
		return buildErr == nil
	})
	return buildErr
}

func runModuleInvokes(mod *moduleSpec, rt *Runtime, logger *slog.Logger, debugEnabled bool) error {
	var invokeErr error
	mod.invokes.Range(func(_ int, invoke InvokeFunc) bool {
		if debugEnabled {
			logger.Debug("running invoke",
				"module", mod.name,
				"label", invoke.meta.Label,
				"dependencies", serviceRefNames(invoke.meta.Dependencies),
				"raw", invoke.meta.Raw,
			)
		}
		invokeErr = invoke.apply(rt.container)
		if invokeErr == nil && debugEnabled {
			logger.Debug("invoke completed", "module", mod.name, "label", invoke.meta.Label)
		}
		return invokeErr == nil
	})
	if invokeErr != nil {
		logger.Error("invoke failed", "module", mod.name, "error", invokeErr)
		return fmt.Errorf("invoke failed in module %s: %w", mod.name, invokeErr)
	}
	return nil
}
