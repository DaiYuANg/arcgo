package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/DaiYuANg/toolkit4go/httpx"
	"github.com/DaiYuANg/toolkit4go/httpx/adapter/std"
	"github.com/DaiYuANg/toolkit4go/httpx/options"
	"github.com/DaiYuANg/toolkit4go/logx"
	"github.com/go-chi/chi/v5/middleware"
)

// UserEndpoint 用户端点
type UserEndpoint struct {
	httpx.BaseEndpoint
}

// ListUsers 获取用户列表
func (e *UserEndpoint) ListUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	e.Success(w, map[string]interface{}{
		"users": []string{"Alice", "Bob", "Charlie"},
	})
	return nil
}

// GetUser 获取单个用户
func (e *UserEndpoint) GetUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	id := e.GetQuery(r, "id", "1")
	e.Success(w, map[string]interface{}{
		"user": map[string]string{"id": id, "name": "User" + id},
	})
	return nil
}

func main() {
	// 创建 logger
	logger, _ := logx.New(logx.WithConsole(true))
	defer logger.Close()
	slogLogger := logx.NewSlog(logger)

	// 示例 1: 使用 ServerOptions 配置
	fmt.Println("=== Example 1: Using ServerOptions ===")
	serverOpts := options.DefaultServerOptions()
	serverOpts.Logger = slogLogger
	serverOpts.BasePath = "/api"
	serverOpts.PrintRoutes = true
	serverOpts.HumaEnabled = true
	serverOpts.HumaTitle = "My API"
	serverOpts.HumaVersion = "1.0.0"
	serverOpts.HumaDescription = "API Documentation"
	serverOpts.ReadTimeout = 15 * time.Second
	serverOpts.WriteTimeout = 15 * time.Second
	serverOpts.IdleTimeout = 60 * time.Second

	// 创建适配器
	stdAdapter := std.New().WithLogger(slogLogger)

	// 【重要】使用 chi 原生方式注册中间件
	stdAdapter.Router().Use(
		middleware.Logger,
		middleware.Recoverer,
		middleware.RequestID,
	)

	server1 := httpx.NewServer(append(serverOpts.Build(), httpx.WithAdapter(stdAdapter))...)
	_ = server1.Register(&UserEndpoint{})

	// 示例 2: 使用 HTTP Client Options
	fmt.Println("\n=== Example 2: Using HTTP Client Options ===")
	clientOpts := &options.HTTPClientOptions{
		Timeout: 30 * time.Second,
	}
	client := clientOpts.Build()
	fmt.Printf("HTTP Client Timeout: %v\n", client.Timeout)

	// 示例 3: 使用 Context Options
	fmt.Println("\n=== Example 3: Using Context Options ===")
	ctxOpts := &options.ContextOptions{
		Timeout:       5 * time.Second,
		CancelOnPanic: true,
	}
	ctxOpts = options.WithContextValueOpt(ctxOpts, "request_id", "12345")
	ctx, cancel := ctxOpts.Build()
	defer cancel()
	fmt.Printf("Context created with timeout: 5s\n")
	fmt.Printf("Context value request_id: %v\n", ctx.Value("request_id"))

	fmt.Println("\n=== All Examples Complete ===")
	fmt.Println("Note: Servers are not started in this example.")
	fmt.Println("Use server.ListenAndServe() to start the server.")
}
