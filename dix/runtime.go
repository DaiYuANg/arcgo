package dix

import (
	"context"
	"errors"
	"log/slog"

	"github.com/samber/do/v2"
)

func newRuntime(spec *appSpec, plan *buildPlan) *Runtime {
	logger := defaultLogger()
	if spec != nil && spec.logger != nil {
		logger = spec.logger
	}

	rt := &Runtime{
		spec:      spec,
		plan:      plan,
		container: newContainer(),
		lifecycle: newLifecycle(),
		logger:    logger,
		state:     AppStateCreated,
	}

	rt.container.logger = rt.logger
	rt.lifecycle.logger = rt.logger

	return rt
}

// Name returns the runtime application name.
func (r *Runtime) Name() string {
	if r == nil || r.spec == nil {
		return ""
	}
	return r.spec.meta.Name
}

// Profile returns the runtime application profile.
func (r *Runtime) Profile() Profile {
	if r == nil || r.spec == nil {
		return ""
	}
	return r.spec.profile
}

// Logger returns the runtime logger.
func (r *Runtime) Logger() *slog.Logger {
	if r == nil {
		return nil
	}
	return r.logger
}

// Meta returns the runtime application metadata.
func (r *Runtime) Meta() AppMeta {
	if r == nil || r.spec == nil {
		return AppMeta{}
	}
	return r.spec.meta
}

// State returns the current runtime state.
func (r *Runtime) State() AppState {
	if r == nil {
		return AppStateCreated
	}
	return r.state
}

// Container returns the runtime container wrapper.
func (r *Runtime) Container() *Container {
	if r == nil {
		return nil
	}
	return r.container
}

// Raw returns the underlying do injector for the runtime.
func (r *Runtime) Raw() do.Injector {
	if r == nil || r.container == nil {
		return nil
	}
	return r.container.Raw()
}

// Start executes lifecycle start hooks for the runtime.
func (r *Runtime) Start(ctx context.Context) error {
	if r == nil {
		return errors.New("runtime is nil")
	}
	if r.state != AppStateBuilt {
		return errors.New("runtime must be built before starting")
	}

	r.transitionState(ctx, AppStateStarting, "start requested")
	if r.logger.Enabled(ctx, slog.LevelInfo) {
		r.logger.Info("starting app", "app", r.Name())
	}
	startedHooks, err := r.lifecycle.executeStartHooks(ctx, r.container)
	if err != nil {
		if r.logger.Enabled(ctx, slog.LevelDebug) {
			r.logger.Debug("rolling back app start",
				"app", r.Name(),
				"started_hooks", startedHooks,
				"rollback_stop_hooks", startedHooks,
			)
		}
		rollbackErr := r.lifecycle.executeStopHooksSubset(ctx, startedHooks)
		shutdownReport := r.container.ShutdownReport(ctx)
		startErr := errors.Join(err, rollbackErr, shutdownReport)
		r.transitionState(ctx, AppStateStopped, "start failed")
		r.logger.Error("app start failed", "app", r.Name(), "error", startErr)
		return startErr
	}

	r.transitionState(ctx, AppStateStarted, "start completed")
	if r.logger.Enabled(ctx, slog.LevelInfo) {
		r.logger.Info("app started", "app", r.Name())
	}
	return nil
}

// Stop executes lifecycle stop hooks and shuts down the runtime.
func (r *Runtime) Stop(ctx context.Context) error {
	report, err := r.StopWithReport(ctx)
	if err != nil {
		return err
	}
	if report != nil {
		return report.Err()
	}
	return nil
}

// StopWithReport executes runtime shutdown and returns a detailed stop report.
func (r *Runtime) StopWithReport(ctx context.Context) (*StopReport, error) {
	if err := r.validateStoppable(); err != nil {
		return nil, err
	}

	infoEnabled := r.logger.Enabled(ctx, slog.LevelInfo)
	debugEnabled := r.logger.Enabled(ctx, slog.LevelDebug)
	if infoEnabled {
		r.logger.Info("stopping app", "app", r.Name())
	}
	if debugEnabled {
		r.logger.Debug("executing runtime stop",
			"app", r.Name(),
			"stop_hooks", r.lifecycle.stopHooks.Len(),
		)
	}

	report := r.executeStopSequence(ctx)
	r.logStopReport(debugEnabled, report)

	r.transitionState(ctx, AppStateStopped, "stop completed")
	if infoEnabled {
		r.logger.Info("app stopped", "app", r.Name())
	}

	stopErr := report.Err()
	return report, stopErr
}

func (r *Runtime) logDebugInformation() {
	if r == nil || r.spec == nil {
		return
	}

	if r.spec.debug.scopeTree {
		injector := do.ExplainInjector(r.container.Raw())
		r.logger.Info("do scope tree", "app", r.Name(), "tree", injector.String())
	}

	r.spec.debug.namedServiceDependencies.Range(func(name string) bool {
		if desc, found := do.ExplainNamedService(r.container.Raw(), name); found {
			r.logger.Info("do named service dependencies", "app", r.Name(), "name", name, "dependencies", desc.String())
		} else {
			r.logger.Warn("do named service not found", "app", r.Name(), "name", name)
		}
		return true
	})
}

func (r *Runtime) transitionState(ctx context.Context, next AppState, reason string) {
	if r == nil {
		return
	}
	prev := r.state
	r.state = next
	if r.logger != nil && prev != next && r.logger.Enabled(ctx, slog.LevelDebug) {
		r.logger.Debug("runtime state transition",
			"app", r.Name(),
			"from", prev.String(),
			"to", next.String(),
			"reason", reason,
		)
	}
}

func (r *Runtime) validateStoppable() error {
	if r == nil {
		return errors.New("runtime is nil")
	}
	if r.state == AppStateStarting {
		return errors.New("runtime is still starting")
	}
	if r.state != AppStateStarted {
		return errors.New("runtime must be started before stopping")
	}
	return nil
}

func (r *Runtime) executeStopSequence(ctx context.Context) *StopReport {
	report := &StopReport{}
	if err := r.lifecycle.executeStopHooks(ctx, r.container); err != nil {
		r.logger.Error("stop hooks failed", "app", r.Name(), "error", err)
		report.HookError = err
	}

	report.ShutdownReport = r.container.ShutdownReport(ctx)
	if report.ShutdownReport != nil && len(report.ShutdownReport.Errors) > 0 {
		r.logger.Error("container shutdown failed", "app", r.Name(), "error", report.ShutdownReport)
	}
	return report
}

func (r *Runtime) logStopReport(debugEnabled bool, report *StopReport) {
	if !debugEnabled {
		return
	}
	r.logger.Debug("runtime stop report",
		"app", r.Name(),
		"hook_error", report.HookError != nil,
		"shutdown_errors", shutdownErrorCount(report),
	)
}

func shutdownErrorCount(report *StopReport) int {
	if report == nil || report.ShutdownReport == nil {
		return 0
	}
	return len(report.ShutdownReport.Errors)
}
