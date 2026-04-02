package dix

import (
	"context"
	"testing"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewModule_StoresCollectionBackedSpec(t *testing.T) {
	shared := NewModule("shared")
	mod := NewModule("feature",
		WithModuleProviders(
			Provider0(func() string { return "a" }),
			Provider0(func() string { return "b" }),
		),
		WithModuleInvokes(
			Invoke0(func() {}),
			Invoke0(func() {}),
		),
		WithModuleImports(shared, shared),
		WithModuleProfiles(ProfileDev, ProfileDev, ProfileProd),
		WithModuleExcludeProfiles(ProfileTest, ProfileTest),
		WithModuleTags("http", "http", "core"),
	)

	require.NotNil(t, mod.spec)
	assert.Equal(t, "feature", mod.Name())
	assert.Equal(t, 2, mod.spec.providers.Len())
	assert.Equal(t, 2, mod.spec.invokes.Len())
	assert.Equal(t, 2, mod.spec.imports.Len())
	assert.True(t, mod.spec.profiles.Contains(ProfileDev))
	assert.True(t, mod.spec.profiles.Contains(ProfileProd))
	assert.Equal(t, 2, mod.spec.profiles.Len())
	assert.True(t, mod.spec.excludeProfiles.Contains(ProfileTest))
	assert.Equal(t, 1, mod.spec.excludeProfiles.Len())
	assert.Equal(t, []string{"http", "core"}, mod.Tags().Values())
}

func TestWithDebugNamedServiceDependencies_Deduplicates(t *testing.T) {
	app := New("debug",
		WithDebugNamedServiceDependencies("tenant.default", "tenant.default", "tenant.backup"),
	)

	require.NotNil(t, app.spec)
	require.Equal(t, 2, app.spec.debug.namedServiceDependencies.Len())
	assert.Equal(t, []string{"tenant.default", "tenant.backup"}, app.spec.debug.namedServiceDependencies.Values())
}

func TestWithModules_StoresSpecModules(t *testing.T) {
	left := NewModule("left")
	right := NewModule("right")

	app := New("modules", WithModules(left, right))

	require.NotNil(t, app.spec)
	require.Equal(t, 2, app.spec.modules.Len())
	values := app.spec.modules.Values()
	require.Len(t, values, 2)
	assert.Equal(t, "left", values[0].Name())
	assert.Equal(t, "right", values[1].Name())
}

func TestContainerHealthChecks_UsesCollectionList(t *testing.T) {
	c := newContainer()

	c.RegisterHealthCheck("general", func(context.Context) error { return nil })
	c.RegisterLivenessCheck("live", func(context.Context) error { return nil })
	c.RegisterReadinessCheck("ready", func(context.Context) error { return nil })

	require.Equal(t, 3, c.healthChecks.Len())
	values := c.healthChecks.Values()
	require.Len(t, values, 3)
	assert.Equal(t, HealthKindGeneral, values[0].kind)
	assert.Equal(t, HealthKindLiveness, values[1].kind)
	assert.Equal(t, HealthKindReadiness, values[2].kind)
}

func TestLifecycle_UsesCollectionBackedHooks(t *testing.T) {
	lc := newLifecycle()
	order := make([]string, 0, 4)

	lc.OnStart(func(context.Context) error {
		order = append(order, "start-1")
		return nil
	})
	lc.OnStart(func(context.Context) error {
		order = append(order, "start-2")
		return nil
	})
	lc.OnStop(func(context.Context) error {
		order = append(order, "stop-2")
		return nil
	})
	lc.OnStop(func(context.Context) error {
		order = append(order, "stop-1")
		return nil
	})

	started, err := lc.executeStartHooks(context.Background(), newContainer())
	require.NoError(t, err)
	assert.Equal(t, 2, started)
	require.NoError(t, lc.executeStopHooks(context.Background(), newContainer()))
	assert.Equal(t, []string{"start-1", "start-2", "stop-1", "stop-2"}, order)
}

func TestFlattenModules_VisitorOrderIsDependencyFirst(t *testing.T) {
	shared := NewModule("shared")
	left := NewModule("left", WithModuleImports(shared))
	root := NewModule("root", WithModuleImports(left))

	modules, err := flattenModules(collectionx.NewList(root), ProfileDefault)
	require.NoError(t, err)
	require.Len(t, modules.Values(), 3)
	assert.Equal(t, []string{"shared", "left", "root"}, []string{
		modules.Values()[0].name,
		modules.Values()[1].name,
		modules.Values()[2].name,
	})
}

func TestWalkModules_DetectsImportCycle(t *testing.T) {
	left := &moduleSpec{name: "left"}
	right := &moduleSpec{name: "right"}

	left.imports.Add(Module{spec: right})
	right.imports.Add(Module{spec: left})

	err := walkModules(collectionx.NewList(Module{spec: left}), ProfileDefault, moduleVisitorFuncs{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "module import cycle detected")
	assert.Contains(t, err.Error(), "left")
	assert.Contains(t, err.Error(), "right")
}
