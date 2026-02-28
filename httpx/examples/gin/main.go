package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/DaiYuANg/toolkit4go/httpx"
	"github.com/DaiYuANg/toolkit4go/httpx/adapter/gin"
	"github.com/DaiYuANg/toolkit4go/logx"
	ginFramework "github.com/gin-gonic/gin"
)

// UserEndpoint 用户相关端点
type UserEndpoint struct {
	httpx.BaseEndpoint
}

// ListUsers 获取用户列表
// 自动生成路由：GET /users
func (e *UserEndpoint) ListUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	e.Success(w, map[string]interface{}{
		"users": []string{"Alice", "Bob", "Charlie"},
	})
	return nil
}

// GetUser 获取单个用户
// 自动生成路由：GET /user
func (e *UserEndpoint) GetUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	id := e.GetQuery(r, "id", "1")
	e.Success(w, map[string]interface{}{
		"user": map[string]string{
			"id":   id,
			"name": "User" + id,
		},
	})
	return nil
}

// CreateNewUser 创建用户
// 自动生成路由：POST /new/user
func (e *UserEndpoint) CreateNewUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	e.Success(w, map[string]interface{}{
		"message": "user created successfully",
	})
	return nil
}

// UpdateUserInfo 更新用户信息
// 自动生成路由：PUT /user/info
func (e *UserEndpoint) UpdateUserInfo(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	e.Success(w, map[string]interface{}{
		"message": "user updated successfully",
	})
	return nil
}

// DeleteUserAccount 删除用户账户
// 自动生成路由：DELETE /user/account
func (e *UserEndpoint) DeleteUserAccount(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	e.Success(w, map[string]interface{}{
		"message": "user deleted successfully",
	})
	return nil
}

func main() {
	// 创建 logx logger
	logger, err := logx.New(
		logx.WithConsole(true),
		logx.WithDebugLevel(),
	)
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	// 创建 slog logger
	slogLogger := logx.NewSlog(logger)

	// 创建端点实例
	userEndpoint := &UserEndpoint{}

	// 创建 Gin 适配器
	// 核心思路：httpx 负责路由注册，中间件使用 Gin 原生方式
	ginAdapter := gin.New()

	// 【重要】使用 Gin 原生方式注册中间件
	// 你可以使用任何 Gin 生态的中间件
	ginAdapter.Engine().Use(
		ginFramework.Logger(),   // Gin 原生日志中间件
		ginFramework.Recovery(), // Gin 原生恢复中间件
		yourCustomMiddleware(),  // 你的自定义中间件
	)

	// 启用 Huma OpenAPI 文档（可选）
	ginAdapter.WithHuma(httpx.ToAdapterHumaOptions(httpx.HumaOptions{
		Enabled:     true,
		Title:       "My API",
		Version:     "1.0.0",
		Description: "API built with httpx (Gin adapter)",
	}))

	// 创建服务器
	server := httpx.NewServer(
		httpx.WithAdapter(ginAdapter),
		httpx.WithLogger(slogLogger),
		httpx.WithPrintRoutes(true),
	)

	// 注册端点
	_ = server.Register(userEndpoint)

	// 注册带前缀的端点
	_ = server.RegisterWithPrefix("/api/v1", userEndpoint)

	// 打印路由信息
	fmt.Println("\n=== Registered Routes ===")
	fmt.Printf("Total routes: %d\n\n", server.RouteCount())

	// 按方法分组打印
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		routes := server.GetRoutesByMethod(method)
		if len(routes) > 0 {
			fmt.Printf("%s:\n", method)
			for _, route := range routes {
				fmt.Printf("  %-30s -> %s\n", route.Path, route.HandlerName)
			}
			fmt.Println()
		}
	}

	// 打印 OpenAPI 信息
	if server.HasHuma() {
		fmt.Println("=== OpenAPI Documentation ===")
		fmt.Printf("OpenAPI JSON: http://localhost:8080/openapi.json\n")
		fmt.Printf("Swagger UI:   http://localhost:8080/docs\n")
		fmt.Println()
	}

	fmt.Println("Server starting on :8080")
	fmt.Println("Adapter: Gin")

	// 启动服务器
	err = server.ListenAndServe(":8080")
	if err != nil {
		panic(err)
	}
}

// yourCustomMiddleware 示例自定义中间件
// 你可以使用任何 Gin 生态的中间件，如：
// - github.com/gin-contrib/cors
// - github.com/gin-contrib/jwt
// - github.com/gin-contrib/gzip
func yourCustomMiddleware() ginFramework.HandlerFunc {
	return func(c *ginFramework.Context) {
		// 你的逻辑
		c.Next()
	}
}
