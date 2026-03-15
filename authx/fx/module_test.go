package fx

import (
	"context"
	"testing"

	"github.com/DaiYuANg/archgo/authx"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestNewAuthxModule(t *testing.T) {
	t.Parallel()

	var engine *authx.Engine

	provider := authx.NewAuthenticationProviderFunc(func(
		ctx context.Context,
		credential string,
	) (authx.AuthenticationResult, error) {
		_ = ctx
		return authx.AuthenticationResult{Principal: credential}, nil
	})

	app := fx.New(
		NewAuthxModule(
			authx.WithAuthenticationManager(authx.NewProviderManager(provider)),
		),
		fx.Populate(&engine),
	)

	startCtx, startCancel := context.WithCancel(context.Background())
	defer startCancel()
	require.NoError(t, app.Start(startCtx))
	t.Cleanup(func() {
		require.NoError(t, app.Stop(context.Background()))
	})

	require.NotNil(t, engine)
	result, err := engine.Check(context.Background(), "alice")
	require.NoError(t, err)
	require.Equal(t, "alice", result.Principal)
}

func TestWithEngineOptions(t *testing.T) {
	t.Parallel()

	var engine *authx.Engine

	provider := authx.NewAuthenticationProviderFunc(func(
		ctx context.Context,
		credential string,
	) (authx.AuthenticationResult, error) {
		_ = ctx
		return authx.AuthenticationResult{Principal: credential}, nil
	})

	app := fx.New(
		NewAuthxModule(),
		WithEngineOptions(authx.WithAuthenticationManager(authx.NewProviderManager(provider))),
		fx.Populate(&engine),
	)

	startCtx, startCancel := context.WithCancel(context.Background())
	defer startCancel()
	require.NoError(t, app.Start(startCtx))
	t.Cleanup(func() {
		require.NoError(t, app.Stop(context.Background()))
	})

	require.NotNil(t, engine)
	result, err := engine.Check(context.Background(), "bob")
	require.NoError(t, err)
	require.Equal(t, "bob", result.Principal)
}
