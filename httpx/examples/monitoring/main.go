package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/DaiYuANg/arcgo/httpx"
	"github.com/DaiYuANg/arcgo/httpx/middleware"
	"github.com/DaiYuANg/arcgo/logx"
	"github.com/danielgtaylor/huma/v2"
)

type HealthOutput struct {
	Body struct {
		Status string `json:"status"`
	}
}

func main() {
	logger, err := logx.New(logx.WithConsole(true), logx.WithDebugLevel())
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	server := httpx.NewServer(
		httpx.WithLogger(logx.NewSlog(logger)),
		httpx.WithPrintRoutes(true),
		httpx.WithHuma(httpx.HumaOptions{Enabled: true, Title: "ArcGo Monitoring API", Version: "1.0.0"}),
	)

	err = httpx.Get(server, "/health", func(ctx context.Context, input *struct{}) (*HealthOutput, error) {
		out := &HealthOutput{}
		out.Body.Status = "ok"
		return out, nil
	}, huma.OperationTags("monitoring"))
	if err != nil {
		panic(err)
	}

	monitored := middleware.PrometheusMiddleware(middleware.OpenTelemetryMiddleware(server.Handler()))

	mux := http.NewServeMux()
	mux.Handle("/", monitored)
	mux.Handle("/metrics", middleware.MetricsHandler())

	fmt.Println("Monitoring server starting on :8080")
	if err = http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
}
