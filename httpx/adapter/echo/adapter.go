//go:build !no_echo

package echo

import (
	"log/slog"
	"net/http"

	"github.com/DaiYuANg/toolkit4go/httpx/adapter"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humaecho"
	"github.com/labstack/echo/v4"
)

// Adapter Echo 框架适配器
//
// 使用方式：
// 1. 创建适配器：echoAdapter := echo.New()
// 2. 注册 Echo 原生中间件：adapter.Engine().Use(yourMiddleware...)
// 3. 创建 httpx server 并注册路由
type Adapter struct {
	engine  *echo.Echo
	group   *echo.Group
	logger  *slog.Logger
	huma    huma.API
	humaCfg adapter.HumaOptions
}

// New 创建 Echo 适配器
func New(engine ...*echo.Echo) *Adapter {
	var eng *echo.Echo
	if len(engine) > 0 {
		eng = engine[0]
	} else {
		eng = echo.New()
	}

	return &Adapter{
		engine: eng,
		group:  nil,
		logger: slog.Default(),
	}
}

// WithHuma 启用 Huma OpenAPI 文档
func (a *Adapter) WithHuma(opts adapter.HumaOptions) *Adapter {
	a.humaCfg = opts
	a.huma = humaecho.New(a.engine, huma.DefaultConfig(opts.Title, opts.Version))
	return a
}

// WithLogger 设置日志记录器
func (a *Adapter) WithLogger(logger *slog.Logger) *Adapter {
	a.logger = logger
	return a
}

// Name 返回适配器名称
func (a *Adapter) Name() string {
	return "echo"
}

// Handle 注册业务处理函数
func (a *Adapter) Handle(method, path string, handler adapter.HandlerFunc) {
	if a.group != nil {
		a.group.Add(method, path, a.echoHandler(handler))
	} else {
		a.engine.Add(method, path, a.echoHandler(handler))
	}
}

// Group 创建路由组
func (a *Adapter) Group(prefix string) adapter.Adapter {
	var g *echo.Group
	if a.group != nil {
		g = a.group.Group(prefix)
	} else {
		g = a.engine.Group(prefix)
	}

	return &Adapter{
		engine:  a.engine,
		group:   g,
		logger:  a.logger,
		huma:    a.huma,
		humaCfg: a.humaCfg,
	}
}

// ServeHTTP 实现 http.Handler 接口
func (a *Adapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.engine.ServeHTTP(w, r)
}

// Engine 返回 Echo 引擎
// 通过此方法可以直接使用 Echo 的中间件生态
// 例如：adapter.Engine().Use(yourMiddleware...)
func (a *Adapter) Engine() *echo.Echo {
	return a.engine
}

// echoHandler 包装处理函数为 Echo 格式
func (a *Adapter) echoHandler(handler adapter.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		if err := handler(req.Context(), c.Response(), req); err != nil {
			a.logger.Error("Handler error",
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.String("error", err.Error()),
			)
			return err
		}
		return nil
	}
}

// HumaAPI 返回 Huma API
func (a *Adapter) HumaAPI() huma.API {
	return a.huma
}

// HasHuma 检查是否启用了 Huma
func (a *Adapter) HasHuma() bool {
	return a.huma != nil
}
