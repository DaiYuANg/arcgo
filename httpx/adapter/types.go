// Package adapter 提供 HTTP 框架适配器的通用接口
package adapter

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// HandlerFunc 通用处理函数签名
type HandlerFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request) error

// MiddlewareFunc 中间件函数签名（通用）
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

type routeParamsCtxKey struct{}

// Adapter HTTP 框架适配器接口
// 核心设计：
// 1. 提供底层框架的原生访问方法（如 Engine()、App()），方便直接使用框架生态
// 2. Handle 和 Group 用于注册业务路由（由 httpx 统一管理）
// 3. 中间件应该直接使用框架原生的方式注册
type Adapter interface {
	// Name 返回适配器名称
	Name() string

	// Handle 注册业务处理函数（由 httpx 调用）
	Handle(method, path string, handler HandlerFunc)

	// Group 创建路由组（由 httpx 调用）
	Group(prefix string) Adapter

	// ServeHTTP 实现 http.Handler 接口
	ServeHTTP(w http.ResponseWriter, r *http.Request)

	// EnableHuma 启用或刷新 Huma OpenAPI 能力
	EnableHuma(opts HumaOptions)

	// HumaAPI 返回 Huma API 对象（未启用时为 nil）
	HumaAPI() huma.API

	// HasHuma 标记当前适配器是否启用了 Huma
	HasHuma() bool
}

// ListenableAdapter 适配器启动能力（可选实现）。
type ListenableAdapter interface {
	Listen(addr string) error
}

// ContextListenableAdapter 适配器启动能力（支持 context，可选实现）。
type ContextListenableAdapter interface {
	ListenContext(ctx context.Context, addr string) error
}

// GinLikeAdapter Gin 风格的适配器接口（可选实现）
type GinLikeAdapter interface {
	Adapter
	// Engine 返回底层 Gin 引擎，方便直接使用 Gin 的中间件生态
	Engine() interface{}
}

// FiberLikeAdapter Fiber 风格的适配器接口（可选实现）
type FiberLikeAdapter interface {
	Adapter
	// App 返回底层 Fiber 应用，方便直接使用 Fiber 的中间件生态
	App() interface{}
}

// EchoLikeAdapter Echo 风格的适配器接口（可选实现）
type EchoLikeAdapter interface {
	Adapter
	// Engine 返回底层 Echo 引擎，方便直接使用 Echo 的中间件生态
	Engine() interface{}
}

// HumaOptions Huma OpenAPI 配置选项
type HumaOptions struct {
	// Enabled 是否启用 OpenAPI 文档
	Enabled bool
	// Title API 标题
	Title string
	// Version API 版本
	Version string
	// Description API 描述
	Description string
	// DocsPath 文档路径（默认 /docs）
	DocsPath string
	// OpenAPIPath OpenAPI JSON 路径（默认 /openapi.json）
	OpenAPIPath string
}

// DefaultHumaOptions 默认 Huma 配置
func DefaultHumaOptions() HumaOptions {
	return HumaOptions{
		Enabled:     false,
		Title:       "My API",
		Version:     "1.0.0",
		Description: "API Documentation",
		DocsPath:    "/docs",
		OpenAPIPath: "/openapi.json",
	}
}

// WithRouteParams 将路径参数写入 context，供上层统一读取。
func WithRouteParams(ctx context.Context, params map[string]string) context.Context {
	if len(params) == 0 {
		return ctx
	}
	return context.WithValue(ctx, routeParamsCtxKey{}, params)
}

// RouteParam 从 context 中读取路径参数。
func RouteParam(ctx context.Context, name string) string {
	if ctx == nil || name == "" {
		return ""
	}

	params, ok := ctx.Value(routeParamsCtxKey{}).(map[string]string)
	if !ok {
		return ""
	}
	return params[name]
}
