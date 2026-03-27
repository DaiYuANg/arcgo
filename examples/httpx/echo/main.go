// Package main demonstrates the httpx echo adapter example.
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/DaiYuANg/arcgo/examples/httpx/shared"
	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/DaiYuANg/arcgo/httpx/adapter/echo"
	"github.com/DaiYuANg/arcgo/pkg/randomport"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
)

func main() {
	logger, closeLogger, err := shared.NewLogger()
	if err != nil {
		panic(err)
	}

	userService := shared.NewMockUserService()
	echoAdapter := echo.New(nil, adapter.HumaOptions{
		Title:       "ArcGo Echo API",
		Version:     "1.0.0",
		Description: "Typed Echo API example",
		DocsPath:    "/docs",
		OpenAPIPath: "/openapi.json",
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

	if err := server.ListenPort(port); err != nil {
		logger.Error("server exited with error", slog.String("error", err.Error()))
		closeLogger()
		os.Exit(1)
	}
	closeLogger()
}
