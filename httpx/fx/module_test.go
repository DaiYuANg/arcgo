package fx

import (
	"context"
	"testing"

	"github.com/DaiYuANg/archgo/httpx"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestNewHttpxModule(t *testing.T) {
	t.Parallel()

	var server httpx.ServerRuntime

	app := fx.New(
		NewHttpxModule(httpx.WithBasePath("/api")),
		fx.Invoke(func(s httpx.ServerRuntime) {
			httpx.MustGet(s, "/ping", func(ctx context.Context, input *struct{}) (*struct{}, error) {
				_ = ctx
				_ = input
				return &struct{}{}, nil
			})
		}),
		fx.Populate(&server),
	)

	startCtx, startCancel := context.WithCancel(context.Background())
	defer startCancel()
	require.NoError(t, app.Start(startCtx))
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
		NewHttpxModule(),
		WithServerOptions(httpx.WithBasePath("/v1")),
		fx.Invoke(func(s httpx.ServerRuntime) {
			httpx.MustGet(s, "/pong", func(ctx context.Context, input *struct{}) (*struct{}, error) {
				_ = ctx
				_ = input
				return &struct{}{}, nil
			})
		}),
		fx.Populate(&server),
	)

	startCtx, startCancel := context.WithCancel(context.Background())
	defer startCancel()
	require.NoError(t, app.Start(startCtx))
	t.Cleanup(func() {
		require.NoError(t, app.Stop(context.Background()))
	})

	require.NotNil(t, server)
	require.True(t, server.HasRoute(httpx.MethodGet, "/v1/pong"))
}
