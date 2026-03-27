package fx_test

import (
	"context"
	"testing"

	"github.com/DaiYuANg/arcgo/httpx"
	httpxfx "github.com/DaiYuANg/arcgo/httpx/fx"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestNewHttpxModule(t *testing.T) {
	t.Parallel()

	var server httpx.ServerRuntime

	app := fx.New(
		httpxfx.NewHttpxModule(httpx.WithBasePath("/api")),
		fx.Invoke(func(s httpx.ServerRuntime) {
			httpx.MustGet(s, "/ping", func(_ context.Context, _ *struct{}) (*struct{}, error) {
				return &struct{}{}, nil
			})
		}),
		fx.Populate(&server),
	)

	require.NoError(t, app.Start(t.Context()))
	t.Cleanup(func() {
		require.NoError(t, app.Stop(context.Background()))
	})

	require.NotNil(t, server)
	require.True(t, server.HasRoute(httpx.MethodGet, "/api/ping"))
}

func TestWithServerOptions(t *testing.T) {
	t.Parallel()

	var server httpx.ServerRuntime

	app := fx.New(
		httpxfx.NewHttpxModule(),
		httpxfx.WithServerOptions(httpx.WithBasePath("/v1")),
		fx.Invoke(func(s httpx.ServerRuntime) {
			httpx.MustGet(s, "/pong", func(_ context.Context, _ *struct{}) (*struct{}, error) {
				return &struct{}{}, nil
			})
		}),
		fx.Populate(&server),
	)

	require.NoError(t, app.Start(t.Context()))
	t.Cleanup(func() {
		require.NoError(t, app.Stop(context.Background()))
	})

	require.NotNil(t, server)
	require.True(t, server.HasRoute(httpx.MethodGet, "/v1/pong"))
}
