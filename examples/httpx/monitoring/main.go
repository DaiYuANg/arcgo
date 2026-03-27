// Package main demonstrates httpx monitoring middleware and metrics exposure.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/DaiYuANg/arcgo/examples/httpx/shared"
	"github.com/DaiYuANg/arcgo/httpx"
	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/DaiYuANg/arcgo/httpx/adapter/std"
	"github.com/DaiYuANg/arcgo/httpx/middleware"
	"github.com/DaiYuANg/arcgo/pkg/randomport"
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

type healthOutput struct {
	Body struct {
		Status string `json:"status"`
	}
}

func main() {
	logger, closeLogger, err := shared.NewLogger()
	if err != nil {
		panic(err)
	}
	slogLogger := logger

	router := chi.NewMux()
	router.Use(middleware.PrometheusMiddleware, middleware.OpenTelemetryMiddleware)

	stdAdapter := std.New(router, adapter.HumaOptions{
		Title:       "ArcGo Monitoring API",
		Version:     "1.0.0",
		Description: "Monitoring API",
		DocsPath:    "/docs",
		OpenAPIPath: "/openapi.json",
	})

	server := httpx.New(
		httpx.WithAdapter(stdAdapter),
		httpx.WithLogger(slogLogger),
		httpx.WithPrintRoutes(true),
	)

	stdAdapter.Router().Handle("/metrics", middleware.MetricsHandler())

	httpx.MustGet(server, "/health", func(_ context.Context, _ *struct{}) (*healthOutput, error) {
		out := &healthOutput{}
		out.Body.Status = "ok"
		return out, nil
	}, huma.OperationTags("monitoring"))

	port := randomport.MustFind()
	addr := fmt.Sprintf(":%d", port)
	slogLogger.Info("example server starting",
		slog.String("example", "monitoring"),
		slog.String("address", addr),
		slog.String("health", fmt.Sprintf("http://localhost%s/health", addr)),
		slog.String("metrics", fmt.Sprintf("http://localhost%s/metrics", addr)),
		slog.String("openapi", fmt.Sprintf("http://localhost%s/openapi.json", addr)),
		slog.String("docs", fmt.Sprintf("http://localhost%s/docs", addr)),
	)

	if err := server.ListenPort(port); err != nil {
		slogLogger.Error("server exited with error", slog.String("error", err.Error()))
		closeLogger()
		os.Exit(1)
	}
	closeLogger()
}
