package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/DaiYuANg/archgo/httpx/adapter"
	"github.com/DaiYuANg/archgo/httpx/adapter/gin"
	"github.com/DaiYuANg/archgo/httpx/examples/shared"
	"github.com/DaiYuANg/archgo/pkg/randomport"
	ginFramework "github.com/gin-gonic/gin"
)

func main() {
	logger, closeLogger, err := shared.NewLogger()
	if err != nil {
		panic(err)
	}
	defer closeLogger()

	userService := shared.NewMockUserService()
	ginAdapter := gin.NewWithOptions(nil, gin.Options{
		Huma: adapter.HumaOptions{
			Title:       "ArcGo Gin API",
			Version:     "1.0.0",
			Description: "Typed Gin API example",
			DocsPath:    "/docs",
			OpenAPIPath: "/openapi.json",
		},
		Logger: logger,
		Server: gin.ServerOptions{
			ReadTimeout:     15 * time.Second,
			WriteTimeout:    15 * time.Second,
			IdleTimeout:     60 * time.Second,
			ShutdownTimeout: 5 * time.Second,
			MaxHeaderBytes:  1 << 20,
		},
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

	if err := server.ListenAndServe(addr); err != nil {
		logger.Error("server exited with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
