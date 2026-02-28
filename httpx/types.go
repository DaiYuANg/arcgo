package httpx

import (
	"context"
	"net/http"

	"github.com/DaiYuANg/toolkit4go/httpx/adapter"
)

// HTTPMethod HTTP 方法常量
const (
	MethodGet     = http.MethodGet
	MethodPost    = http.MethodPost
	MethodPut     = http.MethodPut
	MethodDelete  = http.MethodDelete
	MethodPatch   = http.MethodPatch
	MethodHead    = http.MethodHead
	MethodOptions = http.MethodOptions
)

// RouteInfo 路由信息
type RouteInfo struct {
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	HandlerName string   `json:"handler_name"`
	Comment     string   `json:"comment,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// String 返回路由的字符串表示
func (r RouteInfo) String() string {
	return r.Method + " " + r.Path + " -> " + r.HandlerName
}

// Endpoint HTTP 端点接口
type Endpoint interface {
	Routes() []RouteInfo
}

// HandlerFunc 通用处理函数签名
type HandlerFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request) error

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

// ToAdapterHumaOptions 将 httpx.HumaOptions 转换为 adapter.HumaOptions
func ToAdapterHumaOptions(opts HumaOptions) adapter.HumaOptions {
	return adapter.HumaOptions(opts)
}
