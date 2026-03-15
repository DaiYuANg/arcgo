package dix_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/DaiYuANg/archgo/dix"
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
	fmt.Println("Database connected:", d.dsn)
	return nil
}

func (d *Database) Close() error {
	fmt.Println("Database closed")
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
	fmt.Println("Server starting on", s.addr)
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	fmt.Println("Server stopped")
	return nil
}

// Example modules

var DatabaseModule = dix.NewModule("database",
	dix.WithProviders(
		dix.Provider0(ProvideConfig),
		dix.Provider1(ProvideDatabase),
	),
	dix.WithSetup(func(c *dix.Container, lc dix.Lifecycle) error {
		dix.OnStartHook[*Database](c, func(db *Database) error {
			return db.Connect()
		})(lc)

		dix.OnStopHook[*Database](c, func(db *Database) error {
			return db.Close()
		})(lc)

		return nil
	}),
)

var ServerModule = dix.NewModule("server",
	dix.WithProviders(
		dix.Provider1(ProvideServer),
	),
	dix.WithImports(DatabaseModule),
	dix.WithSetup(func(c *dix.Container, lc dix.Lifecycle) error {
		dix.OnStartHook[*Server](c, func(s *Server) error {
			return s.Start()
		})(lc)

		dix.OnStopHook[*Server](c, func(s *Server) error {
			return s.Stop(context.Background())
		})(lc)

		return nil
	}),
)

// Tests

func TestApp_Build(t *testing.T) {
	app := dix.NewApp("testapp", ServerModule)

	err := app.Build()
	require.NoError(t, err)

	db, err := dix.ResolveAs[*Database](app.Container())
	require.NoError(t, err)
	assert.NotNil(t, db)
	assert.Equal(t, "sqlite://test.db", db.dsn)

	cfg, err := dix.ResolveAs[Config](app.Container())
	require.NoError(t, err)
	assert.Equal(t, 8080, cfg.Port)
}

func TestApp_StartStop(t *testing.T) {
	app := dix.NewApp("testapp", ServerModule)
	ctx := context.Background()

	err := app.Build()
	require.NoError(t, err)

	err = app.Start(ctx)
	require.NoError(t, err)

	err = app.Stop(ctx)
	require.NoError(t, err)
}

func TestApp_Run(t *testing.T) {
	app := dix.NewApp("testapp", ServerModule)

	err := app.Build()
	require.NoError(t, err)
	assert.Equal(t, dix.AppStateBuilt, app.State())

	ctx := context.Background()
	err = app.Start(ctx)
	require.NoError(t, err)
	assert.Equal(t, dix.AppStateStarted, app.State())

	err = app.Stop(ctx)
	require.NoError(t, err)
	assert.Equal(t, dix.AppStateStopped, app.State())
}

func TestModule_WithProfiles(t *testing.T) {
	devOnlyCalled := false
	prodOnlyCalled := false

	devModule := dix.NewModule("dev-only",
		dix.WithProfiles(dix.ProfileDev),
		dix.WithProviders(
			dix.Provider0(func() string {
				devOnlyCalled = true
				return "dev"
			}),
		),
	)

	prodModule := dix.NewModule("prod-only",
		dix.WithProfiles(dix.ProfileProd),
		dix.WithProviders(
			dix.Provider0(func() string {
				prodOnlyCalled = true
				return "prod"
			}),
		),
	)

	appDev := dix.NewAppWithOptions("test",
		[]dix.AppOption{dix.WithProfile(dix.ProfileDev)},
		devModule,
		prodModule,
	)

	err := appDev.Build()
	require.NoError(t, err)

	devStr, err := dix.ResolveAs[string](appDev.Container())
	require.NoError(t, err)
	assert.Equal(t, "dev", devStr)
	assert.True(t, devOnlyCalled)
	assert.False(t, prodOnlyCalled)

	appProd := dix.NewAppWithOptions("test",
		[]dix.AppOption{dix.WithProfile(dix.ProfileProd)},
		devModule,
		prodModule,
	)

	err = appProd.Build()
	require.NoError(t, err)

	prodStr, err := dix.ResolveAs[string](appProd.Container())
	require.NoError(t, err)
	assert.Equal(t, "prod", prodStr)
}

func TestModule_WithExcludeProfiles(t *testing.T) {
	called := false

	module := dix.NewModule("not-test",
		dix.WithExcludeProfiles(dix.ProfileTest),
		dix.WithProviders(
			dix.Provider0(func() string {
				called = true
				return "value"
			}),
		),
	)

	appTest := dix.NewAppWithOptions("test",
		[]dix.AppOption{dix.WithProfile(dix.ProfileTest)},
		module,
	)

	err := appTest.Build()
	require.NoError(t, err)

	_, err = dix.ResolveAs[string](appTest.Container())
	assert.Error(t, err)
	assert.False(t, called)
}

func TestResolveAs(t *testing.T) {
	app := dix.NewApp("test",
		dix.NewModule("test",
			dix.WithProviders(
				dix.Provider0(ProvideConfig),
			),
		),
	)

	err := app.Build()
	require.NoError(t, err)

	cfg, err := dix.ResolveAs[Config](app.Container())
	require.NoError(t, err)
	assert.Equal(t, 8080, cfg.Port)
}

func TestMustResolveAs(t *testing.T) {
	app := dix.NewApp("test",
		dix.NewModule("test",
			dix.WithProviders(
				dix.Provider0(ProvideConfig),
			),
		),
	)

	err := app.Build()
	require.NoError(t, err)

	cfg := dix.MustResolveAs[Config](app.Container())
	assert.Equal(t, 8080, cfg.Port)
}

func TestInvoke(t *testing.T) {
	invoked := false

	app := dix.NewApp("test",
		dix.NewModule("test",
			dix.WithProviders(
				dix.Provider0(ProvideConfig),
			),
			dix.WithInvokes(
				dix.Invoke1(func(cfg Config) {
					invoked = true
					assert.Equal(t, 8080, cfg.Port)
				}),
			),
		),
	)

	err := app.Build()
	require.NoError(t, err)
	assert.True(t, invoked)
}

func TestModule_Imports(t *testing.T) {
	baseModule := dix.NewModule("base",
		dix.WithProviders(
			dix.Provider0(ProvideConfig),
		),
	)

	dependentModule := dix.NewModule("dependent",
		dix.WithImports(baseModule),
		dix.WithProviders(
			dix.Provider1(ProvideDatabase),
		),
	)

	app := dix.NewApp("test", dependentModule)

	err := app.Build()
	require.NoError(t, err)

	cfg, err := dix.ResolveAs[Config](app.Container())
	require.NoError(t, err)

	db, err := dix.ResolveAs[*Database](app.Container())
	require.NoError(t, err)

	assert.NotNil(t, cfg)
	assert.NotNil(t, db)
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
		dix.WithProviders(
			dix.Provider0(func() string { return "hello" }),
			dix.Provider1(func(s string) int { return len(s) }),
		),
		dix.WithSetup(func(c *dix.Container, lc dix.Lifecycle) error {
			lc.OnStart(func(ctx context.Context) error {
				s, _ := dix.ResolveAs[string](c)
				fmt.Println("Starting with:", s)
				return nil
			})
			return nil
		}),
	)

	app := dix.NewApp("test", module)
	_ = app.Build()
}

func ExampleHook() {
	module := dix.NewModule("example",
		dix.WithProviders(
			dix.Provider0(func() *http.Server {
				return &http.Server{Addr: ":8080"}
			}),
		),
		dix.WithSetup(func(c *dix.Container, lc dix.Lifecycle) error {
			dix.OnStartHook[*http.Server](c, func(s *http.Server) error {
				go s.ListenAndServe()
				return nil
			})(lc)

			dix.OnStopHook[*http.Server](c, func(s *http.Server) error {
				return s.Shutdown(context.Background())
			})(lc)

			return nil
		}),
	)

	app := dix.NewApp("test", module)
	_ = app
}
