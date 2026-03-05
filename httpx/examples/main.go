package main

import (
	"context"
	"fmt"

	"github.com/DaiYuANg/arcgo/httpx"
	"github.com/DaiYuANg/arcgo/httpx/adapter/std"
	"github.com/DaiYuANg/arcgo/logx"
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5/middleware"
)

type ListUsersOutput struct {
	Body struct {
		Users []string `json:"users"`
	}
}

type GetUserInput struct {
	ID string `query:"id"`
}

type GetUserOutput struct {
	Body struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
}

func main() {
	logger, err := logx.New(logx.WithConsole(true), logx.WithDebugLevel())
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	slogLogger := logx.NewSlog(logger)
	stdAdapter := std.New()
	stdAdapter.Router().Use(middleware.Logger, middleware.Recoverer, middleware.RequestID)

	server := httpx.NewServer(
		httpx.WithAdapter(stdAdapter),
		httpx.WithLogger(slogLogger),
		httpx.WithPrintRoutes(true),
		httpx.WithHuma(httpx.HumaOptions{
			Enabled:     true,
			Title:       "ArcGo API",
			Version:     "1.0.0",
			Description: "Typed API built with httpx",
		}),
	)

	if err = httpx.Get(server, "/users", func(ctx context.Context, input *struct{}) (*ListUsersOutput, error) {
		out := &ListUsersOutput{}
		out.Body.Users = []string{"Alice", "Bob", "Charlie"}
		return out, nil
	}, huma.OperationTags("users")); err != nil {
		panic(err)
	}

	api := server.Group("/api/v1")
	if err = httpx.GroupGet(api, "/user", func(ctx context.Context, input *GetUserInput) (*GetUserOutput, error) {
		id := input.ID
		if id == "" {
			id = "1"
		}
		out := &GetUserOutput{}
		out.Body.ID = id
		out.Body.Name = "User" + id
		return out, nil
	}, huma.OperationTags("users")); err != nil {
		panic(err)
	}

	fmt.Println("Server starting on :8080")
	fmt.Println("OpenAPI JSON: http://localhost:8080/openapi.json")
	fmt.Println("Swagger UI:   http://localhost:8080/docs")

	if err = server.ListenAndServe(":8080"); err != nil {
		panic(err)
	}
}
