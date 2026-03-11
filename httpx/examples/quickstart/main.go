package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/DaiYuANg/arcgo/httpx"
	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/DaiYuANg/arcgo/httpx/adapter/std"
	"github.com/DaiYuANg/arcgo/httpx/examples/shared"
	"github.com/DaiYuANg/arcgo/pkg/randomport"
)

type healthOutput struct {
	Body struct {
		Status string `json:"status"`
	} `json:"body"`
}

type createUserInput struct {
	Body struct {
		Name  string `json:"name" validate:"required,min=2,max=64"`
		Email string `json:"email" validate:"required,email"`
	} `json:"body"`
}

type createUserOutput struct {
	Body struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"body"`
}

type getUserInput struct {
	ID int `path:"id"`
}

type getUserOutput struct {
	Body struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"body"`
}

func main() {
	logger, closeLogger, err := shared.NewLogger()
	if err != nil {
		panic(err)
	}
	defer closeLogger()

	adapter := std.New(adapter.HumaOptions{
		Title:       "httpx quickstart",
		Version:     "1.0.0",
		Description: "Typed HTTP quickstart example",
		DocsPath:    "/docs",
		OpenAPIPath: "/openapi.json",
	})

	server := httpx.New(
		httpx.WithAdapter(adapter),
		httpx.WithBasePath("/api"),
		httpx.WithValidation(),
		httpx.WithPrintRoutes(true),
	)

	httpx.MustGet(server, "/health", func(ctx context.Context, in *struct{}) (*healthOutput, error) {
		out := &healthOutput{}
		out.Body.Status = "ok"
		return out, nil
	})

	v1 := server.Group("/v1")

	httpx.MustGroupPost(v1, "/users", func(ctx context.Context, in *createUserInput) (*createUserOutput, error) {
		out := &createUserOutput{}
		out.Body.ID = 1001
		out.Body.Name = in.Body.Name
		out.Body.Email = in.Body.Email
		return out, nil
	})

	httpx.MustGroupGet(v1, "/users/{id}", func(ctx context.Context, in *getUserInput) (*getUserOutput, error) {
		out := &getUserOutput{}
		out.Body.ID = in.ID
		out.Body.Name = "demo-user"
		return out, nil
	})

	port := randomport.MustFind()
	addr := fmt.Sprintf(":%d", port)
	logger.Info("example server starting",
		slog.String("example", "quickstart"),
		slog.String("address", addr),
		slog.String("openapi", fmt.Sprintf("http://localhost%s/openapi.json", addr)),
		slog.String("docs", fmt.Sprintf("http://localhost%s/docs", addr)),
	)

	if err := server.ListenAndServe(addr); err != nil {
		logger.Error("server exited with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
