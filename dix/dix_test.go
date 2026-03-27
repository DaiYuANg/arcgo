package dix_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/DaiYuANg/arcgo/dix"
	dixadvanced "github.com/DaiYuANg/arcgo/dix/advanced"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Example providers

type Config struct {
	DSN  string
	Port int
}

func ProvideConfig() Config {
	return Config{
		DSN:  "sqlite://test.db",
		Port: 8080,
	}
}

type Database struct {
	dsn string
}

func NewDatabase(dsn string) *Database {
	return &Database{dsn: dsn}
}

func (d *Database) Connect() error {
	return nil
}

type testGreeter interface {
	Greet() string
}

type testGreeterImpl struct{}

func (g *testGreeterImpl) Greet() string {
	return "hello"
}

type cleanupService struct {
	shutdowns int
}

func (s *cleanupService) Shutdown() error {
	s.shutdowns++
	return nil
}

func (d *Database) Close() error {
	return nil
}

func ProvideDatabase(cfg Config) *Database {
	return NewDatabase(cfg.DSN)
}

type Server struct {
	addr string
}

func ProvideServer(cfg Config) *Server {
	return &Server{
		addr: fmt.Sprintf(":%d", cfg.Port),
	}
}

func (s *Server) Start() error {
	return nil
}

func (s *Server) Stop(_ context.Context) error {
	return nil
}

// Example modules

var DatabaseModule = dix.NewModule("database",
	dix.WithModuleProviders(
		dix.Provider0(ProvideConfig),
		dix.Provider1(ProvideDatabase),
	),
	dix.WithModuleHooks(
		dix.OnStart(func(_ context.Context, db *Database) error {
			return db.Connect()
		}),
		dix.OnStop(func(_ context.Context, db *Database) error {
			return db.Close()
		}),
	),
)

var ServerModule = dix.NewModule("server",
	dix.WithModuleProviders(
		dix.Provider1(ProvideServer),
	),
	dix.WithModuleImports(DatabaseModule),
	dix.WithModuleHooks(
		dix.OnStart(func(_ context.Context, s *Server) error {
			return s.Start()
		}),
		dix.OnStop(func(ctx context.Context, s *Server) error {
			return s.Stop(ctx)
		}),
	),
)

func buildRuntime(t *testing.T, app *dix.App) *dix.Runtime {
	t.Helper()
	rt, err := app.Build()
	require.NoError(t, err)
	require.NotNil(t, rt)
	return rt
}

func newDebugLogger() (*slog.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	return logger, buf
}

// Tests

func TestApp_Build(t *testing.T) {
	app := dix.NewApp("testapp", ServerModule)
	rt := buildRuntime(t, app)

	db, err := dix.ResolveAs[*Database](rt.Container())
	require.NoError(t, err)
	assert.NotNil(t, db)
	assert.Equal(t, "sqlite://test.db", db.dsn)

	cfg, err := dix.ResolveAs[Config](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, 8080, cfg.Port)
	assert.Equal(t, dix.AppStateBuilt, rt.State())
}

func TestApp_BuildCreatesIndependentRuntimes(t *testing.T) {
	app := dix.NewApp("testapp", ServerModule)

	first := buildRuntime(t, app)
	second := buildRuntime(t, app)

	assert.NotSame(t, first, second)
	assert.NotSame(t, first.Container(), second.Container())
	assert.Equal(t, first.Name(), second.Name())
}

func TestRuntime_StartStop(t *testing.T) {
	rt := buildRuntime(t, dix.NewApp("testapp", ServerModule))
	ctx := context.Background()

	require.Equal(t, dix.AppStateBuilt, rt.State())
	require.NoError(t, rt.Start(ctx))
	assert.Equal(t, dix.AppStateStarted, rt.State())

	require.NoError(t, rt.Stop(ctx))
	assert.Equal(t, dix.AppStateStopped, rt.State())
}

func TestModule_WithProfiles(t *testing.T) {
	devOnlyCalled := false
	prodOnlyCalled := false

	devModule := dix.NewModule("dev-only",
		dix.WithModuleProfiles(dix.ProfileDev),
		dix.WithModuleProviders(
			dix.Provider0(func() string {
				devOnlyCalled = true
				return "dev"
			}),
		),
	)

	prodModule := dix.NewModule("prod-only",
		dix.WithModuleProfiles(dix.ProfileProd),
		dix.WithModuleProviders(
			dix.Provider0(func() string {
				prodOnlyCalled = true
				return "prod"
			}),
		),
	)

	appDev := dix.New("test",
		dix.WithProfile(dix.ProfileDev),
		dix.WithModules(devModule, prodModule),
	)

	rtDev := buildRuntime(t, appDev)
	devStr, err := dix.ResolveAs[string](rtDev.Container())
	require.NoError(t, err)
	assert.Equal(t, "dev", devStr)
	assert.True(t, devOnlyCalled)
	assert.False(t, prodOnlyCalled)

	appProd := dix.New("test",
		dix.WithProfile(dix.ProfileProd),
		dix.WithModules(devModule, prodModule),
	)

	rtProd := buildRuntime(t, appProd)
	prodStr, err := dix.ResolveAs[string](rtProd.Container())
	require.NoError(t, err)
	assert.Equal(t, "prod", prodStr)
}

func TestModule_WithExcludeProfiles(t *testing.T) {
	called := false

	module := dix.NewModule("not-test",
		dix.WithModuleExcludeProfiles(dix.ProfileTest),
		dix.WithModuleProviders(
			dix.Provider0(func() string {
				called = true
				return "value"
			}),
		),
	)

	appTest := dix.New("test",
		dix.WithProfile(dix.ProfileTest),
		dix.WithModule(module),
	)

	rt := buildRuntime(t, appTest)
	_, err := dix.ResolveAs[string](rt.Container())
	assert.Error(t, err)
	assert.False(t, called)
}

func TestResolveAs(t *testing.T) {
	rt := buildRuntime(t, dix.NewApp("test",
		dix.NewModule("test",
			dix.WithModuleProviders(
				dix.Provider0(ProvideConfig),
			),
		),
	))

	cfg, err := dix.ResolveAs[Config](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, 8080, cfg.Port)
}

func TestMustResolveAs(t *testing.T) {
	rt := buildRuntime(t, dix.NewApp("test",
		dix.NewModule("test",
			dix.WithModuleProviders(
				dix.Provider0(ProvideConfig),
			),
		),
	))

	cfg := dix.MustResolveAs[Config](rt.Container())
	assert.Equal(t, 8080, cfg.Port)
}

func TestInvoke(t *testing.T) {
	invoked := false

	rt := buildRuntime(t, dix.NewApp("test",
		dix.NewModule("test",
			dix.WithModuleProviders(
				dix.Provider0(ProvideConfig),
			),
			dix.WithModuleInvokes(
				dix.Invoke1(func(cfg Config) {
					invoked = true
					assert.Equal(t, 8080, cfg.Port)
				}),
			),
		),
	))

	assert.True(t, invoked)
	assert.NotNil(t, rt)
}

func TestModule_Imports(t *testing.T) {
	baseModule := dix.NewModule("base",
		dix.WithModuleProviders(
			dix.Provider0(ProvideConfig),
		),
	)

	dependentModule := dix.NewModule("dependent",
		dix.WithModuleImports(baseModule),
		dix.WithModuleProviders(
			dix.Provider1(ProvideDatabase),
		),
	)

	rt := buildRuntime(t, dix.NewApp("test", dependentModule))

	cfg, err := dix.ResolveAs[Config](rt.Container())
	require.NoError(t, err)

	db, err := dix.ResolveAs[*Database](rt.Container())
	require.NoError(t, err)

	assert.NotNil(t, cfg)
	assert.NotNil(t, db)
}

func TestModule_ImportDeduplicatesSharedDependency(t *testing.T) {
	called := 0

	shared := dix.NewModule("shared",
		dix.WithModuleProviders(
			dix.Provider0(func() string {
				called++
				return "shared"
			}),
		),
	)

	left := dix.NewModule("left", dix.WithModuleImports(shared))
	right := dix.NewModule("right", dix.WithModuleImports(shared))

	rt := buildRuntime(t, dix.NewApp("test", left, right))
	value, err := dix.ResolveAs[string](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, "shared", value)
	assert.Equal(t, 1, called)
}

func TestApp_ValidateDetectsDuplicateModuleNames(t *testing.T) {
	app := dix.NewApp("duplicate-modules",
		dix.NewModule("shared",
			dix.WithModuleProviders(
				dix.Provider0(func() string { return "left" }),
			),
		),
		dix.NewModule("shared",
			dix.WithModuleProviders(
				dix.Provider0(func() int { return 42 }),
			),
		),
	)

	err := app.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate module name detected: shared")
}

func TestApp_ValidateGraph(t *testing.T) {
	app := dix.NewApp("test", DatabaseModule, ServerModule)
	require.NoError(t, app.Validate())
}

func TestApp_ValidateAllowsProviderDependencyDeclaredLater(t *testing.T) {
	app := dix.NewApp("validate-order",
		dix.NewModule("ordered",
			dix.WithModuleProviders(
				dix.Provider1(ProvideDatabase),
				dix.Provider0(ProvideConfig),
			),
		),
	)

	require.NoError(t, app.Validate())

	rt := buildRuntime(t, app)
	db, err := dix.ResolveAs[*Database](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, "sqlite://test.db", db.dsn)
}

func TestApp_ValidateDetectsMissingDependency(t *testing.T) {
	app := dix.NewApp("validate-missing",
		dix.NewModule("broken",
			dix.WithModuleProviders(
				dix.Provider1(func(cfg Config) *Database {
					return &Database{dsn: cfg.DSN}
				}),
			),
		),
	)

	err := app.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), do.NameOf[Config]())
}

func TestApp_ValidateDetectsDuplicateProvider(t *testing.T) {
	app := dix.NewApp("validate-duplicate",
		dix.NewModule("dup",
			dix.WithModuleProviders(
				dix.Provider0(ProvideConfig),
				dix.Provider0(ProvideConfig),
			),
		),
	)

	err := app.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate provider output")
	assert.Contains(t, err.Error(), do.NameOf[Config]())
}

func TestApp_ValidateDoesNotEscapeForCoreSetup(t *testing.T) {
	app := dix.NewApp("validate-setup",
		dix.NewModule("health",
			dix.WithModuleSetup(func(c *dix.Container, _ dix.Lifecycle) error {
				c.RegisterHealthCheck("ok", func(_ context.Context) error { return nil })
				return nil
			}),
			dix.WithModuleProviders(
				dix.Provider1(func(cfg Config) *Database {
					return &Database{dsn: cfg.DSN}
				}),
			),
		),
	)

	err := app.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), do.NameOf[Config]())
}

func TestApp_ValidateAdvancedAliasDependency(t *testing.T) {
	app := dix.NewApp("validate-alias",
		dix.NewModule("alias",
			dix.WithModuleProviders(
				dix.Provider0(func() *testGreeterImpl { return &testGreeterImpl{} }),
			),
			dix.WithModuleSetups(
				dixadvanced.BindAlias[*testGreeterImpl, testGreeter](),
			),
			dix.WithModuleInvokes(
				dix.Invoke1(func(testGreeter) {}),
			),
		),
	)

	require.NoError(t, app.Validate())
}

func TestProvider4AndAggregateDependencyStruct(t *testing.T) {
	type Params struct {
		Config Config
		DB     *Database
		Server *Server
		Label  string
	}

	module := dix.NewModule("test",
		dix.WithModuleProviders(
			dix.Provider0(ProvideConfig),
			dix.Provider1(ProvideDatabase),
			dix.Provider1(ProvideServer),
			dix.Provider0(func() string { return "ok" }),
			dix.Provider4(func(cfg Config, db *Database, srv *Server, label string) Params {
				return Params{Config: cfg, DB: db, Server: srv, Label: label}
			}),
		),
	)

	rt := buildRuntime(t, dix.NewApp("test", module))
	params, err := dix.ResolveAs[Params](rt.Container())
	require.NoError(t, err)
	assert.Equal(t, 8080, params.Config.Port)
	assert.Equal(t, "sqlite://test.db", params.DB.dsn)
	assert.Equal(t, ":8080", params.Server.addr)
	assert.Equal(t, "ok", params.Label)
}

func TestLifecycleHookReceivesContext(t *testing.T) {
	type ctxKey string

	const key ctxKey = "trace"
	received := ""

	module := dix.NewModule("ctx",
		dix.WithModuleProviders(dix.Provider0(func() string { return "value" })),
		dix.WithModuleHooks(
			dix.OnStart(func(ctx context.Context, value string) error {
				trace, ok := ctx.Value(key).(string)
				if !ok {
					return errors.New("trace context value missing")
				}
				received = trace + ":" + value
				return nil
			}),
		),
	)

	rt := buildRuntime(t, dix.NewApp("test", module))
	require.NoError(t, rt.Start(context.WithValue(context.Background(), key, "abc")))
	require.NoError(t, rt.Stop(context.Background()))
	assert.Equal(t, "abc:value", received)
}

func TestContainerRegisterProviderDefinitionReturnsError(t *testing.T) {
	rt := buildRuntime(t, dix.NewApp("test"))
	err := rt.Container().Register(dix.Definition{Kind: dix.DefinitionProvider})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}

// Examples

func ExampleApp() {
	app := dix.NewApp("myapp",
		DatabaseModule,
		ServerModule,
	)
	_ = app
}

func ExampleNewModule() {
	module := dix.NewModule("example",
		dix.WithModuleProviders(
			dix.Provider0(func() string { return "hello" }),
			dix.Provider1(func(s string) int { return len(s) }),
		),
		dix.WithModuleSetup(func(c *dix.Container, lc dix.Lifecycle) error {
			lc.OnStart(func(_ context.Context) error {
				s, err := dix.ResolveAs[string](c)
				if err != nil {
					return err
				}
				_, err = fmt.Println("Starting with:", s)
				if err != nil {
					return fmt.Errorf("print startup message: %w", err)
				}
				return nil
			})
			return nil
		}),
	)

	app := dix.NewApp("test", module)
	if _, err := app.Build(); err != nil {
		panic(err)
	}
}

func ExampleWithModuleHooks() {
	module := dix.NewModule("example",
		dix.WithModuleProviders(
			dix.Provider0(func() *http.Server {
				return &http.Server{Addr: ":8080", ReadHeaderTimeout: time.Second}
			}),
		),
		dix.WithModuleHooks(
			dix.OnStart(func(_ context.Context, s *http.Server) error {
				go func() {
					if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
						panic(err)
					}
				}()
				return nil
			}),
			dix.OnStop(func(ctx context.Context, s *http.Server) error {
				return s.Shutdown(ctx)
			}),
		),
	)

	app := dix.NewApp("test", module)
	_ = app
}

func TestResolveOptionalAs(t *testing.T) {
	rt := buildRuntime(t, dix.NewApp("test"))
	_, ok := dix.ResolveOptionalAs[string](rt.Container())
	assert.False(t, ok)
}

func TestResolveOrElse(t *testing.T) {
	rt := buildRuntime(t, dix.NewApp("test",
		dix.NewModule("deps",
			dix.WithModuleProviders(
				dix.Provider0(func() string { return "configured" }),
			),
		),
	))

	assert.Equal(t, "configured", dix.ResolveOrElse[string](rt.Container(), "fallback"))
	assert.Equal(t, 42, dix.ResolveOrElse[int](rt.Container(), 42))
}

func TestProfileFromEnv(t *testing.T) {
	t.Setenv("ARCGO_DIX_PROFILE", string(dix.ProfileDev))
	assert.Equal(t, dix.ProfileDev, dix.ProfileFromEnv("ARCGO_DIX_PROFILE", dix.ProfileProd))

	t.Setenv("ARCGO_DIX_PROFILE", "invalid")
	assert.Equal(t, dix.ProfileProd, dix.ProfileFromEnv("ARCGO_DIX_PROFILE", dix.ProfileProd))
}

func TestWithDoSetup(t *testing.T) {
	called := false
	module := dix.NewModule("advanced",
		dix.WithModuleSetups(
			dixadvanced.DoSetup(func(raw do.Injector) error {
				called = raw != nil
				return nil
			}),
		),
	)
	buildRuntime(t, dix.NewApp("test", module))
	assert.True(t, called)
}

func TestValidateReportWarnsForUndeclaredRawEscapes(t *testing.T) {
	module := dix.NewModule("advanced",
		dix.WithModuleProviders(
			dix.RawProvider(func(*dix.Container) {}),
		),
		dix.WithModuleInvokes(
			dix.RawInvoke(func(*dix.Container) error { return nil }),
		),
		dix.WithModuleHooks(
			dix.RawHook(func(*dix.Container, dix.Lifecycle) {}),
		),
		dix.WithModuleSetups(
			dixadvanced.DoSetup(func(raw do.Injector) error {
				_ = raw
				return nil
			}),
		),
	)

	report := dix.NewApp("warnings", module).ValidateReport()
	require.False(t, report.HasErrors())
	require.True(t, report.HasWarnings())
	assert.Contains(t, report.WarningSummary(), string(dix.ValidationWarningRawProviderUndeclaredOutput))
	assert.Contains(t, report.WarningSummary(), string(dix.ValidationWarningRawInvokeUndeclaredDeps))
	assert.Contains(t, report.WarningSummary(), string(dix.ValidationWarningRawHookUndeclaredDeps))
	assert.Contains(t, report.WarningSummary(), string(dix.ValidationWarningRawSetupUndeclaredGraph))
}

func TestValidateReportUsesDeclaredRawMetadata(t *testing.T) {
	module := dix.NewModule("advanced",
		dix.WithModuleProviders(
			dix.Provider0(ProvideConfig),
			dix.RawProviderWithMetadata(func(c *dix.Container) {
				dix.ProvideValueT(c, &Database{dsn: "sqlite://raw.db"})
			}, dix.ProviderMetadata{
				Label:        "RawDatabaseProvider",
				Output:       dix.TypedService[*Database](),
				Dependencies: []dix.ServiceRef{dix.TypedService[Config]()},
			}),
		),
		dix.WithModuleInvokes(
			dix.RawInvokeWithMetadata(func(c *dix.Container) error {
				_, err := dix.ResolveAs[*Database](c)
				return err
			}, dix.InvokeMetadata{
				Label:        "RawInvokeDatabase",
				Dependencies: []dix.ServiceRef{dix.TypedService[*Database]()},
			}),
		),
		dix.WithModuleHooks(
			dix.RawHookWithMetadata(func(c *dix.Container, lc dix.Lifecycle) {
				lc.OnStart(func(context.Context) error {
					_, err := dix.ResolveAs[*Database](c)
					return err
				})
			}, dix.HookMetadata{
				Label:        "RawStartDatabase",
				Kind:         dix.HookKindStart,
				Dependencies: []dix.ServiceRef{dix.TypedService[*Database]()},
			}),
		),
		dix.WithModuleSetups(
			dixadvanced.DoSetupWithMetadata(func(raw do.Injector) error {
				_ = raw
				return nil
			}, dix.SetupMetadata{
				Label:         "RawDoSetup",
				Dependencies:  []dix.ServiceRef{dix.TypedService[Config]()},
				Provides:      []dix.ServiceRef{dix.NamedService("tenant.default")},
				GraphMutation: true,
			}),
		),
	)

	report := dix.NewApp("warnings", module).ValidateReport()
	require.False(t, report.HasErrors())
	assert.False(t, report.HasWarnings(), report.WarningSummary())
}

func TestBuildFailureShutsDownResolvedServices(t *testing.T) {
	svc := &cleanupService{}
	app := dix.NewApp("cleanup",
		dix.NewModule("cleanup",
			dix.WithModuleProviders(
				dix.Provider0(func() *cleanupService { return svc }),
			),
			dix.WithModuleSetup(func(c *dix.Container, _ dix.Lifecycle) error {
				_, err := dix.ResolveAs[*cleanupService](c)
				require.NoError(t, err)
				return errors.New("setup failed")
			}),
		),
	)

	_, err := app.Build()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "setup failed")
	assert.Equal(t, 1, svc.shutdowns)
}

func TestRuntimeStartFailureRollsBackStopHooks(t *testing.T) {
	type lifecycleService struct {
		starts int
		stops  int
	}

	svc := &lifecycleService{}
	app := dix.NewApp("rollback",
		dix.NewModule("rollback",
			dix.WithModuleProviders(
				dix.Provider0(func() *lifecycleService { return svc }),
			),
			dix.WithModuleHooks(
				dix.OnStart(func(context.Context, *lifecycleService) error {
					svc.starts++
					return nil
				}),
				dix.OnStop(func(context.Context, *lifecycleService) error {
					svc.stops++
					return nil
				}),
				dix.OnStart0(func(context.Context) error {
					return errors.New("boom")
				}),
			),
		),
	)

	rt := buildRuntime(t, app)
	err := rt.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
	assert.Equal(t, 1, svc.starts)
	assert.Equal(t, 1, svc.stops)
	assert.Equal(t, dix.AppStateStopped, rt.State())
}
