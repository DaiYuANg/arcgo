package dix_test

import (
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
						Dependencies: []dix.ServiceRef{dix.TypedService[string]()},
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
