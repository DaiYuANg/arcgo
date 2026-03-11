package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/DaiYuANg/arcgo/httpx"
	"github.com/DaiYuANg/arcgo/httpx/adapter/std"
	"github.com/DaiYuANg/arcgo/httpx/options"
	"github.com/DaiYuANg/arcgo/logx"
	"github.com/DaiYuANg/arcgo/pkg/randomport"
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5/middleware"
)

type UserOutput struct {
	Body struct {
		Users []string `json:"users"`
	}
}

func main() {
	logger, _ := logx.New(logx.WithConsole(true))
	defer func() { _ = logger.Close() }()
	slogLogger := logx.NewSlog(logger)

	slogLogger.Info("config example section", slog.String("section", "server options + adapter options"))
	serverOpts := options.DefaultServerOptions()
	serverOpts.Logger = slogLogger
	serverOpts.BasePath = "/api"
	serverOpts.PrintRoutes = true
	serverOpts.EnableValidation = true
	serverOpts.OpenAPIDocsEnabled = true
	serverOpts.HumaTitle = "ArcGo API"
	serverOpts.HumaVersion = "1.0.0"
	serverOpts.HumaDescription = "API Documentation"
	serverOpts.DocsPath = "/docs"
	serverOpts.OpenAPIPath = "/openapi.json"
	serverOpts.EnablePanicRecover = true
	serverOpts.EnableAccessLog = true

	// httpx server logs and adapter bridge logs are configured separately.
	stdAdapter := std.NewWithOptions(std.Options{
		Logger: slogLogger,
		Server: std.ServerOptions{
			ReadTimeout:     15 * time.Second,
			WriteTimeout:    15 * time.Second,
			IdleTimeout:     60 * time.Second,
			ShutdownTimeout: 5 * time.Second,
			MaxHeaderBytes:  1 << 20,
		},
	})
	stdAdapter.Router().Use(middleware.Logger, middleware.Recoverer, middleware.RequestID)

	server := httpx.New(append(serverOpts.Build(), httpx.WithAdapter(stdAdapter))...)
	httpx.MustGet(server, "/users", func(ctx context.Context, input *struct{}) (*UserOutput, error) {
		out := &UserOutput{}
		out.Body.Users = []string{"Alice", "Bob", "Charlie"}
		return out, nil
	}, huma.OperationTags("users"))

	slogLogger.Info("config example section", slog.String("section", "http client options"))
	clientOpts := &options.HTTPClientOptions{Timeout: 30 * time.Second}
	client := clientOpts.Build()
	slogLogger.Info("http client configured", slog.Duration("timeout", client.Timeout))

	slogLogger.Info("config example section", slog.String("section", "context options"))
	ctxOpts := &options.ContextOptions{Timeout: 5 * time.Second}
	ctxOpts = options.WithContextValueOpt(ctxOpts, "request_id", "12345")
	ctx, cancel := ctxOpts.Build()
	defer cancel()
	slogLogger.Info("context configured", slog.Any("request_id", ctx.Value("request_id")))

	port := randomport.MustFind()
	addr := fmt.Sprintf(":%d", port)
	slogLogger.Info("example server starting",
		slog.String("example", "config"),
		slog.String("address", addr),
		slog.String("openapi", fmt.Sprintf("http://localhost%s/openapi.json", addr)),
		slog.String("docs", fmt.Sprintf("http://localhost%s/docs", addr)),
		slog.Any("routes", server.GetRoutes()),
	)

	if err := server.ListenAndServe(addr); err != nil {
		slogLogger.Error("server exited with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
