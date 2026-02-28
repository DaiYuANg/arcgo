//go:build !no_gin

package gin

import (
	"log/slog"
	"net/http"

	"github.com/DaiYuANg/toolkit4go/httpx/adapter"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
)

// Adapter Gin 框架适配器
//
// 使用方式：
// 1. 创建适配器：ginAdapter := gin.New()
// 2. 注册 Gin 原生中间件：ginAdapter.Engine().Use(gin.Logger(), yourMiddleware...)
// 3. 创建 httpx server 并注册路由
type Adapter struct {
	engine  *gin.Engine
	group   *gin.RouterGroup
	logger  *slog.Logger
	huma    huma.API
	humaCfg adapter.HumaOptions
}

// New 创建 Gin 适配器
func New(engine ...*gin.Engine) *Adapter {
	var eng *gin.Engine
	if len(engine) > 0 {
		eng = engine[0]
	} else {
		eng = gin.New()
	}

	return &Adapter{
		engine: eng,
		group:  &eng.RouterGroup,
		logger: slog.Default(),
	}
}

// WithHuma 启用 Huma OpenAPI 文档
func (a *Adapter) WithHuma(opts adapter.HumaOptions) *Adapter {
	a.humaCfg = opts
	a.huma = humagin.New(a.engine, huma.DefaultConfig(opts.Title, opts.Version))
	return a
}

// WithLogger 设置日志记录器
func (a *Adapter) WithLogger(logger *slog.Logger) *Adapter {
	a.logger = logger
	return a
}

// Name 返回适配器名称
func (a *Adapter) Name() string {
	return "gin"
}

// Handle 注册业务处理函数
func (a *Adapter) Handle(method, path string, handler adapter.HandlerFunc) {
	a.group.Handle(method, path, a.wrapHandler(handler))
}

// Group 创建路由组
func (a *Adapter) Group(prefix string) adapter.Adapter {
	return &Adapter{
		engine:  a.engine,
		group:   a.group.Group(prefix),
		logger:  a.logger,
		huma:    a.huma,
		humaCfg: a.humaCfg,
	}
}

// ServeHTTP 实现 http.Handler 接口
func (a *Adapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.engine.ServeHTTP(w, r)
}

// Engine 返回 Gin 引擎
// 通过此方法可以直接使用 Gin 的中间件生态
// 例如：adapter.Engine().Use(gin.Logger(), yourMiddleware...)
func (a *Adapter) Engine() *gin.Engine {
	return a.engine
}

// wrapHandler 包装处理函数为 Gin 格式
func (a *Adapter) wrapHandler(handler adapter.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := handler(c.Request.Context(), c.Writer, c.Request); err != nil {
			a.logger.Error("Handler error",
				slog.String("method", c.Request.Method),
				slog.String("path", c.Request.URL.Path),
				slog.String("error", err.Error()),
			)
			c.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
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
