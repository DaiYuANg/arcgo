// Package main demonstrates a minimal httpx server using the std adapter.
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
	"github.com/DaiYuANg/arcgo/pkg/randomport"
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
)

type listUsersOutput struct {
	Body struct {
		Users []string `json:"users"`
	}
}

type getUserInput struct {
	ID string `query:"id" validate:"omitempty,min=1,max=32"`
}

type getUserOutput struct {
	Body struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
}

func main() {
	logger, closeLogger, err := shared.NewLogger()
	if err != nil {
		panic(err)
	}

	slogLogger := logger
	router := chi.NewMux()
	router.Use(middleware.Logger, middleware.Recoverer, middleware.RequestID)

	stdAdapter := std.New(router, adapter.HumaOptions{
		Title:       "ArcGo API",
		Version:     "1.0.0",
		Description: "Typed API built with httpx",
		DocsPath:    "/docs",
		OpenAPIPath: "/openapi.json",
	})

	server := httpx.New(
		httpx.WithAdapter(stdAdapter),
		httpx.WithLogger(slogLogger),
		httpx.WithPrintRoutes(true),
		httpx.WithValidator(validator.New(validator.WithRequiredStructEnabled())),
	)

	httpx.MustGet(server, "/users", func(_ context.Context, _ *struct{}) (*listUsersOutput, error) {
		out := &listUsersOutput{}
		out.Body.Users = []string{"Alice", "Bob", "Charlie"}
		return out, nil
	}, huma.OperationTags("users"))

	api := server.Group("/api/v1")
	httpx.MustGroupGet(api, "/user", func(_ context.Context, input *getUserInput) (*getUserOutput, error) {
		id := input.ID
		if id == "" {
			id = "1"
		}
		out := &getUserOutput{}
		out.Body.ID = id
		out.Body.Name = "User" + id
		return out, nil
	}, huma.OperationTags("users"))

	port := randomport.MustFind()
	addr := fmt.Sprintf(":%d", port)
	slogLogger.Info("example server starting",
		slog.String("example", "main"),
		slog.String("address", addr),
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
