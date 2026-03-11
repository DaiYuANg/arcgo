package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/DaiYuANg/arcgo/httpx"
	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/DaiYuANg/arcgo/httpx/adapter/std"
	"github.com/DaiYuANg/arcgo/httpx/middleware"
	"github.com/DaiYuANg/arcgo/logx"
	"github.com/DaiYuANg/arcgo/pkg/randomport"
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
	defer func() { _ = logger.Close() }()
	slogLogger := logx.NewSlog(logger)

	stdAdapter := std.New(adapter.HumaOptions{
		Title:       "ArcGo Monitoring API",
		Version:     "1.0.0",
		Description: "Monitoring API",
		DocsPath:    "/docs",
		OpenAPIPath: "/openapi.json",
	}).WithLogger(slogLogger)

	server := httpx.New(
		httpx.WithAdapter(stdAdapter),
		httpx.WithLogger(slogLogger),
		httpx.WithPrintRoutes(true),
	)

	httpx.MustGet(server, "/health", func(ctx context.Context, input *struct{}) (*HealthOutput, error) {
		out := &HealthOutput{}
		out.Body.Status = "ok"
		return out, nil
	}, huma.OperationTags("monitoring"))

	server.Adapter().Handle(httpx.MethodGet, "/metrics", func(
		ctx context.Context,
		w http.ResponseWriter,
		r *http.Request,
	) error {
		_ = ctx
		middleware.MetricsHandler().ServeHTTP(w, r)
		return nil
	})

	_ = middleware.PrometheusMiddleware(middleware.OpenTelemetryMiddleware(server.Handler()))

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

	if err := server.ListenAndServe(addr); err != nil {
		slogLogger.Error("server exited with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
