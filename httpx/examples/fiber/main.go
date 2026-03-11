package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/DaiYuANg/arcgo/httpx/adapter/fiber"
	"github.com/DaiYuANg/arcgo/httpx/examples/shared"
	"github.com/DaiYuANg/arcgo/pkg/randomport"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	fiberrecover "github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	logger, closeLogger, err := shared.NewLogger()
	if err != nil {
		panic(err)
	}
	defer closeLogger()

	userService := shared.NewMockUserService()
	fiberAdapter := fiber.NewWithOptions(nil, fiber.Options{
		Huma: adapter.HumaOptions{
			Title:       "ArcGo Fiber API",
			Version:     "1.0.0",
			Description: "Typed Fiber API example",
			DocsPath:    "/docs",
			OpenAPIPath: "/openapi.json",
		},
		Logger: logger,
		App: fiber.AppOptions{
			ReadTimeout:     15 * time.Second,
			WriteTimeout:    15 * time.Second,
			IdleTimeout:     60 * time.Second,
			ShutdownTimeout: 5 * time.Second,
		},
	})
	fiberAdapter.Router().Use(fiberrecover.New(), fiberlogger.New())

	server := shared.NewRuntime(fiberAdapter, logger)
	shared.RegisterUserRoutes(server, userService)

	port := randomport.MustFind()
	addr := fmt.Sprintf(":%d", port)
	logger.Info("example server starting",
		slog.String("example", "fiber"),
		slog.String("address", addr),
		slog.String("openapi", fmt.Sprintf("http://localhost%s/openapi.json", addr)),
		slog.String("docs", fmt.Sprintf("http://localhost%s/docs", addr)),
	)

	if err := server.ListenAndServe(addr); err != nil {
		logger.Error("server exited with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
