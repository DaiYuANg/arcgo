package dix_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DaiYuANg/arcgo/dix"
	dixadvanced "github.com/DaiYuANg/arcgo/dix/advanced"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingShutdownService struct{}

func (s *failingShutdownService) Shutdown() error {
	return errors.New("shutdown failed")
}

func TestRuntimeStopWithReportAggregatesErrors(t *testing.T) {
	app := dix.New("stop-report",
		dix.WithModule(
			dix.NewModule("runtime",
				dix.WithModuleProviders(
					dix.Provider0(func() *failingShutdownService { return &failingShutdownService{} }),
				),
				dix.WithModuleInvokes(
					dix.Invoke1(func(*failingShutdownService) {}),
				),
				dix.WithModuleHooks(
					dix.OnStop(func(_ context.Context, _ *failingShutdownService) error {
						return errors.New("stop hook failed")
					}),
				),
			),
		),
	)

	rt := buildRuntime(t, app)
	require.NoError(t, rt.Start(context.Background()))

	report, err := rt.StopWithReport(context.Background())
	require.Error(t, err)
	require.NotNil(t, report)
	require.ErrorContains(t, report.HookError, "stop hook failed")
	require.NotNil(t, report.ShutdownReport)
	assert.NotEmpty(t, report.ShutdownReport.Errors)
	assert.Equal(t, dix.AppStateStopped, rt.State())
}

func TestAdvancedOverrideValue(t *testing.T) {
	app := dix.New("override",
		dix.WithModule(
			dix.NewModule("override",
				dix.WithModuleProviders(
					dix.Provider0(func() string { return "base" }),
				),
				dix.WithModuleSetups(
					dixadvanced.Override0(func() string { return "override" }),
				),
			),
		),
	)

	require.NoError(t, app.Validate())
	rt := buildRuntime(t, app)

	value, err := dix.ResolveAs[string](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, "override", value)
}

func TestAdvancedTransientProvider(t *testing.T) {
	counter := 0
	rt := buildRuntime(t, dix.New("transient",
		dix.WithModule(
			dix.NewModule("transient",
				dix.WithModuleProviders(
					dixadvanced.TransientProvider0(func() int {
						counter++
						return counter
					}),
				),
			),
		),
	))

	first, err := dix.ResolveAs[int](rt.Container())
	require.NoError(t, err)
	second, err := dix.ResolveAs[int](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, 1, first)
	assert.Equal(t, 2, second)
}

func TestAdvancedOverrideTransient(t *testing.T) {
	counter := 100
	rt := buildRuntime(t, dix.New("override-transient",
		dix.WithModule(
			dix.NewModule("override-transient",
				dix.WithModuleProviders(
					dix.Provider0(func() int { return 1 }),
				),
				dix.WithModuleSetups(
					dixadvanced.OverrideTransient0(func() int {
						counter++
						return counter
					}),
				),
			),
		),
	))

	first, err := dix.ResolveAs[int](rt.Container())
	require.NoError(t, err)
	second, err := dix.ResolveAs[int](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, 101, first)
	assert.Equal(t, 102, second)
}

func TestAdvancedInspectRuntime(t *testing.T) {
	rt := buildRuntime(t, dix.New("inspect",
		dix.WithModule(
			dix.NewModule("inspect",
				dix.WithModuleProviders(
					dixadvanced.NamedValue("tenant.default", "public"),
				),
			),
		),
	))

	_, err := dixadvanced.ResolveNamedAs[string](rt.Container(), "tenant.default")
	require.NoError(t, err)

	report := dixadvanced.InspectRuntime(rt, "tenant.default")
	assert.NotEmpty(t, report.ScopeTree)
	assert.NotEmpty(t, report.ProvidedServices)
	assert.NotEmpty(t, report.InvokedServices)
	assert.Contains(t, report.NamedDependencies, "tenant.default")
}

func TestAdvancedInspectRuntimeWithOptions(t *testing.T) {
	rt := buildRuntime(t, dix.New("inspect-light",
		dix.WithModule(
			dix.NewModule("inspect-light",
				dix.WithModuleProviders(
					dixadvanced.NamedValue("tenant.default", "public"),
				),
			),
		),
	))

	report := dixadvanced.InspectRuntimeWithOptions(rt, dixadvanced.InspectOptions{
		IncludeNamedDeps: true,
	}, "tenant.default")

	assert.Empty(t, report.ScopeTree)
	assert.Nil(t, report.ProvidedServices)
	assert.Nil(t, report.InvokedServices)
	assert.Contains(t, report.NamedDependencies, "tenant.default")
}

func TestAdvancedScopeNamedHelpers(t *testing.T) {
	rt := buildRuntime(t, dix.New("scope-named",
		dix.WithModule(
			dix.NewModule("scope-named",
				dix.WithModuleProviders(
					dix.Provider0(func() string { return "root" }),
				),
			),
		),
	))

	scope := dixadvanced.Scope(rt, "named-scope", func(injector do.Injector) {
		dixadvanced.ProvideScopedNamed1[string, string](injector, "greeting", func(root string) string {
			return root + "-scoped"
		})
	})

	value, err := dixadvanced.ResolveScopedNamedAs[string](scope, "greeting")
	require.NoError(t, err)
	assert.Equal(t, "root-scoped", value)
}
