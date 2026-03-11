package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/DaiYuANg/arcgo/httpx/adapter/echo"
	"github.com/DaiYuANg/arcgo/httpx/examples/shared"
	"github.com/DaiYuANg/arcgo/pkg/randomport"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
)

func main() {
	logger, closeLogger, err := shared.NewLogger()
	if err != nil {
		panic(err)
	}
	defer closeLogger()

	userService := shared.NewMockUserService()
	echoAdapter := echo.NewWithOptions(nil, echo.Options{
		Huma: adapter.HumaOptions{
			Title:       "ArcGo Echo API",
			Version:     "1.0.0",
			Description: "Typed Echo API example",
			DocsPath:    "/docs",
			OpenAPIPath: "/openapi.json",
		},
		Logger: logger,
		Server: echo.ServerOptions{
			ReadTimeout:     15 * time.Second,
			WriteTimeout:    15 * time.Second,
			IdleTimeout:     60 * time.Second,
			ShutdownTimeout: 5 * time.Second,
			MaxHeaderBytes:  1 << 20,
		},
	})
	echoAdapter.Router().Use(echoMiddleware.Recover(), echoMiddleware.RequestLogger())

	server := shared.NewRuntime(echoAdapter, logger)
	shared.RegisterUserRoutes(server, userService)

	port := randomport.MustFind()
	addr := fmt.Sprintf(":%d", port)
	logger.Info("example server starting",
		slog.String("example", "echo"),
		slog.String("address", addr),
		slog.String("openapi", fmt.Sprintf("http://localhost%s/openapi.json", addr)),
		slog.String("docs", fmt.Sprintf("http://localhost%s/docs", addr)),
	)

	if err := server.ListenAndServe(addr); err != nil {
		logger.Error("server exited with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
