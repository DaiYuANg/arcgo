package httpxdix_test

import (
	"context"
	"testing"
	"time"

	di "github.com/DaiYuANg/arcgo/dix"
	"github.com/DaiYuANg/arcgo/httpx"
	httpxdix "github.com/DaiYuANg/arcgo/httpx/dix"
	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/require"
)

type testConfig struct {
	Port int
}

type fakeAdapter struct {
	listenCh   chan string
	shutdownCh chan struct{}
}

func newFakeAdapter() *fakeAdapter {
	return &fakeAdapter{
		listenCh:   make(chan string, 1),
		shutdownCh: make(chan struct{}, 1),
	}
}

func (f *fakeAdapter) Name() string { return "fake" }

func (f *fakeAdapter) HumaAPI() huma.API { return nil }

func (f *fakeAdapter) Listen(addr string) error {
	select {
	case f.listenCh <- addr:
	default:
	}
	return nil
}

func (f *fakeAdapter) Shutdown() error {
	select {
	case f.shutdownCh <- struct{}{}:
	default:
	}
	return nil
}

func TestNewModule_StartsServerFromResolvedPortAndShutsDownByDefault(t *testing.T) {
	t.Helper()

	adapter := newFakeAdapter()
	server := httpx.New(httpx.WithAdapter(adapter))

	configModule := di.NewModule("config",
		di.Providers(di.Value(testConfig{Port: 8123})),
	)

	httpModule := httpxdix.NewModule(
		"http",
		di.Value[httpx.ServerRuntime](server),
		httpxdix.WithImports(configModule),
		httpxdix.WithListenPort1(func(cfg testConfig) int { return cfg.Port }),
	)

	app := di.New("httpx-dix-test",
		di.Modules(httpModule),
	)

	rt, err := app.Start(context.Background())
	require.NoError(t, err)

	select {
	case addr := <-adapter.listenCh:
		require.Equal(t, ":8123", addr)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for background listen")
	}

	require.NoError(t, rt.Stop(context.Background()))

	select {
	case <-adapter.shutdownCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for shutdown")
	}
}

func TestListenPort1_WithNilResolverFailsStart(t *testing.T) {
	var resolver func(testConfig) int

	module := httpxdix.NewModule(
		"http",
		di.Value[httpx.ServerRuntime](httpx.New(httpx.WithAdapter(newFakeAdapter()))),
		httpxdix.WithListenPort1(resolver),
	)

	app := di.New("httpx-dix-test",
		di.Modules(
			di.NewModule("config", di.Providers(di.Value(testConfig{Port: 8123}))),
			module,
		),
	)

	_, err := app.Start(context.Background())
	require.Error(t, err)
	require.ErrorContains(t, err, "listen port resolver is nil")
}
