package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/DaiYuANg/arcgo/examples/httpx/shared"
	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/DaiYuANg/arcgo/httpx/adapter/std"
	"github.com/DaiYuANg/arcgo/pkg/randomport"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	logger, closeLogger, err := shared.NewLogger()
	if err != nil {
		panic(err)
	}
	defer closeLogger()

	userService := shared.NewMockUserService()
	stdAdapter := std.NewWithOptions(std.Options{
		Huma: adapter.HumaOptions{
			Title:       "ArcGo Std API",
			Version:     "1.0.0",
			Description: "Typed std API example",
			DocsPath:    "/docs",
			OpenAPIPath: "/openapi.json",
		},
		Logger: logger,
		Server: std.ServerOptions{
			ReadTimeout:     15 * time.Second,
			WriteTimeout:    15 * time.Second,
			IdleTimeout:     60 * time.Second,
			ShutdownTimeout: 5 * time.Second,
			MaxHeaderBytes:  1 << 20,
		},
	})
	stdAdapter.Router().Use(middleware.Logger, middleware.Recoverer, middleware.RequestID)

	server := shared.NewRuntime(stdAdapter, logger)
	shared.RegisterUserRoutes(server, userService)

	port := randomport.MustFind()
	addr := fmt.Sprintf(":%d", port)
	logger.Info("example server starting",
		slog.String("example", "std"),
		slog.String("address", addr),
		slog.String("openapi", fmt.Sprintf("http://localhost%s/openapi.json", addr)),
		slog.String("docs", fmt.Sprintf("http://localhost%s/docs", addr)),
	)

	if err := server.ListenAndServe(addr); err != nil {
		logger.Error("server exited with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
