package main

import (
	"context"
	"fmt"
	"time"

	"github.com/DaiYuANg/arcgo/httpx"
	"github.com/DaiYuANg/arcgo/httpx/adapter/std"
	"github.com/DaiYuANg/arcgo/httpx/options"
	"github.com/DaiYuANg/arcgo/logx"
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
	defer logger.Close()
	slogLogger := logx.NewSlog(logger)

	fmt.Println("=== Example 1: Using ServerOptions ===")
	serverOpts := options.DefaultServerOptions()
	serverOpts.Logger = slogLogger
	serverOpts.BasePath = "/api"
	serverOpts.PrintRoutes = true
	serverOpts.HumaEnabled = true
	serverOpts.HumaTitle = "ArcGo API"
	serverOpts.HumaVersion = "1.0.0"
	serverOpts.HumaDescription = "API Documentation"
	serverOpts.ReadTimeout = 15 * time.Second
	serverOpts.WriteTimeout = 15 * time.Second
	serverOpts.IdleTimeout = 60 * time.Second

	stdAdapter := std.New().WithLogger(slogLogger)
	stdAdapter.Router().Use(middleware.Logger, middleware.Recoverer, middleware.RequestID)

	server := httpx.NewServer(append(serverOpts.Build(), httpx.WithAdapter(stdAdapter))...)
	err := httpx.Get(server, "/users", func(ctx context.Context, input *struct{}) (*UserOutput, error) {
		out := &UserOutput{}
		out.Body.Users = []string{"Alice", "Bob", "Charlie"}
		return out, nil
	}, huma.OperationTags("users"))
	if err != nil {
		panic(err)
	}

	fmt.Println("=== Example 2: Using HTTP Client Options ===")
	clientOpts := &options.HTTPClientOptions{Timeout: 30 * time.Second}
	client := clientOpts.Build()
	fmt.Printf("HTTP Client Timeout: %v\n", client.Timeout)

	fmt.Println("=== Example 3: Using Context Options ===")
	ctxOpts := &options.ContextOptions{Timeout: 5 * time.Second, CancelOnPanic: true}
	ctxOpts = options.WithContextValueOpt(ctxOpts, "request_id", "12345")
	ctx, cancel := ctxOpts.Build()
	defer cancel()
	fmt.Printf("Context value request_id: %v\n", ctx.Value("request_id"))

	fmt.Println("All examples complete. Use server.ListenAndServe() to start server.")
}
