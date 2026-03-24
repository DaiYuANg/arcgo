package main

import (
	"fmt"

	"github.com/DaiYuANg/arcgo/configx"
	"github.com/DaiYuANg/arcgo/httpx"
	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/DaiYuANg/arcgo/httpx/adapter/std"
	"github.com/DaiYuANg/arcgo/observabilityx"
	otelobs "github.com/DaiYuANg/arcgo/observabilityx/otel"
	promobs "github.com/DaiYuANg/arcgo/observabilityx/prometheus"
)

type appConfig struct {
	Name string `validate:"required"`
	Port int    `validate:"required,min=1,max=65535"`
}

func main() {
	prom := promobs.New(promobs.WithNamespace("configx_example"))
	obs := observabilityx.Multi(otelobs.New(), prom)

	cfg, err := configx.LoadTErr[appConfig](
		configx.WithObservability(obs),
		configx.WithDefaults(map[string]any{
			"name": "arcgo",
			"port": 8080,
		}),
		configx.WithValidateLevel(configx.ValidateLevelStruct),
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("loaded config: %+v\n", cfg)

	stdAdapter := std.New(nil, adapter.HumaOptions{DisableDocsRoutes: true})
	metricsServer := httpx.New(
		httpx.WithAdapter(stdAdapter),
	)
	stdAdapter.Router().Handle("/metrics", prom.Handler())

	fmt.Println("httpx metrics route registered: GET /metrics")
	_ = metricsServer
	err = metricsServer.ListenAndServe(":8080")
	if err != nil {
		panic(err)
	}
}
