package dix_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type frameworkLoggerCarrier struct {
	logger *slog.Logger
}

func TestBuildDebugLogging(t *testing.T) {
	logger, buf := newDebugLogger()
	app := dix.New("debug-build",
		dix.WithLogger(logger),
		dix.WithModule(
			dix.NewModule("debug",
				dix.WithModuleProviders(
					dix.Provider0(func() string { return "value" }),
				),
				dix.WithModuleHooks(
					dix.OnStart(func(context.Context, string) error { return nil }),
				),
				dix.WithModuleSetups(
					dix.SetupWithMetadata(func(*dix.Container, dix.Lifecycle) error { return nil }, dix.SetupMetadata{
						Label:        "DebugSetup",
						Dependencies: dix.ServiceRefs(dix.TypedService[string]()),
					}),
				),
				dix.WithModuleInvokes(
					dix.Invoke1(func(string) {}),
				),
			),
		),
	)

	_, err := app.Build()
	require.NoError(t, err)

	logs := buf.String()
	assert.True(t, strings.Contains(logs, "build plan ready"), logs)
	assert.True(t, strings.Contains(logs, "registering provider"), logs)
	assert.True(t, strings.Contains(logs, "binding lifecycle hook"), logs)
	assert.True(t, strings.Contains(logs, "module setup completed"), logs)
	assert.True(t, strings.Contains(logs, "invoke completed"), logs)
}

func TestRuntimeStartRollbackDebugLogging(t *testing.T) {
	logger, buf := newDebugLogger()
	app := dix.New("debug-start",
		dix.WithLogger(logger),
		dix.WithModule(
			dix.NewModule("debug-start",
				dix.WithModuleProviders(
					dix.Provider0(func() string { return "value" }),
				),
				dix.WithModuleHooks(
					dix.OnStart(func(context.Context, string) error { return nil }),
					dix.OnStop(func(context.Context, string) error { return nil }),
					dix.OnStart0(func(context.Context) error { return errors.New("boom") }),
				),
			),
		),
	)

	rt := buildRuntime(t, app)
	err := rt.Start(context.Background())
	require.Error(t, err)

	logs := buf.String()
	assert.True(t, strings.Contains(logs, "runtime state transition"), logs)
	assert.True(t, strings.Contains(logs, "executing start hook"), logs)
	assert.True(t, strings.Contains(logs, "rolling back app start"), logs)
	assert.True(t, strings.Contains(logs, "executing stop hook"), logs)
	assert.True(t, strings.Contains(logs, "shutting down container"), logs)
}

func TestHealthCheckReport(t *testing.T) {
	module := dix.NewModule("health",
		dix.WithModuleSetup(func(c *dix.Container, _ dix.Lifecycle) error {
			c.RegisterHealthCheck("db", func(_ context.Context) error { return nil })
			c.RegisterHealthCheck("cache", func(_ context.Context) error { return errors.New("down") })
			return nil
		}),
	)

	rt := buildRuntime(t, dix.NewApp("test", module))
	report := rt.CheckHealth(context.Background())
	assert.False(t, report.Healthy())
	require.Error(t, report.Error())
	assert.Contains(t, report.Error().Error(), "cache")
}

func TestRuntime_HealthHandlers(t *testing.T) {
	module := dix.NewModule("health",
		dix.WithModuleSetup(func(c *dix.Container, _ dix.Lifecycle) error {
			c.RegisterLivenessCheck("live", func(_ context.Context) error { return nil })
			c.RegisterReadinessCheck("ready", func(_ context.Context) error { return errors.New("booting") })
			return nil
		}),
	)

	rt := buildRuntime(t, dix.NewApp("health", module))
	reqCtx := context.Background()

	liveReq := httptest.NewRequestWithContext(reqCtx, http.MethodGet, "/livez", http.NoBody)
	liveRes := httptest.NewRecorder()
	rt.LivenessHandler()(liveRes, liveReq)
	assert.Equal(t, http.StatusOK, liveRes.Code)

	readyReq := httptest.NewRequestWithContext(reqCtx, http.MethodGet, "/readyz", http.NoBody)
	readyRes := httptest.NewRecorder()
	rt.ReadinessHandler()(readyRes, readyReq)
	assert.Equal(t, http.StatusServiceUnavailable, readyRes.Code)
}

func TestNew_WithModulesOption(t *testing.T) {
	rt := buildRuntime(t, dix.New("test",
		dix.WithProfile(dix.ProfileDev),
		dix.WithModule(DatabaseModule),
	))

	logger, err := dix.ResolveAs[*slog.Logger](rt.Container())
	require.NoError(t, err)
	assert.NotNil(t, logger)
}

func TestHealthKinds(t *testing.T) {
	mod := dix.NewModule("health",
		dix.WithModuleSetup(func(c *dix.Container, _ dix.Lifecycle) error {
			c.RegisterLivenessCheck("live", func(_ context.Context) error { return nil })
			c.RegisterReadinessCheck("ready", func(_ context.Context) error { return nil })
			return nil
		}),
	)

	rt := buildRuntime(t, dix.New("health-app", dix.WithModule(mod)))
	live := rt.CheckLiveness(context.Background())
	ready := rt.CheckReadiness(context.Background())

	assert.True(t, live.Healthy())
	assert.True(t, ready.Healthy())
	require.NotNil(t, live.Checks)
	require.NotNil(t, ready.Checks)
	assert.Equal(t, 1, live.Checks.Len())
	assert.Equal(t, 1, ready.Checks.Len())
}

func TestNewDefault(t *testing.T) {
	app := dix.NewDefault()
	assert.Equal(t, dix.DefaultAppName, app.Name())
}

func TestApp_StartBuildsAndStartsRuntime(t *testing.T) {
	app := dix.New("start",
		dix.WithModule(
			dix.NewModule("start",
				dix.WithModuleProvider(dix.Provider0(func() string { return "value" })),
				dix.WithModuleHook(dix.OnStart(func(context.Context, string) error { return nil })),
			),
		),
	)

	rt, err := app.Start(context.Background())
	require.NoError(t, err)
	assert.Equal(t, dix.AppStateStarted, rt.State())

	value, err := dix.ResolveAs[string](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, "value", value)

	require.NoError(t, rt.Stop(context.Background()))
}

func TestWithLoggerFrom1_UsesDIProvidedLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	diLogger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	app := dix.New("di-logger",
		dix.WithLoggerFrom1(func(carrier *frameworkLoggerCarrier) *slog.Logger {
			return carrier.logger
		}),
		dix.WithModule(
			dix.NewModule("logger",
				dix.WithModuleProviders(
					dix.Provider0(func() *frameworkLoggerCarrier {
						return &frameworkLoggerCarrier{logger: diLogger}
					}),
				),
				dix.WithModuleHooks(
					dix.OnStart0(func(context.Context) error { return nil }),
					dix.OnStop0(func(context.Context) error { return nil }),
				),
			),
		),
	)

	rt := buildRuntime(t, app)

	resolved, err := dix.ResolveAs[*slog.Logger](rt.Container())
	require.NoError(t, err)
	assert.Same(t, diLogger, rt.Logger())
	assert.Same(t, diLogger, resolved)

	require.NoError(t, rt.Start(context.Background()))
	require.NoError(t, rt.Stop(context.Background()))

	logs := buf.String()
	assert.Contains(t, logs, "building app")
	assert.Contains(t, logs, "registering provider")
	assert.Contains(t, logs, "starting app")
	assert.Contains(t, logs, "app stopped")
}

func TestWithLoggerFrom1_MissingDependencyFailsBuild(t *testing.T) {
	app := dix.New("di-logger-missing",
		dix.WithLoggerFrom1(func(*frameworkLoggerCarrier) *slog.Logger {
			return slog.Default()
		}),
	)

	_, err := app.Build()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve framework logger failed")
}
