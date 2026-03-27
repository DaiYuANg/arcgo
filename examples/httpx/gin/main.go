// Package main demonstrates the httpx gin adapter example.
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/DaiYuANg/arcgo/examples/httpx/shared"
	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/DaiYuANg/arcgo/httpx/adapter/gin"
	"github.com/DaiYuANg/arcgo/pkg/randomport"
	ginFramework "github.com/gin-gonic/gin"
)

func main() {
	logger, closeLogger, err := shared.NewLogger()
	if err != nil {
		panic(err)
	}

	userService := shared.NewMockUserService()
	ginAdapter := gin.New(nil, adapter.HumaOptions{
		Title:       "ArcGo Gin API",
		Version:     "1.0.0",
		Description: "Typed Gin API example",
		DocsPath:    "/docs",
		OpenAPIPath: "/openapi.json",
	})
	ginAdapter.Router().Use(ginFramework.Logger(), ginFramework.Recovery())

	server := shared.NewRuntime(ginAdapter, logger)
	shared.RegisterUserRoutes(server, userService)

	port := randomport.MustFind()
	addr := fmt.Sprintf(":%d", port)
	logger.Info("example server starting",
		slog.String("example", "gin"),
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
