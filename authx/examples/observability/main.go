package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/DaiYuANg/arcgo/authx"
	"github.com/DaiYuANg/arcgo/httpx"
	"github.com/DaiYuANg/arcgo/httpx/adapter/std"
	"github.com/DaiYuANg/arcgo/logx"
	"github.com/DaiYuANg/arcgo/observabilityx"
	otelobs "github.com/DaiYuANg/arcgo/observabilityx/otel"
	promobs "github.com/DaiYuANg/arcgo/observabilityx/prometheus"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}

	provider := authx.NewInMemoryIdentityProvider()
	if err := provider.UpsertUser(authx.UserDetails{
		ID:           "u-1",
		Principal:    "alice",
		PasswordHash: string(hashedPassword),
		Name:         "Alice",
	}); err != nil {
		panic(err)
	}

	source := authx.NewMemoryPolicySource(authx.MemoryPolicySourceConfig{
		Name: "observability-policy-source",
		InitialPermissions: []authx.PermissionRule{
			authx.AllowPermission("u-1", "order:1001", "read"),
		},
	})

	logger, err := logx.New(logx.WithConsole(true), logx.WithDebugLevel())
	if err != nil {
		panic(err)
	}
	defer func() { _ = logger.Close() }()

	prom := promobs.New(promobs.WithNamespace("authx_example"))
	obs := observabilityx.Multi(otelobs.New(), prom)

	manager, err := authx.NewManager(
		authx.WithLogger(logx.NewSlog(logger)),
		authx.WithObservability(obs),
		authx.WithProvider(provider),
		authx.WithSource(source),
	)
	if err != nil {
		panic(err)
	}

	if _, err := manager.LoadPolicies(context.Background()); err != nil {
		panic(err)
	}

	ctx, _, err := manager.AuthenticatePassword(context.Background(), "alice", "secret")
	if err != nil {
		panic(err)
	}

	allowed, err := manager.Can(ctx, "read", "order:1001")
	if err != nil {
		panic(err)
	}
	fmt.Println("allowed:", allowed)

	metricsServer := httpx.New(
		httpx.WithAdapter(std.New()),
		httpx.WithOpenAPIDocs(false),
	)
	metricsServer.Adapter().Handle(httpx.MethodGet, "/metrics", func(
		ctx context.Context,
		w http.ResponseWriter,
		r *http.Request,
	) error {
		_ = ctx
		prom.Handler().ServeHTTP(w, r)
		return nil
	})

	fmt.Println("httpx metrics route registered: GET /metrics")
	_ = metricsServer
}
