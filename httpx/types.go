package httpx

import (
	"context"
	"net/http"

	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/danielgtaylor/huma/v2"
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

// TypedHandler 泛型强类型路由处理函数
// 核心签名与 huma.Register/huma.Post 保持一致。
type TypedHandler[I, O any] func(ctx context.Context, input *I) (*O, error)

// RequestBinder 自定义请求绑定接口。
//
// 当输入结构实现该接口时，httpx 将跳过默认反射绑定逻辑，
// 直接调用该方法完成请求解析。
type RequestBinder interface {
	BindRequest(r *http.Request) error
}

// RequestBodyBinder 自定义请求体绑定接口。
//
// 当输入结构实现该接口时，httpx 将使用该方法绑定请求体，
// 并继续执行默认参数绑定（或 RequestParamsBinder）。
type RequestBodyBinder interface {
	BindRequestBody(r *http.Request) error
}

// RequestParamsBinder 自定义路径/查询/Header/Cookie 参数绑定接口。
//
// 当输入结构实现该接口时，httpx 将使用该方法绑定参数，
// 并跳过默认反射参数绑定逻辑。
type RequestParamsBinder interface {
	BindRequestParams(r *http.Request) error
}

// OperationOption 路由元数据选项（可直接传 huma.OperationTags 等）。
type OperationOption func(*huma.Operation)

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
