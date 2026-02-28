// Package options 提供统一的配置选项模式
package options

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/DaiYuANg/toolkit4go/httpx"
	"github.com/DaiYuANg/toolkit4go/httpx/adapter"
)

// ServerOptions Server 配置选项集合
//
// 注意：中间件应该直接使用各框架的原生方式注册
// 例如：adapter.Engine().Use(yourMiddleware...)
type ServerOptions struct {
	Adapter            adapter.Adapter
	Logger             *slog.Logger
	BasePath           string
	PrintRoutes        bool
	HumaEnabled        bool
	HumaTitle          string
	HumaVersion        string
	HumaDescription    string
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	IdleTimeout        time.Duration
	MaxHeaderBytes     int
	EnablePanicRecover bool
	EnableAccessLog    bool
}

// DefaultServerOptions 默认 Server 配置
func DefaultServerOptions() *ServerOptions {
	return &ServerOptions{
		Logger:             slog.Default(),
		PrintRoutes:        false,
		HumaEnabled:        false,
		HumaTitle:          "My API",
		HumaVersion:        "1.0.0",
		HumaDescription:    "API Documentation",
		ReadTimeout:        15 * time.Second,
		WriteTimeout:       15 * time.Second,
		IdleTimeout:        60 * time.Second,
		MaxHeaderBytes:     1 << 20, // 1MB
		EnablePanicRecover: true,
		EnableAccessLog:    true,
	}
}

// ServerOption Server 配置选项函数
type ServerOption func(*ServerOptions)

// Compose 将多个选项组合为一个
func Compose(opts ...ServerOption) ServerOption {
	return func(o *ServerOptions) {
		for _, opt := range opts {
			opt(o)
		}
	}
}

// WithAdapter 设置适配器
func WithAdapter(adapter adapter.Adapter) ServerOption {
	return func(o *ServerOptions) {
		o.Adapter = adapter
	}
}

// WithLogger 设置日志记录器
func WithLogger(logger *slog.Logger) ServerOption {
	return func(o *ServerOptions) {
		o.Logger = logger
	}
}

// WithBasePath 设置基础路径
func WithBasePath(path string) ServerOption {
	return func(o *ServerOptions) {
		o.BasePath = path
	}
}

// WithPrintRoutes 设置是否打印路由
func WithPrintRoutes(enabled bool) ServerOption {
	return func(o *ServerOptions) {
		o.PrintRoutes = enabled
	}
}

// WithHuma 启用 Huma OpenAPI 文档
func WithHuma(enabled bool, title, version, description string) ServerOption {
	return func(o *ServerOptions) {
		o.HumaEnabled = enabled
		o.HumaTitle = title
		o.HumaVersion = version
		o.HumaDescription = description
	}
}

// WithTimeouts 设置服务器超时配置
func WithTimeouts(read, write, idle time.Duration) ServerOption {
	return func(o *ServerOptions) {
		o.ReadTimeout = read
		o.WriteTimeout = write
		o.IdleTimeout = idle
	}
}

// WithReadTimeout 设置读取超时
func WithReadTimeout(timeout time.Duration) ServerOption {
	return func(o *ServerOptions) {
		o.ReadTimeout = timeout
	}
}

// WithWriteTimeout 设置写入超时
func WithWriteTimeout(timeout time.Duration) ServerOption {
	return func(o *ServerOptions) {
		o.WriteTimeout = timeout
	}
}

// WithIdleTimeout 设置空闲超时
func WithIdleTimeout(timeout time.Duration) ServerOption {
	return func(o *ServerOptions) {
		o.IdleTimeout = timeout
	}
}

// WithMaxHeaderBytes 设置最大请求头大小
func WithMaxHeaderBytes(bytes int) ServerOption {
	return func(o *ServerOptions) {
		o.MaxHeaderBytes = bytes
	}
}

// WithPanicRecover 设置是否启用 panic 恢复
func WithPanicRecover(enabled bool) ServerOption {
	return func(o *ServerOptions) {
		o.EnablePanicRecover = enabled
	}
}

// WithAccessLog 设置是否启用访问日志
func WithAccessLog(enabled bool) ServerOption {
	return func(o *ServerOptions) {
		o.EnableAccessLog = enabled
	}
}

// Build 构建 Server 配置
func (o *ServerOptions) Build() []httpx.ServerOption {
	opts := []httpx.ServerOption{
		httpx.WithLogger(o.Logger),
		httpx.WithPrintRoutes(o.PrintRoutes),
	}

	if o.Adapter != nil {
		opts = append(opts, httpx.WithAdapter(o.Adapter))
	}

	if o.BasePath != "" {
		opts = append(opts, httpx.WithBasePath(o.BasePath))
	}

	if o.HumaEnabled {
		opts = append(opts, httpx.WithHuma(httpx.HumaOptions{
			Enabled:     true,
			Title:       o.HumaTitle,
			Version:     o.HumaVersion,
			Description: o.HumaDescription,
		}))
	}

	return opts
}

// HTTPClientOptions HTTP 客户端配置
type HTTPClientOptions struct {
	Timeout   time.Duration
	Transport http.RoundTripper
	Jar       http.CookieJar
}

// DefaultHTTPClientOptions 默认 HTTP 客户端配置
func DefaultHTTPClientOptions() *HTTPClientOptions {
	return &HTTPClientOptions{
		Timeout: 30 * time.Second,
	}
}

// HTTPClientOption HTTP 客户端配置选项
type HTTPClientOption func(*HTTPClientOptions)

// WithHTTPTimeout 设置客户端超时
func WithHTTPTimeout(timeout time.Duration) HTTPClientOption {
	return func(o *HTTPClientOptions) {
		o.Timeout = timeout
	}
}

// WithHTTPTransport 设置客户端传输层
func WithHTTPTransport(transport http.RoundTripper) HTTPClientOption {
	return func(o *HTTPClientOptions) {
		o.Transport = transport
	}
}

// WithHTTPCookieJar 设置 Cookie Jar
func WithHTTPCookieJar(jar http.CookieJar) HTTPClientOption {
	return func(o *HTTPClientOptions) {
		o.Jar = jar
	}
}

// Build 构建 HTTP 客户端
func (o *HTTPClientOptions) Build() *http.Client {
	return &http.Client{
		Timeout:   o.Timeout,
		Transport: o.Transport,
		Jar:       o.Jar,
	}
}

// ContextOptions Context 配置选项
type ContextOptions struct {
	Timeout       time.Duration
	Deadline      time.Time
	ValueKeys     map[string]interface{}
	CancelOnPanic bool
}

// ContextOption Context 配置选项函数
type ContextOption func(*ContextOptions)

// WithContextTimeout 设置 Context 超时
func WithContextTimeout(timeout time.Duration) ContextOption {
	return func(o *ContextOptions) {
		o.Timeout = timeout
	}
}

// WithContextDeadline 设置 Context 截止时间
func WithContextDeadline(deadline time.Time) ContextOption {
	return func(o *ContextOptions) {
		o.Deadline = deadline
	}
}

// WithContextValue 设置 Context 值
func WithContextValue(key string, value interface{}) ContextOption {
	return func(o *ContextOptions) {
		if o.ValueKeys == nil {
			o.ValueKeys = make(map[string]interface{})
		}
		o.ValueKeys[key] = value
	}
}

// WithContextCancelOnPanic 设置是否在 panic 时取消 context
func WithContextCancelOnPanic(enabled bool) ContextOption {
	return func(o *ContextOptions) {
		o.CancelOnPanic = enabled
	}
}

// Build 构建 Context
func (o *ContextOptions) Build() (context.Context, context.CancelFunc) {
	var ctx context.Context
	var cancel context.CancelFunc

	if o.Deadline.IsZero() && o.Timeout == 0 {
		ctx = context.Background()
	} else if !o.Deadline.IsZero() {
		ctx, cancel = context.WithDeadline(context.Background(), o.Deadline)
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), o.Timeout)
	}

	for k, v := range o.ValueKeys {
		ctx = context.WithValue(ctx, k, v)
	}

	return ctx, cancel
}

// WithContextValue 设置 Context 值（辅助函数）
func WithContextValueOpt(o *ContextOptions, key string, value interface{}) *ContextOptions {
	if o.ValueKeys == nil {
		o.ValueKeys = make(map[string]interface{})
	}
	o.ValueKeys[key] = value
	return o
}
