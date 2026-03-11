package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/DaiYuANg/arcgo/configx"
	"github.com/DaiYuANg/arcgo/httpx"
	"github.com/DaiYuANg/arcgo/httpx/adapter/std"
	"github.com/DaiYuANg/arcgo/observabilityx"
	otelobs "github.com/DaiYuANg/arcgo/observabilityx/otel"
	promobs "github.com/DaiYuANg/arcgo/observabilityx/prometheus"
)

type appConfig struct {
	Name string `mapstructure:"name" validate:"required"`
	Port int    `mapstructure:"port" validate:"required,min=1,max=65535"`
}

func main() {
	prom := promobs.New(promobs.WithNamespace("configx_example"))
	obs := observabilityx.Multi(otelobs.New(), prom)

	cfg := appConfig{}
	if err := configx.Load(&cfg,
		configx.WithObservability(obs),
		configx.WithDefaults(map[string]any{
			"name": "arcgo",
			"port": 8080,
		}),
		configx.WithValidateLevel(configx.ValidateLevelRequired),
	); err != nil {
		panic(err)
	}

	fmt.Printf("loaded config: %+v\n", cfg)

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
	err := metricsServer.ListenAndServe(":8080")
	if err != nil {
		panic(err)
	}
}
