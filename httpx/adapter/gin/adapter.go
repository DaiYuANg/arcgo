//go:build !no_gin

package gin

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/DaiYuANg/arcgo/httpx/adapter"
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
	cfg := huma.DefaultConfig(opts.Title, opts.Version)
	cfg.OpenAPI.Info.Description = opts.Description
	a.huma = humagin.New(a.engine, cfg)
	return a
}

// EnableHuma 启用 Huma OpenAPI 文档
func (a *Adapter) EnableHuma(opts adapter.HumaOptions) {
	a.WithHuma(opts)
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

// Listen 启动 Gin 服务。
func (a *Adapter) Listen(addr string) error {
	return a.engine.Run(addr)
}

// ListenContext 启动 Gin 服务并在 ctx 结束时优雅关闭。
func (a *Adapter) ListenContext(ctx context.Context, addr string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	server := &http.Server{
		Addr:    addr,
		Handler: a.engine,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		err := <-errCh
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

// wrapHandler 包装处理函数为 Gin 格式
func (a *Adapter) wrapHandler(handler adapter.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		params := make(map[string]string, len(c.Params))
		for _, p := range c.Params {
			params[p.Key] = p.Value
		}

		req := c.Request.WithContext(adapter.WithRouteParams(c.Request.Context(), params))

		if err := handler(req.Context(), c.Writer, req); err != nil {
			a.logger.Error("Handler error",
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
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
